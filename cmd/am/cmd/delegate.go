/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"github.com/spf13/cobra"
	"net/http"
)

// chpermCmd represents the chperm command
var delegateCmd = &cobra.Command{
	Use:   "delegate data-path",
	Short: "Get a delegation token relative to a specified dataset",
	Long: `
The delegate command gets a delegation credential for a particular
dataset. This will succeed if you have VIEW permission on the dataset.
The resulting delegation credential will have roles which are the 
intersection of the roles held by the requestor and the roles that 
the dataset specifies its "data-info" annotation.
`,
	Args: validAmPath(common.StandardPrefix),
	RunE: func(_ *cobra.Command, args []string) error {
		return delegate(args[0])
	},
}

func delegate(path string) error {
	if !common.StandardPattern.MatchString(path) {
		return fmt.Errorf("invalid path: %s", path)
	}
	responseBody, err := request(context.Background(), "/datasetcredential", http.MethodGet, "path", path)
	if err != nil {
		return fmt.Errorf(`error reading response from %s: %w`, path, err)
	}
	fmt.Printf("response: %s\n", responseBody)
	details := struct {
		Error Status `json:"error"`
	}{}
	err = json.Unmarshal(responseBody, &details)
	if err != nil {
		return fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return fmt.Errorf("%s: %s", path, details.Error.Message)
	}
	return nil
}
