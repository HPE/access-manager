/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"io"
	"regexp"
	"strings"
)

// A MetaTree is a useful way to pass around an instantiated tree
// but probably isn't what we want to use for anything but data
// transfer. The primary use is for injecting known data for tests
// or for bootstrapping the metadata with a minimal state.
type MetaTree struct {
	Path     string
	Meta     []*Annotation
	Children []*MetaTree
}

func (t *MetaTree) GetPath(key string) (*MetaTree, error) {
	after, found := strings.CutPrefix(key, common.StandardPrefix)
	if !found {
		return nil, fmt.Errorf("Path must start with %s, got %s", common.StandardPrefix, key)
	}
	return t.Get(strings.Split(after, "/")...)
}

func (t *MetaTree) Get(keys ...string) (*MetaTree, error) {
	accumulatedPath := "am:/"
	r := t
	for _, key := range keys {
		// test allows us to have am://a//b as a synonym for am://a/b
		if key == "" {
			continue
		}
		accumulatedPath = accumulatedPath + "/" + key
		var found *MetaTree = nil
		for _, child := range r.Children {
			if child.Path == accumulatedPath {
				found = child
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("%s not found in tree", accumulatedPath)
		}
		r = found
	}
	return r, nil
}

type TraversalOrder int

const (
	TOP_DOWN TraversalOrder = iota
	BOTTOM_UP
)

/*
Walk traverses a tree calling a function on each interior and leaf node. The
order of traversal is controlled (roughly) by the first argument. If the
function returns without error, then the traversal will continue, but if it
returns an error, the traversal will be aborted and tha error will be returned.
*/
func (t *MetaTree) Walk(direction TraversalOrder, f func(tx *MetaTree) error) error {
	if direction == BOTTOM_UP {
		for _, child := range t.Children {
			if err := child.Walk(direction, f); err != nil {
				return err
			}
		}
	}
	if err := f(t); err != nil {
		return err
	}
	if direction == TOP_DOWN {
		for _, child := range t.Children {
			if err := child.Walk(direction, f); err != nil {
				return err
			}
		}
	}
	return nil
}

//go:embed sample/new_sample.json
//go:embed sample/bootstrap.json
var f embed.FS

func getData(name string) ([]byte, error) {
	f, err := f.Open(name)
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// SampleMetaTree reads an embedded metadata tree from a file an embedded file
// called `sample/<base>.json`. The base is the name of the file without the
// `.json` extension. The file should be in the `sample` directory and should
// contain a valid JSON representation of a MetaTree. The function returns an
// error if the file cannot be found or if the JSON is invalid.
//
// If the base is not valid, an error is returned.
func SampleMetaTree(base string) (*MetaTree, error) {
	if !regexp.MustCompile("[a-zA-Z]+").MatchString(base) {
		return nil, fmt.Errorf("invalid metadata file: %s", base)
	}
	all, err := getData("sample/" + base + ".json")
	if err != nil {
		valid := []string{}
		dir, _ := f.ReadDir("sample")
		for _, file := range dir {
			valid = append(
				valid,
				strings.TrimSuffix(
					strings.TrimPrefix(file.Name(), "sample/"),
					".json",
				),
			)
		}
		return nil, fmt.Errorf("data not found, try one of %v", valid)
	}
	tree := MetaTree{}
	if err := json.Unmarshal(all, &tree); err != nil {
		return nil, fmt.Errorf("error parsing metadata from %s: %w", "sample/new_sample.json", err)
	}
	return &tree, nil
}

// InjectOperatorPublicKey secures `the-operator` with an ssh key. The key should be
// in the form of a line from the `authorized_keys` file for ssh. This is used during
// the bootstrapping of a secure universe.
func (m *MetaTree) InjectOperatorPublicKey(sshPublicKey []byte) error {
	op, err := m.GetPath("am:/user/the-operator")
	if err != nil {
		return err
	}
	keyAnnotation, err := UserAnnotationString("ssh-pubkey", string(sshPublicKey))
	if err != nil {
		return err
	}
	op.Meta = append(op.Meta, keyAnnotation)
	return nil
}
