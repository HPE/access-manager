/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hpe/access-manager/internal/services/common"
	"github.com/spf13/cobra"
)

// chpermCmd represents the chperm command
var vouchCmd = &cobra.Command{
	Use:   "vouch identity-path",
	Short: "Get credentials for a user or workload that we can vouch for",
	Long: `
Get a signed credential for a specified user for whom we have VouchFor permission.

For instance, in the "new_sample" metadata boot image, there is a workload
"am://workload/yoyodyne/id-plugin" who has VouchFor permission for all of
the users under "am://user/yoyodyne". We can get a credential for 
"am://user/yoyodyne/bob" using the "id-plugin" identity (which is not 
secured by default:

$ export AM_URL=am://workload/yoyodyne/id-plugin
$ ./am vouch am://user/yoyodyne/bob | tee bob_cred
$ export AM_URL=@bob_cred
$ ./am ls am://user/yoyodyne
`,
	Args: validAmPath(common.StandardPrefix),
	RunE: func(_ *cobra.Command, args []string) error {
		return vouch(args[0])
	},
}

func vouch(path string) error {
	response, err := request(
		context.Background(),
		"/credential",
		http.MethodGet,
		"path", path)
	if err != nil {
		return err
	}
	details := struct {
		Credential string `json:"credential"`
		Error      Status `json:"error"`
	}{}
	err = json.Unmarshal(response, &details)
	if err != nil {
		return fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return fmt.Errorf("%s: %s", path, details.Error.Message)
	}
	fmt.Printf("%s\n", details.Credential)
	return nil
}
