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
	"net/url"
	"strconv"
	"strings"

	"github.com/hpe/access-manager/internal/services/common"

	"github.com/spf13/cobra"
)

var mkCmd = &cobra.Command{
	Use:   "mk [path-for-object]",
	Short: "Creates a user, workload, role or dataset at a specified path",
	Long: `
The "mk" command creates an object at a specified path.

Once created, the next action depends on the kind of object
you created. 

For users or workloads, you will need to add 
one or more "VouchFor" permissions for a known identity 
plugin. Any plugin with this permission will be able to 
request a credential for this user. That plugin may look 
for an annotation on the user to use to authenticate the 
user or it may have independent information. Alternatively, 
you can add an "ssh-pubkey" annotation containing an ssh 
public key to use to authenticate the user. In that case, the
user would establish an ssh connection to the access manager
server using the user path as the ssh user name. If ssh
authentication succeeds, a credential will be returned
over the ssh connection.

For object stores, you will need to specify the physical path
and identity a credential agent that can generate access 
tokens for the data. For structured data systems like SQL
databases or for DAOS, you will need to specify the user attributes
that are exported to the system for access control.
`,
	Args: validAmPath(common.StandardPrefix),
	RunE: func(_ *cobra.Command, args []string) error {
		return mk(args[0], "object-path", callerID, false)
	},
}

func mk(path, kind, callerID string, asDirectory bool) error {
	params := url.Values{}
	params.Add("asDirectory", strconv.FormatBool(asDirectory))
	params.Add("caller_id", callerID)
	u, _ := url.ParseRequestURI(baseURL)
	u.Path = strings.Replace(path, "am://", "am/", 1)
	u.RawQuery = params.Encode()
	if Verbose {
		fmt.Printf("Creating object %s\n", u.String())
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		u.String(),
		http.NoBody,
	)
	if err != nil {
		return fmt.Errorf(`cannot create request for "%s": %w`, u.String(), err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf(`cannot access "%s": %w`, u.String(), err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf(`error reading response from %s: %w`, u.String(), err)
	}
	details := struct {
		Error Status `json:"error"`
	}{}
	err = json.Unmarshal(body, &details)
	if err != nil {
		return fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return fmt.Errorf("error creating %s: %s", kind, details.Error.Message)
	}

	fmt.Printf("Created %s\n", path)
	return nil
}
