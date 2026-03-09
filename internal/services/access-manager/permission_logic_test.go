/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package accessmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"regexp"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"

	"github.com/hpe/access-manager/internal/services/metadata"
)

var ctx = context.Background()

func TestDelegatingAccessManager_DeleteObject(t *testing.T) {
	testStore, err := metadata.OpenTestStore("new_sample")
	assert.NoError(t, err)

	am := NewPermissionLogic(testStore)
	ctx := context.Background()

	// Use alice as our caller who has admin permissions
	alice := "am://user/hpe/bu1/alice"
	bob := "am://user/hpe/bu1/bob"

	// this alice has admin permissions on yoyodyne
	demoGod := "am://user/demo-god"

	t.Run("delete_user", func(t *testing.T) {
		// Verify bob exists first
		exists, err := am.Exists(ctx, bob, alice)
		assert.NoError(t, err)
		assert.True(t, exists, "Bob should exist before deletion")

		// Delete bob
		err = am.DeleteObject(ctx, bob, false, alice)
		assert.NoError(t, err)

		// Verify bob is gone
		exists, err = am.Exists(ctx, bob, alice)
		assert.NoError(t, err)
		assert.False(t, exists, "Bob should not exist after deletion")
	})

	t.Run("delete_folder_recursive", func(t *testing.T) {
		folder := "am://data/yoyodyne"

		// Verify folder exists first
		exists, err := am.Exists(ctx, folder, alice)
		assert.NoError(t, err)
		assert.False(t, exists, "Alice shouldn't see yoyodyne folder")

		exists, err = am.Exists(ctx, folder, demoGod)
		assert.NoError(t, err)
		assert.True(t, exists, "Folder should exist before deletion")

		yoyoAlice, err := am.GetPrincipalCredential(context.Background(), "am://user/yoyodyne/alice", "am://workload/yoyodyne/id-plugin")
		assert.NoError(t, err, "Should get credential for alice")

		// Get children to verify it's not empty
		children, err := am.GetChildren(ctx, folder, yoyoAlice)
		assert.NoError(t, err)
		assert.NotEmpty(t, children, "Folder should have children")

		// Attempt to delete without recursive flag (should fail)
		err = am.DeleteObject(ctx, folder, false, yoyoAlice)
		assert.Error(t, err, "Should not be able to delete non-empty folder without recursive flag")

		// Delete with recursive flag
		err = am.DeleteObject(ctx, folder, true, yoyoAlice)
		assert.NoError(t, err)

		// Verify folder is gone
		exists, err = am.Exists(ctx, folder, yoyoAlice)
		assert.NoError(t, err)
		assert.False(t, exists, "Folder should not exist after deletion")
	})

	t.Run("permission_denied", func(t *testing.T) {
		// Recreate bob for this test
		leaf := &metadata.Annotation{Tag: "leaf"}
		err = am.CreateObject(ctx, bob, alice, leaf)
		assert.NoError(t, err)

		// Bob shouldn't have permission to delete alice
		err = am.DeleteObject(ctx, "am://user/hpe/bu1/alice", false, bob)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied", "Expected permission denied error")

		// Verify alice still exists
		exists, err := am.Exists(ctx, "am://user/hpe/bu1/alice", alice)
		assert.NoError(t, err)
		assert.True(t, exists, "Alice should still exist")
	})

	t.Run("delete_nonexistent_object", func(t *testing.T) {
		err = am.DeleteObject(ctx, "am://user/hpe/nonexistent", false, alice)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found", "Expected not found error")
	})
}

func TestDelegatingAccessManager_CreatePrincipal(t *testing.T) {
	testStore, err := metadata.OpenTestStore("new_sample")
	assert.NoError(t, err)

	am := NewPermissionLogic(testStore)
	ctx := context.Background()

	leaf := &metadata.Annotation{Tag: "leaf"}
	dir := &metadata.Annotation{Tag: "dir"}

	err = am.CreateObject(ctx, "bogus_uri!", "", leaf)
	assert.ErrorContains(t, err, "invalid credential")

	err = am.CreateObject(ctx, "am://user/hpe/fang_bob", "", leaf)
	assert.ErrorContains(t, err, "invalid credential")

	err = am.CreateObject(ctx, "am://user/hpe/fang_bob", "am://user/hpe/bu1/alice")
	assert.ErrorContains(t, err, "dir or leaf")

	err = am.CreateObject(ctx, "am://user/hpe/fang_bob", "am://user/hpe/bu1/alice", leaf)
	assert.NoError(t, err)

	err = am.CreateObject(ctx, "am://user/hpe/zoo", "am://user/hpe/bu1/alice", dir)
	assert.NoError(t, err)
	err = am.CreateObject(ctx, "am://user/hpe/zoo/tiger", "am://user/hpe/bu1/alice", leaf)
	assert.NoError(t, err)
	err = am.CreateObject(ctx, "am://user/hpe/zoo/lion", "am://user/hpe/bu1/alice", leaf)
	assert.NoError(t, err)

	sp, err := metadata.UserAnnotationString("svid", `spiffe:/hpe.com/major_tom`)
	assert.NoError(t, err)
	if err = am.CreateObject(ctx, "am://user/hpe/bu1/tom", "am://user/hpe/bu1/bob", sp, leaf); err == nil {
		t.Errorf("create should have failed due to lack of admin permission")
	}

	if err = am.CreateObject(ctx, "am://user/hpe/bu1/tom", "am://user/hpe/bu1/alice", sp, leaf); err != nil {
		t.Errorf("create should have succeeded but got: %s", err.Error())
	}

	if err = am.CreateObject(ctx, "am://user/hpe/bu1/tom", "am://user/hpe/bu1/alice", sp, leaf); err == nil {
		t.Errorf("create should have failed due to user already exists")
	}

	tom, err := am.Exists(ctx, "am://user/hpe/bu1/tom", "am://user/hpe/bu1/alice")
	assert.NoErrorf(t, err, "error looking for tom")
	assert.True(t, tom, "expected to find tom")

	tomIsDir, err := am.IsFolder(ctx, "am://user/hpe/bu1/tom", "am://user/hpe/bu1/bob")
	assert.NoError(t, err, "error testing tom")
	assert.Falsef(t, tomIsDir, "should not be a directory")

	ax, err := am.GetAnnotations(ctx, "am://user/hpe/bu1/tom", "am://user/hpe/bu1/alice", metadata.WithType("svid"))
	assert.Equal(t, 1, len(ax), "expected to find svid")
	mx, err := ax[0].AsUserAnnotation()
	assert.NoError(t, err, "error instantiating svid")
	s, err := mx.AsString("svid")
	assert.NoError(t, err, "error retrieving svid value")
	assert.Equal(t, `spiffe:/hpe.com/major_tom`, s)
}

func Test_Annotate(t *testing.T) {
	setup := func() PermissionLogic {
		testStore, err := metadata.OpenTestStore("new_sample")
		assert.NoError(t, err)
		return NewPermissionLogic(testStore)
	}
	ctx := context.Background()
	alice := "am://user/hpe/bu1/alice"
	bob := "am://user/hpe/bu1/bob"
	charlie := "am://user/hpe/bu1/charlie"
	t.Run("basic annotation", func(t *testing.T) {
		am := setup()
		adminRole, err := (&metadata.AppliedRole{Tag: "applied-role", Role: "am://role/hpe/bu1/bu1-admin"}).AsAnnotation()
		assert.NoError(t, err)
		err = am.Annotate(ctx, bob, adminRole, alice)
		assert.NoError(t, err)
	})
	t.Run("admin overreach", func(t *testing.T) {
		am := setup()
		operator, err := (&metadata.AppliedRole{Tag: "applied-role", Role: "am://role/operator-admin"}).AsAnnotation()
		assert.NoError(t, err)
		err = am.Annotate(ctx, bob, operator, alice)
		assert.ErrorContains(t, err, "permission denied for applying role")

		rx, err := am.GetRoles(ctx, bob, alice)
		assert.NoError(t, err)
		assert.Len(t, rx, 0)
	})

	t.Run("annotation update", func(t *testing.T) {
		am := setup()
		t1 := time.Now().Add(1 * time.Hour).UnixMilli()
		role := &metadata.AppliedRole{
			Tag:       "applied-role",
			Role:      "am://role/hpe/bu1/bu1-admin",
			Unique:    31,
			EndMillis: t1,
		}
		adminRole, err := role.AsAnnotation()
		assert.NoError(t, err)
		err = am.Annotate(ctx, bob, adminRole, alice)
		assert.NoError(t, err)

		role = &metadata.AppliedRole{
			Unique:    33,
			EndMillis: t1,
			Tag:       "applied-role",
			Role:      "am://role/hpe/bu1/bu1-user",
		}
		secondRole, err := role.AsAnnotation()
		assert.NoError(t, err)
		err = am.Annotate(ctx, bob, secondRole, alice)
		assert.NoError(t, err)

		t2 := time.Now().Add(2 * time.Hour).UnixMilli()
		adminRole.EndMillis = t2
		adminRole.Unique = 33 // collision by design!!
		_, err = am.UpdateAppliedRole(ctx, adminRole, bob, alice)
		assert.ErrorContains(t, err, "already has value")

		adminRole.Unique = 31
		adminRole.Version = 1
		_, err = am.UpdateAppliedRole(ctx, adminRole, bob, alice)
		assert.NoError(t, err)

		rx, err := am.GetRoles(ctx, bob, alice)
		assert.NoError(t, err)
		for _, ax := range rx {
			if ax.Unique == adminRole.Unique {
				r, err := ax.AsAppliedRole()
				assert.NoError(t, err)
				assert.Equal(t, `am://role/hpe/bu1/bu1-admin`, r.Role)
			}
		}
	})

	t.Run("delete role", func(t *testing.T) {
		am := setup()
		t1 := time.Now().Add(1 * time.Hour).UnixMilli()
		role := &metadata.AppliedRole{Tag: "applied-role", Role: "am://role/hpe/bu1/bu1-admin", EndMillis: t1}
		adminRole, err := role.AsAnnotation()

		assert.NoError(t, err)
		err = am.Annotate(ctx, bob, adminRole, alice)
		assert.NoError(t, err)

		rx, err := am.GetRoles(ctx, bob, alice)
		assert.NoError(t, err)
		assert.Len(t, rx, 1)
		err = am.DeleteAnnotation(ctx, bob, "applied-role", rx[0].Unique, alice)
		assert.NoError(t, err)

		err = am.DeleteAnnotation(ctx, bob, "applied-role", rx[0].Unique, alice)
		assert.ErrorContains(t, err, "no applied-role/", "should have failed")
		assert.ErrorContains(t, err, "annotation found on am://user/hpe/bu1/bob", "should have failed")

		rx, err = am.GetRoles(ctx, charlie, bob)
		assert.NoError(t, err)
		assert.Len(t, rx, 1, "bob should see charlies's roles")

		err = am.DeleteAnnotation(ctx, charlie, "applied-role", rx[0].Unique, bob)
		assert.ErrorContains(t, err, "permission denied", "bob doesn't have permission to delete charlie's role")

		err = am.DeleteAnnotation(ctx, charlie, "applied-role", rx[0].Unique, alice)
		assert.NoError(t, err, "alice should be able to delete charlie's role")
	})

	t.Run("add permission, can't UseRole", func(t *testing.T) {
		am := setup()
		hidden, err := (&metadata.ACE{
			Tag:   "ace",
			Op:    metadata.View,
			Local: false,
			Acls: []*metadata.ACL{
				{Roles: []string{"am://role/operator-admin"}},
			},
		}).AsAnnotation()
		// try to restrict bob's visibility, but Alice can't apply the operator role
		err = am.Annotate(ctx, bob, hidden, alice)
		assert.ErrorContains(t, err, "permission denied for using role")
	})
	t.Run("add permission", func(t *testing.T) {
		am := setup()
		hidden, err := (&metadata.ACE{
			Tag:   "ace",
			Op:    metadata.View,
			Local: false,
			Acls: []*metadata.ACL{
				{Roles: []string{"am://role/hpe/bu1/bu1-admin"}},
			},
		}).AsAnnotation()
		assert.NoError(t, err)
		// restrict bob's visibility
		err = am.Annotate(ctx, bob, hidden, alice)
		assert.NoError(t, err)

		// alice can see him, but bob can't see himself
		_, err = am.GetRoles(ctx, bob, alice)
		assert.NoError(t, err)
		rx, err := am.GetRoles(ctx, bob, bob)
		assert.ErrorContains(t, err, "bob not found")
		assert.Equal(t, 0, len(rx))

		// remove the restriction
		ax, err := am.GetACEs(ctx, bob, alice)
		assert.NoError(t, err)
		assert.Len(t, ax, 1)
		assert.Equal(t, int64(1), ax[0].Version)
		ace, err := ax[0].AsACE()
		assert.NoError(t, err)
		assert.Equal(t, metadata.View, ace.Op)
		unique := ax[0].Unique
		err = am.DeleteAnnotation(ctx, bob, "ace", unique, alice)
		assert.NoError(t, err)
	})
	t.Run("can't remove ACE without ApplyRole permission", func(t *testing.T) {
		am := setup()
		hidden, err := (&metadata.ACE{
			Tag:   "ace",
			Op:    metadata.View,
			Local: false,
			Acls: []*metadata.ACL{
				{Roles: []string{"am://role/hpe/bu1/bu1-user"}},
			},
		}).AsAnnotation()
		assert.NoError(t, err)
		// Put the ace directly on bob, it is redundant but still there
		err = am.Annotate(ctx, bob, hidden, alice)
		assert.NoError(t, err)

		ax, err := am.GetACEs(ctx, bob, alice)
		assert.NoError(t, err)
		assert.Len(t, ax, 1)
		assert.Equal(t, int64(1), ax[0].Version)
		ace, err := ax[0].AsACE()
		assert.NoError(t, err)
		assert.Equal(t, metadata.View, ace.Op)
		unique := ax[0].Unique

		// bob can't remove it
		err = am.DeleteAnnotation(ctx, bob, "ace", unique, bob)
		assert.ErrorContains(t, err, "permission denied, must have View and Admin permissions")

		// remove the restriction
		err = am.DeleteAnnotation(ctx, bob, "ace", unique, alice)
		assert.NoError(t, err)
	})
}

func Test_InheritedRoles(t *testing.T) {
	testStore, err := metadata.OpenTestStore("new_sample")
	assert.NoError(t, err)
	am := NewPermissionLogic(testStore)
	ctx := context.Background()

	t.Run("basic inheritance", func(t *testing.T) {
		inheritedRoles, err := am.GetInheritedRoles(ctx, "am://user/hpe/bu1/alice", "am://user/hpe/bu1/alice")
		assert.NoError(t, err)
		r := []string{}
		for _, ax := range inheritedRoles {
			rx, err := ax.AsAppliedRole()
			assert.NoError(t, err)
			r = append(r, rx.Role)
		}
		assert.Equal(t, 3, len(inheritedRoles))
		assert.Equal(t, []string{"am://role/hpe/hpe-user", "am://role/hpe/bu1/bu1-user", "am://role/hpe/bu1/bu1-admin"}, r)
	})

	t.Run("inherited roles, limited visibility", func(t *testing.T) {
		inheritedRoles, err := am.GetInheritedRoles(ctx, "am://user/hpe/bu1/alice", "am://user/hpe/bu2/alice")
		assert.NoError(t, err)
		r := []string{}
		for _, ax := range inheritedRoles {
			rx, err := ax.AsAppliedRole()
			assert.NoError(t, err)
			r = append(r, rx.Role)
		}
		assert.Equal(t, 3, len(inheritedRoles))
		assert.Equal(t, []string{"am://role/hpe/hpe-user", "am://role/hpe/bu1/bu1-user", "## Redacted role ##"}, r)
	})
}

func Test_GetAces(t *testing.T) {
	testStore, err := metadata.OpenTestStore("new_sample")
	assert.NoError(t, err)
	am := NewPermissionLogic(testStore)
	ctx := context.Background()

	t.Run("basics", func(t *testing.T) {
		r, err := am.GetACEs(ctx, "am://user/hpe/bu1", "am://user/hpe/bu1/alice")
		assert.NoError(t, err)

		assert.NoError(t, compareAnnotations(r, []string{
			`{"op": "WRITE","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin"]}],"tag": "ace"}`,
			`{"op": "ADMIN","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin"]}],"tag": "ace"}`,
		}, ""))
	})
}

func Test_GetChildren(t *testing.T) {
	testStore, err := metadata.OpenTestStore("new_sample")
	assert.NoError(t, err)
	am := NewPermissionLogic(testStore)
	ctx := context.Background()
	t.Run("basics", func(t *testing.T) {
		got, err := am.GetChildren(ctx, "am://role/hpe", "am://user/hpe/bu1/bob")
		assert.NoError(t, err)
		expected := []string{
			"am://role/hpe/hpe-workload", "am://role/hpe/bu2", "am://role/hpe/hpe-user",
			"am://role/hpe/bu1", "am://role/hpe/hpe-admin", "am://role/hpe/id-plugin"}
		ok, diff := common.EqualSets(got, expected)
		if !ok {
			t.Errorf("got %v, expected %v", diff, expected)
		}

	})
}

func compareAnnotations(a []*metadata.Annotation, b []string, message string) error {
	ax := map[string]string{}
	whiteSpace := regexp.MustCompile(`\s*`)
	extraFields := regexp.MustCompile(`,\s*("unique"|"version"):\s*\d+`)
	for _, x := range a {
		s := common.CleanJson(x)
		s = extraFields.ReplaceAllString(s, "")
		key := whiteSpace.ReplaceAllString(s, "")

		ax[key] = s
	}
	bx := map[string]string{}
	for _, s := range b {
		s = extraFields.ReplaceAllString(s, "")
		key := whiteSpace.ReplaceAllString(s, "")
		bx[key] = s
	}
	ak := mapset.NewSetFromMapKeys(ax)
	bk := mapset.NewSetFromMapKeys(bx)
	if !ak.Equal(bk) {
		extra := values(ax, ak.Difference(bk))
		missing := values(bx, bk.Difference(ak))
		return fmt.Errorf("%s: got extra %v, missing %v", message, extra, missing)
	}
	return nil
}

type opHolder struct {
	A, B, C metadata.Operation
}

func Test_Operation(t *testing.T) {
	t.Run("round-trip", func(t *testing.T) {
		x := opHolder{metadata.Admin, metadata.View, metadata.ApplyRole}
		z, err := json.Marshal(x)
		if err != nil {
			t.Errorf("failed to marshal: %s", err.Error())
		}
		if string(z) != `{"A":"ADMIN","B":"VIEW","C":"APPLYROLE"}` {
			t.Errorf("invalid marshaled form: %s", string(z))
		}
		y := opHolder{}
		err = json.Unmarshal(z, &y)
		if err != nil {
			t.Errorf("failed to unmarshal %s: %s", z, err.Error())
		}
		assert.Equal(t, x, y, "unmarshaled forms not equal")
	})
	t.Run("invalid-op", func(t *testing.T) {
		z := "{\"A\":\"Admin\",\"B\":\"View\",\"C\":\"Bogus\"}"
		y := opHolder{}
		err := json.Unmarshal([]byte(z), &y)
		assert.ErrorContains(t, err, "invalid operation")
	})
	t.Run("invalid-op-2", func(t *testing.T) {
		z := "{\"A\":\"Admin\",\"B\":\"View\",\"C\":3}"
		y := opHolder{}
		err := json.Unmarshal([]byte(z), &y)
		assert.ErrorContains(t, err, "invalid operation")
	})

}

func TestDelegationToken(t *testing.T) {
	testStore, err := metadata.OpenTestStore("new_sample")
	assert.NoError(t, err)
	am := NewPermissionLogic(testStore)
	// get an AM credential for bob, who has some attributes for DAOS
	cred, err := am.GetPrincipalCredential(
		context.Background(),
		"am://user/yoyodyne/bob",
		"am://workload/yoyodyne/id-plugin",
	)
	// now get the delegation credential for the DAOS system
	a, b, err := am.GetDatasetCredential(
		context.Background(),
		"am://data/yoyodyne/daos1",
		[]metadata.Operation{metadata.Read},
		cred,
	)
	assert.NoError(t, err)
	assert.Equal(t, "", b)
	// result should be a valid and signed JWT
	claims, err := testStore.ValidateJWT(a)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	// the claims should have the two DAOS roles that bob has out of the three that exist
	ok, diff := common.EqualSets([]string{"am://role/yoyodyne/daos/raw", "am://role/yoyodyne/daos/ref"}, claims.Roles)
	assert.True(t, ok)
	if !ok {
		fmt.Printf("diff %v\n", diff)
	}

	cred, err = am.GetPrincipalCredential(
		context.Background(),
		"am://user/yoyodyne/alice",
		"am://workload/yoyodyne/id-plugin",
	)
	a, b, err = am.GetDatasetCredential(
		context.Background(),
		"am://data/yoyodyne/daos1",
		[]metadata.Operation{metadata.Read},
		cred,
	)
	assert.ErrorContains(t, err, "[READ] not allowed on am://data/yoyodyne/daos1")
	assert.Equal(t, "", b)
}
