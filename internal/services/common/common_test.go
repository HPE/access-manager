/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestUrlPattern(t *testing.T) {
	assert.True(t, StandardPattern.MatchString(StandardPrefix))
	assert.True(t, StandardPattern.MatchString("am://role"))
	assert.True(t, StandardPattern.MatchString("am://role/foo"))
	assert.True(t, StandardPattern.MatchString("am://user/foo/"))
}

func TestJoin(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected string
	}{
		{"am://user", "alice", "am://user/alice"},
		{"am://user/", "alice", "am://user/alice"},
		{"am://user", "/alice", "am://user/alice"},
		{"am://user/", "/alice", "am://user/alice"},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := Join(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanup(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"am://user", "am://user"},
		{"am://user/", "am://user"},
		{"am://user//alice", "am://user/alice"},
		{"am://user///alice", "am://user/alice"},
		{"am://user/alice/", "am://user/alice"},
		{"am:////user", "am://user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Cleanup(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathComponents(t *testing.T) {
	tests := []struct {
		path     string
		expected []int
		bits     []string
	}{
		{"am://abc/def/g", []int{6, 9, 13, 15}, []string{"am://", "am://abc", "am://abc/def", "am://abc/def/g"}},
		{"am://abc", []int{6, 9}, []string{"am://", "am://abc"}},
		{"am://abc/", []int{6, 9}, []string{"am://", "am://abc"}},
		{"am://", []int{6}, []string{"am://"}},
		{"not-a-valid-path", []int{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := PathComponents(tt.path)
			assert.Equal(t, tt.expected, result)
			for i, idx := range result {
				if i < len(tt.bits) {
					assert.Equal(t, tt.bits[i], tt.path[:idx])
				} else {
					assert.Fail(t, "More indices than expected bits")
				}
			}
		})
	}
}

func TestCleanJson(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			"simple object",
			map[string]string{"key": "value"},
			`{
   "key": "value"
}`,
		},
		{
			"nested object",
			map[string]interface{}{"key": map[string]string{"nested": "value"}},
			`{
   "key": {
      "nested": "value"
   }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanJson(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEqualSets(t *testing.T) {
	tests := []struct {
		name           string
		actual         []string
		expected       []string
		shouldBeEqual  bool
		diffCardinaliy int
	}{
		{
			"identical sets",
			[]string{"a", "b", "c"},
			[]string{"a", "b", "c"},
			true,
			0,
		},
		{
			"subset",
			[]string{"a", "b", "c"},
			[]string{"a", "c"},
			false,
			1,
		},
		{
			"same elements different order",
			[]string{"c", "a", "b"},
			[]string{"a", "b", "c"},
			true,
			0,
		},
		{
			"different sets",
			[]string{"a", "b", "c"},
			[]string{"a", "b", "d"},
			false,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal, diff := EqualSets(tt.actual, tt.expected)
			assert.Equal(t, tt.shouldBeEqual, equal)
			assert.Equal(t, tt.diffCardinaliy, diff.Cardinality())
		})
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Now().UnixMilli()
	future := now + 10000
	past := now - 10000

	tests := []struct {
		name     string
		time     int64
		expected bool
	}{
		{"zero time", 0, false},
		{"future time", future, false},
		{"past time", past, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExpired(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFixJson(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"empty objects removed",
			`{
  "Children": [], 
  "Meta": []
}`,
			"{\n}",
		},
		{
			"null converted to empty array",
			`{"values": null}`,
			`{"values": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixJson([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParent(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"am://abc/def/ghi", "am://abc/def"},
		{"am://abc/def", "am://abc"},
		{"am://abc", "am://"},
		{"am://", "am://"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := Parent(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeUnique(t *testing.T) {
	// Generate multiple values and verify they're all within JavaScript's safe integer range
	for i := 0; i < 100; i++ {
		value := SafeUnique()
		assert.LessOrEqual(t, value, int64(1<<53-1))
		assert.GreaterOrEqual(t, value, int64(0))
	}
}

func TestValidPrincipal(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"am://user/alice", true},
		{"am://workload/service", true},
		{"am://role/admin", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := ValidPrincipal(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}
