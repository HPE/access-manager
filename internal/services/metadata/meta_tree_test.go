/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stretchr/testify/assert"
)

func TestPrincipalJSON(t *testing.T) {
	ua := UserAnnotation{
		Tag:         "principal",
		Version:     3,
		StartMillis: 55,
		EndMillis:   987,
	}
	b, err := protojson.Marshal(&ua)
	assert.NoError(t, err)
	assert.True(t, len(b) > 0)
	var ua2 UserAnnotation
	err = protojson.Unmarshal(b, &ua2)
	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(ua.Tag, ua2.Tag))
}

func TestLiveJSON(t *testing.T) {
	ace := ACE{
		Tag:   "ace",
		Op:    VouchFor,
		Local: true,
		Acls: []*ACL{
			{Roles: []string{"am://role/hpe-user"}},
		},
	}
	var ace2 ACE
	err := protojson.Unmarshal([]byte(`{"tag":"ace", "op":"VOUCHFOR", "local":true, "acls":[{"roles":["am://role/hpe-user"]}]}`), &ace2)
	assert.NoError(t, err)

	b1, err := protojson.Marshal(&ace)
	assert.NoError(t, err)
	b2, err := protojson.Marshal(&ace2)
	assert.NoError(t, err)
	assert.Equal(t, string(b1), string(b2))
}

func TestSampleMetaTree(t *testing.T) {
	tx, err := SampleMetaTree("new_sample")
	assert.NoError(t, err)
	err = tx.Walk(TOP_DOWN, func(tz *MetaTree) error {
		dir := false
		leaf := false
		for _, ann := range tz.Meta {
			switch ann.Tag {
			case "applied-role":
				_, err := ann.AsAppliedRole()
				assert.NoError(t, err)
			case "ace":
				_, err := ann.AsACE()
				assert.NoError(t, err)
			case "dir":
				dir = true
			case "leaf":
				leaf = true
			}
		}
		assert.True(t, dir || leaf)
		assert.False(t, dir && leaf)
		assert.True(t, len(tz.Children) == 0 || dir)   // has_child => dir
		assert.True(t, !leaf || len(tz.Children) == 0) // leaf => !has_child
		return nil
	})
	assert.NoError(t, err)

	_, err = SampleMetaTree("foo bar")
	assert.ErrorContains(t, err, "data not found, try one of [bootstrap new_sample")
}

//go:embed data/permission-1.json
//go:embed data/permission-2.json
//go:embed data/permission-3.json
var f0 embed.FS

func TestAceRoundTrip(t *testing.T) {
	var tests = []struct {
		file, sx string
	}{
		{"data/permission-1.json", "op:VIEW acls:{roles:\"am://role/hpe/hpe-user\"} tag:\"ace\" unique:8347083 version:2"},
		{"data/permission-2.json", "op:ADMIN acls:{roles:\"am://role/hpe/bu1/bu1-admin\"} tag:\"ace\" unique:64234897 version:2"},
		{"data/permission-3.json", "op:USEROLE acls:{roles:\"am://role/hpe/bu1/bu1-admin\"} tag:\"ace\" unique:467340 version:2"},
	}
	for _, tx := range tests {
		t.Run(tx.file, func(t *testing.T) {
			b, err := f0.ReadFile(tx.file)
			assert.NoError(t, err)
			var a ACE
			err = json.Unmarshal(b, &a)
			assert.NoError(t, err)
			assert.Equal(t, tx.sx, strings.ReplaceAll(a.String(), "  ", " "))

			b, err = json.MarshalIndent(&a, "", "  ")
			assert.NoError(t, err)

			var a2 *ACE
			err = json.Unmarshal(b, &a2)
			assert.NoError(t, err)
			assert.Equal(t, tx.sx, strings.ReplaceAll(a2.String(), "  ", " "))
		})
	}
}

func TestMetaTree_Walk(t *testing.T) {
	tree, _ := SampleMetaTree("new_sample")
	paths, err := pathCollector(tree, TOP_DOWN)
	assert.ErrorContains(t, err, "break loop at 6")
	assert.Equal(t, "[am:// am://user am://user/the-operator am://user/demo-god am://user/hpe am://user/hpe/bu1 am://user/hpe/bu1/x]", fmt.Sprintf("%v", paths))

	paths, err = pathCollector(tree, BOTTOM_UP)
	assert.ErrorContains(t, err, "break loop at 6")
	assert.Equal(t, "[am://user/the-operator am://user/demo-god am://user/hpe/bu1/x/y/buried am://user/hpe/bu1/x/y am://user/hpe/bu1/x am://user/hpe/bu1/invisible-man am://user/hpe/bu1/bob]", fmt.Sprintf("%v", paths))
}

func pathCollector(tree *MetaTree, direction TraversalOrder) ([]string, error) {
	var paths []string
	err := tree.Walk(direction, func(tx *MetaTree) error {
		if len(paths) > 6 {
			return fmt.Errorf("break loop at 6")
		}
		paths = append(paths, tx.Path)
		return nil
	})
	return paths, err
}

func TestMetaTree_Adjust(t *testing.T) {
	// verify that sane values come out of an adjusted MetaTree
	tree, err := SampleMetaTree("new_sample")
	if err != nil {
		t.Error(err)
	}
	tree.adjust()
	k1, k2, k3, n := 0, 0, 0, 0
	count := map[int64]int{}
	err = tree.Walk(TOP_DOWN, func(tx *MetaTree) error {
		for _, m := range tx.Meta {
			n++
			if m.Version == 0 {
				k1++
			}
			if m.Version == 0 {
				k3++
			}
			if m.Unique == 0 {
				k2++
			}
			count[m.Unique]++
		}
		return nil
	})
	if err != nil {
		t.Log("failed in tree walk ... impossible?")
		t.Fail()
	}
	assert.Greater(t, n, 70)
	assert.Equal(t, 0, k1)
	assert.Equal(t, 0, k2)
	assert.Equal(t, 0, k3)
	for k, cnt := range count {
		assert.Equal(t, 1, cnt, "#unique[%d] = %d", k, cnt)
		if cnt > 1 {
			fmt.Printf("count[%d] = %d\n", k, cnt)
		}
	}
}
