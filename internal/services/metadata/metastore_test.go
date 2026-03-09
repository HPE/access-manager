/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Regex(t *testing.T) {
	r := regexp.MustCompile(`^(a) ((b)(c))?$`)
	m := r.FindStringSubmatch(`a bc`)
	assert.NotNil(t, m)
	assert.Equal(t, "a", m[1])
	assert.Equal(t, "bc", m[2])
	assert.Equal(t, "b", m[3])
	assert.Equal(t, "c", m[4])

	m = r.FindStringSubmatch(`a b`)
	assert.Nil(t, m)

	m = r.FindStringSubmatch(`a `)
	assert.NotNil(t, m)
	assert.Equal(t, "a", m[1])
	assert.Equal(t, "", m[2])
	assert.Equal(t, "", m[3])
	assert.Equal(t, "", m[4])
}

func TestPut(t *testing.T) {
	s, err := OpenTestStore("bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	// jumped ahead of ourselves here because foo doesn't exist
	if _, err := s.PutNode(context.Background(), "am://user/foo/bar", 0); err == nil {
		t.Fatal("should have failed")
	}
	if v, err := s.PutNode(context.Background(), "am://user/foo", 0); err != nil {
		t.Fatal(err)
	} else {
		assert.Equal(t, int64(1), v)
	}
	if v, err := s.PutNode(context.Background(), "am://user/foo/baz", -1); err != nil {
		t.Fatal(err)
	} else {
		assert.Equal(t, int64(1), v)
	}
	if v, err := s.PutNode(context.Background(), "am://user/foo/baz", 1); err != nil {
		t.Fatal(err)
	} else {
		assert.Equal(t, int64(2), v)
	}

	if _, err := s.PutNode(context.Background(), "am://user/foo/pigdog", 3); err == nil {
		t.Fatal("should have failed update")
	} else {
		assert.ErrorContains(t, err, "does not exist")
	}
}

func TestAnnotate(t *testing.T) {
	s, err := OpenTestStore("bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.PutNode(context.Background(), "am://role/common-folk", 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = s.PutNode(context.Background(), "am://user/foo", 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = s.Annotate(context.Background(), "am://user/foo", &Annotation{
		Tag: "leaf",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = s.Annotate(context.Background(), "am://user/foo", &AppliedRole{
		Tag:  "applied-role",
		Role: "am://role/common-folk",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = s.Annotate(context.Background(), "am://user/foo", &ACE{
		Tag:   "ace",
		Op:    View,
		Local: false,
		Acls: []*ACL{
			{
				Roles: []string{"am://role/operator-admin", "am://role/common-folk"},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)

	sx, err := s.GetTree(context.Background())
	assert.NoError(t, err)
	js, err := json.MarshalIndent(sx, "", "  ")
	assert.NoError(t, err)
	fmt.Println(string(js))
}

func Test_parseKey(t *testing.T) {
	type args struct {
		k    []byte
		path string
	}
	tests := []struct {
		name    string
		args    args
		kind    string
		unique  int64
		wantErr bool
	}{
		{"root", args{k: []byte("123 am://root"), path: "am://root"}, "", 0, false},
		{"happy Path", args{k: []byte("234 am://root#ace-123"), path: "am://root"}, "ace", 123, false},
		{"long happy Path", args{k: []byte("1 am://role#ace-8746944124676713302"), path: "am://root"}, "ace", 8746944124676713302, false},
		{"bad unique", args{k: []byte("32 am://root#ace-xyz"), path: "am://root"}, "", 0, true},
		{"missing unique", args{k: []byte("134 am://root#ace"), path: "am://root"}, "", 0, true},
		{"missing kind", args{k: []byte("256 am://root#"), path: "am://root"}, "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, kind, unique, err := parseKey(tt.args.k, tt.args.path)
			if tt.wantErr != (err != nil) {
				return
			}
			assert.Equalf(t, tt.kind, kind, "parseKey(%v, %v)", tt.args.k, tt.args.path)
			assert.Equalf(t, tt.unique, unique, "parseKey(%v, %v)", tt.args.k, tt.args.path)
		})
	}
}
