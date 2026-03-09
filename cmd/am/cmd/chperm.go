/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hpe/access-manager/internal/services/common"

	"github.com/spf13/cobra"
)

// chpermCmd represents the chperm command
var chpermCmd = &cobra.Command{
	Use:   "chperm [-r unique-to-delete] path-to-change < new-or-changed-permissions.json",
	Short: "Change permissions on a path",
	Long: `
The "chperm" command changes the permissions on a particular
path. The new permissions are expressed as JSON and are taken
as a command line argument or, more commonly, from a file or
from standard input. The form of these new permissions should
be the same as the form produced by "ls -ld" except that only
the "aces" component is significant to this command. Further, 
the version number must either be 0 for new permissions, -1 for
unconditional creation or update or match the current version 
number on the permissions for the specified path. The easiest way 
to ensure this is to get the permissions using "ls -ld", change 
the "aces" field as desired and send the same structure back using 
"chperm". Using the version allows an atomic read/modify/update 
that will fail if somebody (or something) has made a conflicting 
update.

When adding a new permission, simply omit the version and unique
fields. This sets them to zero and  implies creation of a new 
permission. When the unique is zero (or missing) a new random 
value will be assigned.

You must have UseRole and View permission on all roles in a 
permission that you add, modify or remove. You must also have 
Admin and View permission on the object whose permissions you are
changing.

Deletion of permissions is done by using the --remove (-r for short)
option followed by a comma separated list of unique values for
the permissions you want to delete.

As an example, to modify one permission, add another and delete
a third you could use the command below command. 

$ am chperm -r 6534743860862523031 --edit - <<EOF
{
   "Details": {
      "aces": [
         {
            "op": "UseRole",
            "unique": "406519965772129914",
            "version": "1",
            "permissions": [{"roles": ["am://role/hpe/bu1/bu1-admin", "am://role/operator-admin"]}]
         },
         {
            "op": "VIEW",
            "permissions": [{"roles": ["am://role/hpe-user"]}]
         }
      ],
   },
}
EOF
$

The format of permission edits is fairly flexible and allows the input
to be highly abbreviated. For instance, if you only wanted to create the permission, 
new permission from the example above, the following would suffice:

$ am chperm -r 6534743860862523031 --edit - <<EOF
{
   "op": "VIEW",
   "permissions": [{"roles": ["am://role/hpe-user"]}]
}
EOF
$
`,
	Args: validAmPath(common.StandardPrefix),
	RunE: func(_ *cobra.Command, args []string) error {
		return chperm(args[0])
	},
}

var RemovePerms string
var EditPerms string

func chperm(path string) error {
	if EditPerms != "" {
		var perms []byte
		var err error
		if strings.HasPrefix(EditPerms, "@") {
			perms, err = os.ReadFile(EditPerms[1:])
			if err != nil {
				return err
			}
			return editPerms(path, string(perms))
		} else if EditPerms == "-" {
			perms, err = io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			err = editPerms(path, string(perms))
			if err != nil {
				return err
			}
		} else {
			err := editPerms(path, EditPerms)
			if err != nil {
				return err
			}
		}
	}
	if RemovePerms != "" {
		return removePerms(path, RemovePerms)
	}
	return nil
}

func removePerms(path string, perms string) error {
	if !common.StandardPattern.MatchString(path) {
		return fmt.Errorf("invalid path: %s", path)
	}
	uniquePattern := regexp.MustCompile(`^\d+$`)
	for _, perm := range strings.Split(perms, ",") {
		if !uniquePattern.MatchString(perm) {
			return fmt.Errorf("invalid format for unique ID: %s", perm)
		}
		uniqueId, err := strconv.Atoi(perm)
		response, err := request(
			context.Background(),
			fmt.Sprintf("/annotate/%s/ace/%d", strings.Replace(path, "am://", "am/", 1), uniqueId),
			http.MethodDelete,
		)
		if err != nil {
			return err
		}
		details := struct {
			Error Status `json:"error"`
		}{}
		err = json.Unmarshal(response, &details)
		if err != nil {
			return fmt.Errorf(`error parsing response: %w`, err)
		}
		if details.Error.Error != 0 {
			return fmt.Errorf("%s: %s", path, details.Error.Message)
		}
	}
	return nil
}

func editPerms(path string, perms string) error {
	aces, err2 := parseAces(perms)
	if err2 != nil {
		return err2
	}
	if aces == nil {
		return fmt.Errorf("invalid permission format: %s", perms)
	}
	for _, ace := range aces {
		jsonAce, err := json.Marshal(struct {
			Tag  string
			Meta *ACE
		}{
			Tag:  "ace",
			Meta: ace,
		})
		if err != nil {
			return err
		}
		response, err := request(
			context.Background(),
			fmt.Sprintf("/annotate/%s/ace", strings.Replace(path, "am://", "am/", 1)),
			http.MethodPost,
			"annotation", string(jsonAce),
		)
		details := struct {
			Error Status `json:"error"`
		}{}
		err = json.Unmarshal(response, &details)
		if err != nil {
			return fmt.Errorf(`error parsing response: %w`, err)
		}
		if details.Error.Error != 0 {
			return fmt.Errorf("%s: %s", path, details.Error.Message)
		}
	}
	return nil
}

func parseAces(perms string) ([]*ACE, error) {
	// this structure has all the legal wrapped forms in one package
	var v1 struct {
		Details struct {
			aces []*ACE
		}
		aces []*ACE
	}
	err := json.Unmarshal([]byte(perms), &v1)
	if err == nil {
		if v1.Details.aces != nil {
			return v1.Details.aces, nil
		}
		if v1.aces != nil {
			return v1.aces, nil
		}
	}

	// if that didn't work, let's try unwrapped form
	var v2 ACE
	err = json.Unmarshal([]byte(perms), &v2)
	if err != nil {
		return nil, err
	}
	if v2.Roles == nil {
		return nil, fmt.Errorf("invalid permission format: %s", perms)
	}
	return []*ACE{&v2}, nil
}
