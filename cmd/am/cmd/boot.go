/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
)

var bootCmd = &cobra.Command{
	Use:   "boot bootstrap_file [--key public_key_file]",
	Short: "Loads a bootstrap file into the access manager, optionally sets the public key for the operator",
	Long: `
    The "boot" command initializes the access manager with a bootstrap file.

    This defines the operator ` + "`am: //user/the-operator`" + ` and the basic 
    roles as well as the permissions on the top-level directories. With the
    ` + "`--key`" + ` option, you can set an ssh public key on the operator
    user.

	Once created, the next action is typically to create organizations and 
    define the administrators of those organizations. Initially, those 
    administrators will typically use ssh authentication to connect to the access
    manager, but they will likely want to authorize authentication plugins
    for their organization. This will allow users to authenticate with
    the access manager using their organization credentials.

    This command will only succeed if the metadata store is empty. If you
    have an existing metadata store, you will need to delete all metadata
	before you can run this command. This can be done by stopping all etcd
	servers and deleting the data directory. You can also use the ` + "`etcdctl`" + `
	command to delete all keys in the metadata store. For example, if you
	are using the default etcd data directory, you can run the following
	command to delete all keys:

		etcdctl [... security options omitted ...] del --prefix /meta/
    
	`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return boot(args[0], sshKey)
	},
}

func boot(path, key string) error {
	if Verbose {
		fmt.Printf("Bootstrapping from %s\n", path)
	}
	body, err := request(context.Background(), "/bootstrap", http.MethodPost, "boot", path, "key", key)

	details := struct {
		Error Status `json:"error"`
	}{}
	err = json.Unmarshal(body, &details)
	if err != nil {
		return fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return fmt.Errorf(`error loading "%s": %s`, path, details.Error.Message)
	}

	fmt.Printf("Loaded %s\n", path)
	return nil
}
