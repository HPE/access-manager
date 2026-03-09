/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package cmd

import (
	"reflect"
	"testing"
)

func Test_parseAces(t *testing.T) {
	tests := []struct {
		name    string
		perms   string
		want    []*ACE
		wantErr bool
	}{
		{
			"bare",
			`{"op": "VIEW","permissions": [{"roles": ["am://role/hpe-user"]}]}`,
			[]*ACE{
				{
					Op:            "VIEW",
					Unique:        0,
					Local:         false,
					Version:       0,
					EndTimeMillis: 0,
					Roles:         [][]string{{"am://role/hpe-user"}},
				},
			},
			false,
		},
		{
			"it's a wrap",
			`{"aces": {"op": "VIEW","permissions": [{"roles": ["am://role/hpe-user"]}]}}`,
			[]*ACE{
				{
					Op:            "VIEW",
					Unique:        0,
					Local:         false,
					Version:       0,
					EndTimeMillis: 0,
					Roles:         [][]string{{"am://role/hpe-user"}},
				},
			},
			false,
		}, {
			"double-wrapped to preserve freshness",
			`{"Details": {"aces": {"op": "VIEW","permissions": [{"roles": ["am://role/hpe-user"]}]}}}`,
			[]*ACE{
				{
					Op:            "VIEW",
					Unique:        0,
					Local:         false,
					Version:       0,
					EndTimeMillis: 0,
					Roles:         [][]string{{"am://role/hpe-user"}},
				},
			},
			false,
		},
		{
			name:  "bare with unique and version",
			perms: `{"unique": "123", "op": "VIEW", "local":true, "version": "321", "permissions": [{"roles": ["am://role/hpe-user"]}]}`,
			want: []*ACE{
				{
					Op:            "VIEW",
					Unique:        123,
					Local:         true,
					Version:       321,
					EndTimeMillis: 0,
					Roles:         [][]string{{"am://role/hpe-user"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAces(tt.perms)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !acesEqual(got, tt.want) {
				t.Errorf("parseAces() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func acesEqual(got []*ACE, want []*ACE) bool {
	for i, v := range got {
		if !aceEqual(v, want[i]) {
			return false
		}
	}
	return true
}

func aceEqual(got *ACE, want *ACE) bool {
	for i, v := range got.Roles {
		if !reflect.DeepEqual(v, want.Roles[i]) {
			return false
		}
	}

	return got.Unique == want.Unique && got.Local == want.Local &&
		got.Version == want.Version
}
