/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	credential string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a credential with the access manager",
	Long:  `Sends a credential to the access manager for validation and displays the result.`,
	Run: func(cmd *cobra.Command, args []string) {
		if credential == "" {
			if _, err := fmt.Fprintln(os.Stderr, "Error: credential is required"); err != nil {
				panic(err)
			}

			if err := cmd.Help(); err != nil {
				panic(err)
			}
			return
		}

		resp, err := request(context.Background(), "/validate", http.MethodGet, "credential", credential)
		if err != nil {
			if _, err = fmt.Fprintf(os.Stderr, "Error validating credential: %v\n", err); err != nil {
				panic(err)
			}
			return
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp, &result); err != nil {
			if _, err = fmt.Fprintf(os.Stderr, "Error parsing validation response: %v\n%s\n", err, resp); err != nil {
				panic(err)
			}
			return
		}

		prettyResult, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			if _, err = fmt.Fprintf(os.Stderr, "Error formatting response: %v\n", err); err != nil {
				panic(err)
			}
			return
		}

		fmt.Println(string(prettyResult))
	},
}

func init() {
	validateCmd.Flags().StringVarP(&credential, "credential", "c", "", "Credential to validate")
	_ = validateCmd.MarkFlagRequired("credential")
}
