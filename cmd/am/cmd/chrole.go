/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"github.com/hpe/access-manager/internal/services/metadata"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// chpermCmd represents the chperm command
var chroleCmd = &cobra.Command{
	Use:   "chrole path-to-change role-to-change",
	Short: "Add or remove roles for a user or workload",
	Long: `
The "chrole" command adds or deletes roles for a user or
workload. Only paths under am://user or am://workload 
can have roles, so those are the only paths that are allowed. 
The desired change is expressed as comma-separated list of
roles as the value for the --add or --remove flags. 
The changes are made one at a time and if a change fails,
earlier changes are not rolled back. The changes are made
unconditionally (without paying attention to versions of
the AppliedRole metadata) which may cause some confusion 
if there are other changes to the same role at the same time.".  
`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return chrole(args[0], AddRoles, RemoveRoles)
	},
}

var AddRoles string
var RemoveRoles string

func chrole(path, rolesToAdd, rolesToRemove string) error {
	details, err := getTree(baseURL, path, callerID, 0)
	if err != nil {
		return err
	}
	allRoles := map[string]*metadata.AppliedRole{}
	for _, r := range details.Details.Roles {
		allRoles[r.Role] = r
	}

	if rolesToAdd != "" {
		for _, s := range strings.Split(rolesToAdd, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if !common.StandardPattern.MatchString(s) {
				return fmt.Errorf("invalid role %s", s)
			}
			_, ok := allRoles[s]
			if ok {
				return fmt.Errorf("duplicate role %s", s)
			}
			err := addRole(path, s)
			if err != nil {
				return err
			}
			fmt.Printf("Added role %s\n", s)
		}
	}

	if rolesToRemove != "" {
		for _, role := range strings.Split(rolesToRemove, ",") {
			if strings.TrimSpace(role) == "" {
				continue
			}
			if !common.StandardPattern.MatchString(role) {
				return fmt.Errorf("invalid role %s", role)
			}
			rx, ok := allRoles[role]
			if ok {
				err := removeRole(path, rx)
				if err != nil {
					return fmt.Errorf("error removing role %s: %w", role, err)
				}
				fmt.Printf("Removed role %s\n", role)
			}
		}
	}
	return nil
}

func addRole(path, role string) error {
	fmt.Println("Adding role")
	r := struct {
		Tag  string
		Meta struct {
			Version int64
			Role    string
		}
	}{
		Tag: "applied-role",
		Meta: struct {
			Version int64
			Role    string
		}{
			Version: -1,
			Role:    role,
		},
	}
	req, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}
	if Verbose {
		fmt.Printf("Request %s\n", string(req))
	}
	responseBody, err := request(context.Background(), "/annotate", http.MethodPost, "path", path, "annotation", string(req))
	return finishChange(path, responseBody, err)
}

func removeRole(path string, role *metadata.AppliedRole) error {
	r := struct {
		Tag  string
		Meta any
	}{
		Tag: "applied-role",
		Meta: struct {
			Version int64
			Role    string
		}{
			Version: -1,
			Role:    role.Role,
		},
	}
	req, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}
	responseBody, err := request(context.Background(), "/annotate", http.MethodDelete, "path", path, "annotation", string(req))
	return finishChange(path, responseBody, err)
}

func finishChange(path string, responseBody []byte, err error) error {
	if err != nil {
		return fmt.Errorf(`error reading response from %s: %w`, path, err)
	}
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
