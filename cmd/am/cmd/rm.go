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
	"os"
	"regexp"
	"strings"

	"github.com/hpe/access-manager/internal/services/common"
	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm [path-to-delete]",
	Short: "Removes metadata at a path, possibly recursively",
	Long: `
The "rm" command removes metadata entries. The normal behavior
is to remove the metadata for a single dataset, role, user or 
workload, but an entire directory tree can also be removed in 
a single step by using the recursive option.

Annotations can also be removed adding the unique identifier
for the annotation to the path. For example, to remove the
ssh public key for a user, the path might look like this:

am://user/yoyodyne/fred#ssh-pubkey-116645369962115

where the "ssh-pubkey-116645369962115" is the tag for the 
annotation followed by a "-" and the unique identifier for the
annotation.  Using the unique identity for the annotation this
way is the only way to remove a user annotation, but ACEs and 
applied roles can also be removed or modified using chperm or 
chrole commands.

You can find the unique identifier for an annotation by using the
"ls" command.
`,
	Args: validAmPath(common.StandardPrefix),
	RunE: func(_ *cobra.Command, args []string) error {
		return rm(args[0])
	},
}

var hashPattern = regexp.MustCompile(`(am://.+)#([[:alnum:]-]+)-([[:digit:]]+)$`)

func rm(path string) error {
	params := url.Values{}
	params.Add("caller_id", callerID)
	u, _ := url.ParseRequestURI(baseURL)
	bits := hashPattern.FindStringSubmatch(path)
	if bits != nil {
		params.Add("path", bits[1])
		params.Add("tag", bits[2])
		params.Add("unique", bits[3])
		u.Path = "annotate"
	} else {
		u.Path = strings.Replace(path, "am://", "am/", 1)
	}
	u.RawQuery = params.Encode()
	if Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "Remove path=%s, caller=%s, url=%s\n",
			path, callerID, u.String())
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf(`can't build request for %s: %w`, u.String(), err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf(`cannot access "%s": %w`, u.String(), err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(`error deleting data: %s`, response.Status)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf(`error reading response: %w`, err)
	}
	if Verbose {
		_, _ = fmt.Fprintf(os.Stderr, `response="%s"`, body)
	}
	details := struct {
		Error Status `json:"error"`
	}{}
	err = json.Unmarshal(body, &details)
	if err != nil {
		return fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return fmt.Errorf("error deleting data: %s", details.Error.Message)
	}
	return nil
}
