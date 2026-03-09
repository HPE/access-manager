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

	access_manager "github.com/hpe/access-manager/internal/services/access-manager"
	"github.com/hpe/access-manager/internal/services/common"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls [path-to-list]",
	Short: "Show metadata for a particular path",
	Args:  validAmPath(common.StandardPrefix),
	Long: `
The "ls" command lets you show the metadata for a path. 
Options allow you to see more or less detail and to show
direct children or to recursively descend through the 
metadata tree until leaf nodes are encountered such as
datasets, users, workloads or roles.`,
	Run: func(_ *cobra.Command, args []string) {
		ls(MoreDetails, Direct, Recursive, args[0])
	},
}
var (
	Direct      bool
	MoreDetails bool
	Recursive   bool
)

func getTree(baseUrl, path, id string, depth int) (*NodeTree, error) {
	params := url.Values{}
	params.Add("caller_id", id)
	params.Add("include_children", "true")
	u, _ := url.ParseRequestURI(baseUrl)
	u.Path = strings.Replace(path, "am://", "am/", 1)
	u.RawQuery = params.Encode()
	if Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "listing %s, callerId=%s, url=%s\n",
			path, id, u.String())
	}
	request, err := http.NewRequestWithContext(context.Background(), "GET", u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf(`cannot create request for "%s": %w`, u.String(), err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf(`cannot access "%s": %w`, u.String(), err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`error reading response from access manager (%s)`, response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(`error reading response from %s: %w`, u.String(), err)
	}

	if Verbose {
		_, _ = fmt.Fprintf(os.Stderr, `response="%s"`, body)
	}
	var details access_manager.GetDetailsResponse
	err = protojson.Unmarshal(body, &details)
	if err != nil {
		return nil, fmt.Errorf(`error parsing response: %w`, err)
	}
	if details.Error.Error != 0 {
		return nil, fmt.Errorf("error getting data: %s", details.Error.Message)
	}
	r := NodeTree{Details: details.Details}
	if depth > 0 {
		for _, child := range details.Children {
			tree, err := getTree(baseUrl, child, id, depth-1)
			if err != nil {
				return nil, err
			}
			if tree != nil {
				r.Children = append(r.Children, tree)
			}
		}
	}
	return &r, nil
}

func sparseTree(tree *NodeTree) *SparseTree {
	if tree == nil {
		return nil
	} else {
		r := SparseTree{
			tree.Details.Path,
			make([]*SparseTree, len(tree.Children)),
		}
		for i, child := range tree.Children {
			r.Children[i] = sparseTree(child)
		}
		return &r
	}
}

func ls(getDetails, noChildren, recursive bool, path string) {
	depth := 1
	if noChildren {
		depth = 0
	} else if recursive {
		// this is just a big number ... no system is likely to be this deep, but we have to draw the line somewhere
		depth = 10000
	}
	tree, err := getTree(baseURL, path, callerID, depth)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get details: %s\n", err.Error())
	}
	if getDetails {
		out := common.CleanJson(tree)
		_, err = fmt.Printf("%s\n", out)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, `error writing to standard out': %s\n`, err.Error())
			return
		}
	} else {
		tmp := sparseTree(tree)
		out := common.CleanJson(tmp)
		_, err = fmt.Fprintf(os.Stdout, "%s\n", out)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, `error writing to standard out': %s\n`, err.Error())
			return
		}
	}
}

type SparseTree struct {
	Path     string        `json:"path"`
	Children []*SparseTree `json:"children,omitempty"`
}

type NodeTree struct {
	Details  *access_manager.NodeDetails
	Children []*NodeTree
}
type NodeDetails struct {
	Path           string        `json:"path"`
	Roles          []AppliedRole `json:"roles"`
	InheritedRoles []AppliedRole `json:"inheritedRoles"`
	Aces           []ACE         `json:"aces"`
	InheritedAces  []ACE         `json:"inheritedAces"`
	IsDirectory    bool          `json:"isDirectory"`
}

type AppliedRole struct {
	Version       int64  `json:"version,string"`
	Unique        int64  `json:"unique,string"`
	Role          string `json:"role"`
	EndTimeMillis int64  `json:"endTime,string"`
}

type ACE struct {
	Op            string     `json:"op"`
	Unique        int64      `json:"unique,string"`
	Local         bool       `json:"local"`
	Version       int64      `json:"version,string"`
	EndTimeMillis int64      `json:"endTimeMillis,int"`
	Roles         [][]string `json:"roles"`
}

type Status struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
	Global  string `json:"global"`
}
