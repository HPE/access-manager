/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var amCmd = &cobra.Command{
	Use:   "am",
	Short: "Explore and manipulate metadata in the Access Manager",
	Long: `
The "am" command lets you display metadata and directory trees
for the Access Manager. You can also modify that
data subject to whether you have permission to do so.

The command structure is inspired by Unix commands. Thus, "ls"
is used to show content and "ls -l" gives more details. The output
is, however, not in the line-oriented format of most Unix/Linux
commands but exclusively in JSON format. You can make this a bit
easier to read by pretty-printing it. One easy way to do this is
to use the built-in Python module "json.tool". For instance, this

    am ls -r  am://user/ | python3 -m json.tool

would show all users in a relatively concise form.
`,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if callerID == "" {
			return fmt.Errorf(
				"need user id. This can be supplied with AM_USER environment variable or with --user")
		}
		if baseURL == "" {
			return fmt.Errorf(
				`need URL for access manager which can be supplied via ACCESS_MANAGER_URL environment variable or by supplying --am_url=$URL`)
		}
		return nil
	},
}

var callerID string
var sshKey string
var baseURL string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the amCmd.
func Execute() {
	if strings.HasPrefix(callerID, "@") {
		c, err := os.ReadFile(strings.TrimSpace(callerID[1:]))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error reading user credential from %s: %v\n", callerID[1:], err)
			os.Exit(1)
		}
		callerID = string(c)
	}
	err := amCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var Verbose = false

func Init() {
	amCmd.PersistentFlags().StringVar(
		&callerID,
		"user",
		os.Getenv("AM_USER"),
		"Path name of caller (may also be set using AM_USER environment variable)",
	)
	amCmd.PersistentFlags().StringVar(
		&baseURL,
		"am_url",
		os.Getenv("ACCESS_MANAGER_URL"),
		"URL for access manager (may also be set using ACCESS_MANAGER_URL environment variable)",
	)
	amCmd.PersistentFlags().BoolVarP(
		&Verbose,
		"verbose",
		"v",
		false,
		"Print out internal details like URLs used",
	)

	amCmd.AddCommand(bootCmd)
	bootCmd.Flags().StringVarP(&sshKey, "key", "u", "", "SSH public key to set on the operator user")

	amCmd.AddCommand(annotateCmd)
	amCmd.AddCommand(delegateCmd)
	amCmd.AddCommand(vouchCmd)

	amCmd.AddCommand(chpermCmd)
	chpermCmd.Flags().StringVarP(&RemovePerms, "remove", "r", "", "A comma-separated list of unique IDs of permissions to remove")
	chpermCmd.Flags().StringVarP(&EditPerms, "edit", "e", "", "A JSON object specifying new and modified permissions. "+
		"Can also be @file or - to get input from a file or standard input")

	amCmd.AddCommand(chroleCmd)
	chroleCmd.Flags().StringVarP(&AddRoles, "add", "a", "", "A comma-separated list of roles to add")
	chroleCmd.Flags().StringVarP(&RemoveRoles, "remove", "r", "", "A comma-separated list of roles to remove")

	amCmd.AddCommand(lsCmd)

	lsCmd.Flags().BoolVarP(&Recursive, "recursive", "r", false, "Recursively show directory structure")
	lsCmd.Flags().BoolVarP(&Direct, "direct", "d", false, "Don't show any children at all")
	lsCmd.Flags().BoolVarP(&MoreDetails, "long", "l", false, "Show more detail")
	lsCmd.MarkFlagsMutuallyExclusive("direct", "recursive")

	amCmd.AddCommand(mkCmd)

	amCmd.AddCommand(mkDirCmd)

	amCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&Recursive, "recursive", "r", false, "Remove recursively")

	amCmd.AddCommand(signingKeysCmd)

	amCmd.AddCommand(validateCmd)
}

func validAmPath(kind string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}
		if strings.HasPrefix(args[0], kind) {
			return nil
		} else {
			return fmt.Errorf("path must start with %s", kind)
		}
	}
}

func request(ctx context.Context, apiPath string, method string, kv ...string) ([]byte, error) {
	u, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return nil, fmt.Errorf("can't happen: %w", err)
	}
	u.Path = apiPath
	params := url.Values{}
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("must have an even number of key/value pairs")
	}
	for i := 0; i < len(kv); i += 2 {
		params.Set(kv[i], kv[i+1])
	}

	params.Set("caller_id", callerID)
	u.RawQuery = params.Encode()
	if Verbose {
		fmt.Printf("URL: %s\n", u.String())
	}
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		u.String(),
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf(`cannot create request for "%s": %w`, u.String(), err)
	}
	if Verbose {
		fmt.Printf("request: %v\n", request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf(`cannot access "%s": %w`, u.String(), err)
	}
	if Verbose {
		fmt.Printf("response %v\n", response)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`unexpected status code "%d"`, response.StatusCode)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(`error reading response from %s: %w`, u.String(), err)
	}
	return responseBody, err
}
