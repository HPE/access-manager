/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"context"
	"errors"
	"github.com/hpe/access-manager/internal/services/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanPath(t *testing.T) {
	ctx := context.Background()
	s, err := OpenTestStore("new_sample")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test hierarchy with permissions at different levels
	setupPathHierarchy(t, ctx, s)

	t.Run("scan_all_path_components", func(t *testing.T) {
		// This test checks that we process ACEs at each path component
		path := "am://user/hpe/bu1/alice"
		visitedPaths := []string{}
		visitedOps := []Operation{}

		err := s.ScanPath(ctx, path, true, func(path string, ace *ACE, abortScan error) error {
			visitedPaths = append(visitedPaths, path)
			visitedOps = append(visitedOps, ace.Op)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, []string{"am://user/hpe", "am://user/hpe/bu1", "am://user/hpe/bu1"}, visitedPaths)
		eq, diff := common.EqualSets([]Operation{View, Admin, Write}, visitedOps)
		assert.Truef(t, eq, "Expected operations to match, but got %v. Difference: %v", visitedPaths, diff)
	})

	t.Run("scan_without_last_component", func(t *testing.T) {
		// This test checks that when includeLast=false, we skip the last component
		path := "am://user/hpe/bu1/"
		visitedPaths := []string{}
		visitedOps := []Operation{}

		err := s.ScanPath(ctx, path, false, func(px string, ace *ACE, abortScan error) error {
			visitedPaths = append(visitedPaths, px)
			visitedOps = append(visitedOps, ace.Op)
			return nil
		})

		assert.NoError(t, err)
		eq, diff := common.EqualSets([]Operation{View}, visitedOps)
		assert.Truef(t, eq, "Expected operations to match, but got %v. Difference: %v", visitedOps, diff)
		assert.Equal(t, []string{"am://user/hpe"}, visitedPaths)
	})

	t.Run("early_termination", func(t *testing.T) {
		// This test checks that we can stop scanning early by returning the sentinel error
		path := "am://user/hpe/bu1/alice"
		visitedPaths := []string{}
		visitedOps := []Operation{}

		i := 0
		err := s.ScanPath(ctx, path, true, func(px string, ace *ACE, abortScan error) error {
			visitedPaths = append(visitedPaths, px)
			visitedOps = append(visitedOps, ace.Op)
			i++
			if i == 1 {
				return abortScan // Stop after processing alice's ACE
			}
			return nil
		})
		assert.Equal(t, 1, i, "Expected to process only two ACEs before stopping")
		assert.NoError(t, err)
		assert.Equal(t, []string{"am://user/hpe"}, visitedPaths)
		eq, diff := common.EqualSets([]Operation{View}, visitedOps)
		assert.Truef(t, eq, "Expected operations to match, but got %v. Difference: %v", visitedPaths, diff)
	})

	t.Run("error_propagation", func(t *testing.T) {
		// This test checks that errors from the checker function are propagated
		path := "am://user/hpe/bu1/alice"
		expectedErr := errors.New("test error")

		i := 0
		err := s.ScanPath(ctx, path, true, func(_ string, ace *ACE, abortScan error) error {
			i++
			if i == 2 {
				return expectedErr
			}
			return nil
		})

		assert.Equal(t, expectedErr, err)
	})

	t.Run("local_ace_behavior-1", func(t *testing.T) {
		// This test checks that local ACEs are not applied at steps before the last
		path := "am://user/hpe/bu1/alice"
		visitedTags := []string{}
		visitedPaths := []string{}

		err := s.ScanPath(ctx, path, true, func(p string, ace *ACE, abortScan error) error {
			if ace.Local {
				visitedTags = append(visitedTags, "local-"+ace.Tag)
			} else {
				visitedTags = append(visitedTags, ace.Tag)
			}
			visitedPaths = append(visitedPaths, p)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, len(visitedTags), "Expected to visit 3 ACEs")
		assert.Contains(t, visitedTags, "ace")
		assert.NotContains(t, visitedTags, "local-ace")
		assert.NotContains(t, visitedPaths, "am://")          // ACEs at this level are local and thus not applied
		assert.NotContains(t, visitedPaths, "am://user")      // ACEs at this level are local and thus not applied
		assert.Contains(t, visitedPaths, "am://user/hpe")     // ACEs at this level are *not* local and thus are applied
		assert.Contains(t, visitedPaths, "am://user/hpe/bu1") // ACEs at this level are *not* local and thus are applied
	})
	t.Run("local_ace_behavior-2", func(t *testing.T) {
		// This test checks that local ACEs are not applied at steps before the last
		path := "am://user/hpe"
		visitedTags := []string{}
		visitedPaths := []string{}

		err = s.ScanPath(ctx, path, true, func(p string, ace *ACE, abortScan error) error {
			if ace.Local {
				visitedTags = append(visitedTags, "local-"+ace.Tag)
			} else {
				visitedTags = append(visitedTags, ace.Tag)
			}
			visitedPaths = append(visitedPaths, p)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, len(visitedTags), "Expected to visit 2 ACEs")
		assert.Contains(t, visitedTags, "ace")
		assert.Contains(t, visitedTags, "local-ace")
		assert.Contains(t, visitedPaths, "am://user/hpe")
		assert.NotContains(t, visitedPaths, "am://user")
		assert.NotContains(t, visitedPaths, "am://")
	})

	t.Run("local_ace_behavior-3", func(t *testing.T) {
		// This test checks that local ACEs are not applied at steps before the last
		path := "am://user/hpe"
		visitedTags := []string{}
		visitedPaths := []string{}

		err = s.ScanPath(ctx, path, true, func(p string, ace *ACE, abortScan error) error {
			if ace.Local {
				visitedTags = append(visitedTags, "local-"+ace.Tag)
			} else {
				visitedTags = append(visitedTags, ace.Tag)
			}
			visitedPaths = append(visitedPaths, p)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, len(visitedTags), "Expected to visit 2 ACEs")
		assert.Contains(t, visitedTags, "ace")
		assert.Contains(t, visitedTags, "local-ace")
		assert.Contains(t, visitedPaths, "am://user/hpe")
		assert.NotContains(t, visitedPaths, "am://user")
		assert.NotContains(t, visitedPaths, "am://")
	})

	t.Run("local_ace_behavior-4", func(t *testing.T) {
		// This test should *only* see local ACEs and should only retain the ones at "am://user"
		path := "am://user"
		visitedTags := []string{}
		visitedPaths := []string{}

		err = s.ScanPath(ctx, path, true, func(p string, ace *ACE, abortScan error) error {
			if ace.Local {
				visitedTags = append(visitedTags, "local-"+ace.Tag)
			} else {
				visitedTags = append(visitedTags, ace.Tag)
			}
			visitedPaths = append(visitedPaths, p)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, len(visitedTags), "Expected to visit 2 ACEs")
		assert.NotContains(t, visitedTags, "ace")
		assert.Contains(t, visitedTags, "local-ace")
		assert.Contains(t, visitedPaths, "am://user")
		assert.NotContains(t, visitedPaths, "am://")
	})

	t.Run("non_existent_path", func(t *testing.T) {
		// Test behavior with a path that doesn't exist
		path := "am://user/nonexistent/path"
		called := false

		err := s.ScanPath(ctx, path, true, func(_ string, ace *ACE, abortScan error) error {
			called = true
			return nil
		})

		// The exact expected behavior depends on implementation details
		// Either it should return an error, or the checker should never be called
		if err == nil {
			assert.False(t, called, "Checker shouldn't be called for non-existent paths")
		}
	})
}

// Helper function to set up a test hierarchy with permissions
func setupPathHierarchy(t *testing.T, ctx context.Context, s AnnotationStore) {
	// Create base paths
	paths := []string{
		"am://user/alice",
		"am://user/alice/profile",
	}

	for _, path := range paths {
		_, err := s.PutNode(ctx, path, 0)
		if err != nil {
			t.Fatalf("Failed to create path %s: %v", path, err)
		}
	}

	// Add ACEs at each level
	_, err := s.Annotate(ctx, "am://user", &ACE{
		Tag:   "user-ace",
		Op:    View,
		Local: false,
		Acls: []*ACL{
			{
				Roles: []string{"am://role/common-folk"},
			},
		},
	})
	assert.NoError(t, err)

	_, err = s.Annotate(ctx, "am://user/alice", &ACE{
		Tag:   "alice-ace",
		Op:    View,
		Local: false,
		Acls: []*ACL{
			{
				Roles: []string{"am://role/alice-friends"},
			},
		},
	})
	assert.NoError(t, err)

	_, err = s.Annotate(ctx, "am://user/alice/profile", &ACE{
		Tag:   "profile-ace",
		Op:    View,
		Local: true,
		Acls: []*ACL{
			{
				Roles: []string{"am://role/alice-only"},
			},
		},
	})
	assert.NoError(t, err)

	// Add a local ACE at a non-leaf level to test local ACE behavior
	_, err = s.Annotate(ctx, "am://user", &ACE{
		Tag:   "user-local-ace",
		Op:    View,
		Local: true,
		Acls: []*ACL{
			{
				Roles: []string{"am://role/user-local"},
			},
		},
	})
	assert.NoError(t, err)
}
