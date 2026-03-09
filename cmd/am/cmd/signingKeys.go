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

var signingKeysCmd = &cobra.Command{
	Use:   "signing-keys",
	Short: "Get current signing keys from the access manager",
	Long:  `Retrieves the current signing keys from the access manager and displays them in JSON format.`,
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := request(context.Background(), "/signingkeys", http.MethodGet)
		if err != nil {
			if _, err = fmt.Fprintf(os.Stderr, "Error fetching signing keys: %v\n", err); err != nil {
				panic(err)
			}
			return
		}

		var x struct {
			Keys map[int64]string
		}
		if err := json.Unmarshal(resp, &x); err != nil {
			if _, err = fmt.Fprintf(os.Stderr, "Error parsing signing keys response: %v\n%s\n", err, resp); err != nil {
				panic(err)
			}
			return
		}

		var r struct {
			Keys map[int64]string `json:"keys"`
		}
		r.Keys = make(map[int64]string, len(x.Keys))
		for exp, bytes := range x.Keys {
			r.Keys[exp] = string(bytes)
		}
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			if _, err = fmt.Fprintf(os.Stderr, "Error formatting signing keys: %v\n", err); err != nil {
				panic(err)
			}
		}
		fmt.Println(string(data))
	},
}
