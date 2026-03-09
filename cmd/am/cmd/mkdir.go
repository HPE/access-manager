/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"

	"github.com/spf13/cobra"
)

var mkDirCmd = &cobra.Command{
	Use:   "mkdir [path-for-directory]",
	Short: "Creates a directory at a specified path",
	Long: `
The "mkdir" command creates a directory at a specified path.

Once you have created a directory at a specified path you can
add annotations such as roles or permissions to be inherited
by objects in this directory or sub-directories.,
`,
	Args: validAmPath(common.StandardPrefix),
	RunE: func(_ *cobra.Command, args []string) error {
		// force the path to have exactly one /
		fmt.Printf("Creating directory at %s\n", args[0])
		return mk(args[0], "object-path", callerID, true)
	},
}
