/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package accessmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hpe/access-manager/pkg/logger"
	"google.golang.org/protobuf/proto"

	"slices"

	"github.com/hpe/access-manager/internal/services/metadata"
)

type OperationLog struct {
	who   string
	did   int32
	to    string
	start uint64
	end   uint64
}

/*
PermissionLogic implements the core intelligence of the Access Manager including
checking permissions, but does not handle any version checking. Data is stored
in an underlying metadata store which does version checking and persistence of
metadata.
*/
type PermissionLogic interface {
	// GetPrincipalCredential validates that the caller is allowed to vouch for a
	// user or workload and, if so, generates and returns a credential that can be
	// used by that user or workload on subsequently validate their identity. The
	// caller is typically an identity plugin acting on behalf of a user or workload.
	GetPrincipalCredential(ctx context.Context, principal string, callerId string) (string, error)

	// CreateObject establishes a node in the metadata tree that must be a directory
	// or a leaf. A directory can be the parent of other directories or leaves. A
	// leaf is something like a user, role or dataset and cannot contain other nodes.
	// CreateObject should always be called with a "leaf" or "dir" annotation; an
	// error will be returned otherwise.
	CreateObject(ctx context.Context, path string, callerId string, annotations ...*metadata.Annotation) error

	Annotate(ctx context.Context, path string, annotation *metadata.Annotation, callerId string) error
	GetAnnotations(ctx context.Context, path string, callerID string, filters ...metadata.KeyOption) ([]*metadata.Annotation, error)

	// DeleteObject deletes the specified object which can be a principal, role, user or workload.
	// Note that this will force the deletion of all metadata attached to the object as well.
	// If the object is a directory, then it must be empty or else the recursive flag must be
	// used to force the deletion of all children.
	DeleteObject(ctx context.Context, uri string, recursive bool, callerID string) error

	DeleteAnnotation(ctx context.Context, path string, tag string, unique int64, callerID string) error

	// GetDatasetCredential returns a delegation token or a credential suitable for the datastore
	GetDatasetCredential(ctx context.Context, path string, ops []metadata.Operation, callerID string) (string, string, error)

	// Access control expression operations -- these operations manipulate the access control
	// expressions associated with paths.

	// GetACEs returns a list of access control expressions for a URI. The version of the
	// metadata is also returned to allow safe update. The version should be used when
	// calling UpdateAces. This function will return an error if the id is invalid. An
	// error will also be returned if the caller does not have permission for the visible
	// operation.
	GetACEs(ctx context.Context, uri string, callerID string) ([]*metadata.Annotation, error)

	// UpdateACE sets the ACE for a path. ACEs have a globally unique identifier that can
	// be used to determine if the ACEs being passed in to this call refer to ACEs already
	// on the path or not. ACEs that do not already exist on the path but which are given
	// in the argument to UpdateACEs will be added. Those ACEs that already exist on the
	// path, but which are not mentioned in the argument here will be deleted. Those ACEs
	// that exist on the path and which are passed in here will be updated. An error will
	// be returned if the caller id does not exist as a principal. An error will be returned
	// if the version does not match the current version associated with the URI being
	// modified. An error will be returned if the caller does not have permission for the
	// admin operation on the specified object. An error will be returned if caller does
	// not have `UseRole` permission on any roles in ACEs that are deleted, updated or added.
	// An error may be returned if the ACEs refer to any operation that are not applicable
	// to the path.
	UpdateACE(ctx context.Context, uri string, perm *metadata.Annotation, callerID string) (int64, error)
	//	UpdateACE(ctx context.Context, uri string, perm *metadata.ACE, callerID string) (uint64, error)

	// Role operations -- these operations concern the creation and use of roles. Roles have
	// operations `View` and `Admin` common with any paths and with the conventional meanings,
	// but they also have `UseRole` and `ApplyRole` operations. The `ApplyRole` ACE controls
	// which principals can add or remove this role to a path. The `UseRole` ACE controls
	// which principals can add, update or delete an ACE that contains this role.

	// GetRoles returns a list of roles for any path starting with `am://user` or
	// `am://workload`. The roles are returned as Annotation structures including a
	// version number to allow safe update of individual applied roles. This function
	// will return an error if the URI is invalid. An error will also be returned if
	// the requester does not have permission for the `View` operation on the path
	// and all prefixes. The roles returned are for the exact path given and do not
	// include any roles inherited from prefixes of the path.
	GetRoles(ctx context.Context, uri string, callerID string) ([]*metadata.Annotation, error)

	// GetAllRoles returns a list all the children and grand children of roles for
	// any path starting with `am://role` This function will return an error
	// if the path is invalid. An error will also be returned if the requester does
	// not have permission for the `View` operation on the path and all prefixes.
	GetAllRoles(ctx context.Context, path, callerID string) ([]string, error)

	GetDetails(ctx context.Context, path string, includeChildren bool, cred string) (*NodeDetails, []string, error)

	// GetInheritedRoles returns a list of all inherited or direct roles for any path starting
	// with `am://user` or `am://workload`. No version is returned because the roles
	// returned don't come from any single structure that could be updated. This function
	// will return an error if the URI is invalid. An error will also be returned if
	// the requester does not have permission for the `View` operation on the path
	// and all prefixes. The roles returned are for the exact path given and do not include
	// any roles inherited from prefixes of the path.
	GetInheritedRoles(ctx context.Context, uri string, callerID string) ([]*metadata.Annotation, error)

	// UpdateAppliedRole adds roles to or removes roles from a principal. An error will be returned
	// if the caller id does not exist as a principal. An error will be returned if the version
	// does not match the current version of the URI. An error will be returned if the caller
	// does not have permission for the `Admin` and `View` operations on the path or if the caller
	// does not have permission for the Apply operation on any roles being added or removed.
	UpdateAppliedRole(ctx context.Context, appliedRole *metadata.Annotation, uri, callerID string) (int64, error)

	// ValidateRoles determines if the list of given roles are valid and exist in the store
	ValidateRoles(ctx context.Context, roles []string) error

	// Exists returns true if a path refers to a directory or a leaf node
	Exists(ctx context.Context, path string, callerId string) (bool, error)

	// IsFolder returns true if a path exists and refers to a directory (i.e. not a Principal, Role or Data)
	IsFolder(ctx context.Context, uri string, callerID string) (bool, error)

	// GetChildren returns a list of the children of a path that are visible to the caller.
	GetChildren(ctx context.Context, path string, callerID string) ([]string, error)

	// Bootstrap loads the metadata store with a bootstrap file. The bootstrap file must be one of
	// a small number of predefined files. The key (if present) is injected as the ssh public key
	// for the operator user.
	Bootstrap(bootstrap string, key string) error

	// GetSigningKeys returns a list of public keys. Any unexpired credential
	// will have been signed the private key corresponding to one of these..
	GetSigningKeys(ctx context.Context, id string) (map[int64]string, error)

	// ValidateCredential verifies that a credential has been properly signed and has not
	// expired.
	ValidateCredential(ctx context.Context, credential string, callerId string) (string, error)
}

type PermissionLogicManager struct {
	ms  metadata.MetaStore
	key []byte
}

func (plm *PermissionLogicManager) ValidateCredential(_ context.Context, credential string, _ string) (string, error) {
	claims, err := plm.ms.ValidateJWT(credential)
	if err != nil {
		return "", err
	}
	r, err := json.MarshalIndent(claims, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling claims: %w", err)
	}
	return string(r), nil
}

func (plm *PermissionLogicManager) GetSigningKeys(ctx context.Context, _ string) (map[int64]string, error) {
	return plm.ms.GetPublicSigningKeys(ctx)
}

var _ PermissionLogic = &PermissionLogicManager{}

func NewPermissionLogic(pc metadata.MetaStore) PermissionLogic {
	return &PermissionLogicManager{ms: pc}
}

// checkCredential evaluates whether a credential is valid and returns the associated user or workload.
//
// There are two alternatives. The credential can be a user or workload name. If
// so, that identity must not have any ssh keys, nor have any direct or inherited
// `VouchFor` permissions. The other alternative is that the credential is a JWT
// signed using a valid key and not yet expired. In that case, the JWT will
// contain the user or workload name.
func (plm *PermissionLogicManager) checkCredential(ctx context.Context, cred string) (string, error) {
	if strings.HasPrefix(cred, common.UserPrefix) || strings.HasPrefix(cred, common.WorkloadPrefix) {
		// the "credential" supplied is just a user or workload name. That's only OK in limited cases
		// check to verify no ssh keys or inherited or direct VouchFor opinions
		// and if not, just trust this ID
		ax, err := plm.ms.Get(ctx, cred, metadata.WithType("ssh-pubkey"))
		if err != nil {
			return "", err
		}
		if len(ax) != 0 {
			return "", fmt.Errorf("%s has ssh key, can't use plaintext ID", cred)
		}
		hasIdentityPlugins := false
		err = plm.ms.ScanPath(ctx, cred, true, func(_ string, ace *metadata.ACE, done error) error {
			if ace.Op == metadata.VouchFor {
				hasIdentityPlugins = true
				return done
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		if hasIdentityPlugins {
			return "", fmt.Errorf("%s has identity plugins, can't use plaintext ID", cred)
		}
		if !common.ValidPrincipal(cred) {
			return "", fmt.Errorf("%s has invalid user or workload name", cred)
		}
		return cred, nil

	} else {
		// credential should be a JWT, decode and check for valid signature to find the user
		claims, err := plm.ms.ValidateJWT(cred)
		if err != nil {
			return "", err
		}
		callerID := claims.Identity
		if !common.ValidPrincipal(callerID) {
			return "", errors.New("invalid user or workload name inside valid JWT")
		}
		return callerID, nil
	}
}

func (plm *PermissionLogicManager) Annotate(ctx context.Context, path string, annotation *metadata.Annotation, cred string) error {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return err
	}
	flag, err := plm.checkPermissions(
		ctx,
		path,
		callerID,
		[]metadata.Operation{metadata.View, metadata.Admin},
		true,
	)
	if err != nil {
		return err
	}
	if !flag {
		return fmt.Errorf("permission denied")
	}

	// permission may be required on the annotation itself (or parts thereof)
	if annotation.Tag == "applied-role" {
		role, err := annotation.AsAppliedRole()
		if err != nil {
			return err
		}
		flag, err = plm.checkPermissions(
			ctx,
			role.Role,
			callerID,
			[]metadata.Operation{metadata.View, metadata.ApplyRole},
			true,
		)
		if err != nil {
			return err
		}
		if !flag {
			return fmt.Errorf("permission denied for applying role %s", role)
		}
	} else if annotation.Tag == "ace" {
		ace, err := annotation.AsACE()
		if err != nil {
			return err
		}
		var roles []string
		for _, perm := range ace.Acls {
			roles = append(roles, perm.Roles...)
		}

		for _, role := range mapset.NewSet(roles...).ToSlice() {
			flag, err = plm.checkPermissions(
				ctx,
				role,
				callerID,
				[]metadata.Operation{metadata.View, metadata.UseRole},
				true,
			)
			if err != nil {
				return err
			}
			if !flag {
				return fmt.Errorf("permission denied for using role %s in permission", role)
			}
		}
	}
	_, err = plm.ms.Annotate(ctx, path, annotation)
	if err != nil {
		return err
	}
	return err
}

func (plm *PermissionLogicManager) CreateObject(ctx context.Context, path string, cred string, annotations ...*metadata.Annotation) error {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return err
	}
	// must have leaf or dir annotation
	if len(annotations) < 1 {
		return fmt.Errorf("annotations must contain at least a dir or leaf annotation")
	}
	isLeaf, isDir := false, false
	for _, annotation := range annotations {
		if annotation.Tag == "leaf" {
			isLeaf = true
		}
		if annotation.Tag == "dir" {
			isDir = true
		}
		if annotation.Tag == "" {
			return fmt.Errorf("all annotations must contain a tag")
		}
	}
	if !(isLeaf || isDir) {
		return fmt.Errorf("annotations must contain leaf or dir annotation")
	}
	flag, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View, metadata.Write}, false)
	if err != nil {
		return err
	}
	if !flag {
		return fmt.Errorf("permission denied for object creation")
	}

	_, err = plm.ms.PutNode(ctx, path, 0)
	if err != nil {
		return err
	}
	for _, annotation := range annotations {
		_, err := plm.ms.Annotate(ctx, path, annotation)
		if err != nil {
			return err
		}
	}
	return nil
}

func (plm *PermissionLogicManager) DeleteAnnotation(ctx context.Context, path string, tag string, unique int64, cred string) error {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return err
	}
	ax, err := plm.ms.Get(ctx, path, metadata.WithType(tag), metadata.WithUnique(unique))
	if err != nil {
		return err
	}
	// since we used the unique in our search we should only get one result
	if len(ax) == 0 {
		return fmt.Errorf("no %s/%d annotation found on %s", tag, unique, path)
	}
	if len(ax) > 1 {
		return fmt.Errorf("%s/%d annotation not unique on %s (found %d duplicates)", tag, unique, path, len(ax))
	}

	flag, err := plm.checkPermissions(
		ctx,
		path,
		callerID,
		[]metadata.Operation{metadata.View, metadata.Admin},
		true,
	)
	if err != nil {
		return err
	}
	if !flag {
		return fmt.Errorf("permission denied, must have View and Admin permissions for %s", path)
	}

	switch tag {
	case "role":
		return fmt.Errorf("to delete a role, you must use a tag of applied-role, not role")
	case "applied-role":
		// roles require same permission to delete as to add
		role, err := ax[0].AsAppliedRole()
		if err != nil {
			return err
		}
		flag, err := plm.checkPermissions(
			ctx,
			role.Role,
			callerID,
			[]metadata.Operation{metadata.View, metadata.ApplyRole},
			true,
		)
		if err != nil {
			return err
		}
		if !flag {
			return fmt.Errorf("permission denied, must have View and ApplyRole permissions for %s", role)
		}
	case "ace":
		// we can't remove an ACE unless we could have added it originally
		ace, err := ax[0].AsACE()
		if err != nil {
			return err
		}
		var roles []string
		for _, perm := range ace.Acls {
			roles = append(roles, perm.Roles...)
		}

		for _, role := range mapset.NewSet(roles...).ToSlice() {
			flag, err := plm.checkPermissions(
				ctx,
				role,
				callerID,
				[]metadata.Operation{metadata.View, metadata.UseRole},
				true,
			)
			if err != nil {
				return err
			}
			if !flag {
				return fmt.Errorf("cannot delete ACE, permission denied for using role %s", role)
			}
		}
	default:
		// no special permissions required for user annotations
	}

	if err := plm.ms.DeleteAnnotation(ctx, path, tag, -1, unique); err != nil {
		return err
	}
	return nil
}

// GetPrincipalCredential returns a credential for the specified principal based
// on the authority of the user or workload in the specified credential `cred`.
// The caller must have `View` and `VouchFor` permission on `principal`, else an error will
// be returned. This is the mechanism that identity plugins use to get
// credentials for users or workloads.
func (plm *PermissionLogicManager) GetPrincipalCredential(ctx context.Context, principal, cred string) (string, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return "", err
	}
	flag, err := plm.checkPermissions(ctx, principal, callerID, []metadata.Operation{metadata.View, metadata.VouchFor}, true)
	if err != nil {
		return "", err
	}

	if !flag {
		return "", errors.New("permission denied")
	}

	return plm.ms.GetSignedJWTWithClaims(30*time.Minute, metadata.DefaultClaims(principal, nil))
}

func (plm *PermissionLogicManager) DeleteObject(
	ctx context.Context,
	path string,
	recursive bool,
	cred string,
) error {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return err
	}
	// check target is well-formed
	if !strings.HasPrefix(path, common.StandardPrefix) {
		return errors.New("invalid URI, must start with am://")
	}

	callerRoles, err := plm.getEffectiveRoles(ctx, callerID)
	if err != nil {
		return err
	}

	// there are two basic paths from here. The first is the deletion of a non-empty
	// directory and the other is deletion of an object or empty directory. The
	// difference host mostly to do with how much work we have to do to check for
	// permission.

	isFolder, err := plm.IsFolder(ctx, path, cred)
	if err != nil {
		return fmt.Errorf(`can't determine if "%s" is a folder: %w`, path, err)
	}

	// can our caller see the object in question?
	visible, err := plm.checkRolePermissions(ctx, path, callerRoles, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return err
	}
	if !visible {
		return errors.New("object " + path + " not found")
	}

	// we can see it, but do we have permissions on the directory it is in?
	directoryWriteAllowed, _ := plm.checkRolePermissions(ctx, path, callerRoles, []metadata.Operation{metadata.Write}, false)
	if !directoryWriteAllowed {
		return errors.New("permission denied on directory " + common.Parent(path))
	}

	if isFolder {
		return plm.deleteFolder(ctx, path, recursive, callerRoles)
	} else {
		return plm.ms.Delete(ctx, path)
	}
}

func (plm *PermissionLogicManager) deleteFolder(ctx context.Context, path string, recursive bool, callerRoles []*metadata.Annotation) error {
	if recursive {
		children, err := plm.getAllChildren(ctx, path, true)
		if err != nil {
			return fmt.Errorf(`error getting children of "%s": %w`, path, err)
		}
		allowed := true
		denials := []string{}
		for _, child := range children {
			directoryWriteAllowed, _ := plm.checkRolePermissions(
				ctx,
				child,
				callerRoles,
				[]metadata.Operation{metadata.Write},
				false,
			)
			if !directoryWriteAllowed {
				denials = append(denials, child)
			}
			allowed = allowed && directoryWriteAllowed
		}
		if !allowed {
			return fmt.Errorf(`permission denied for recursive deletion of directory %s for components %v`, path, denials)
		} else {
			err := plm.ms.Delete(ctx, path)
			if err != nil {
				return fmt.Errorf(`error deleting "%s": %w`, path, err)
			}
			return nil
		}
	} else {
		// for non-recursive directory deletion, the directory needs to be empty
		children, err := plm.ms.GetChildren(ctx, path)
		if err != nil {
			return err
		}
		if len(children) != 0 {
			return fmt.Errorf(`directory %s is not empty, cannot be deleted without recursive flag`, path)
		}
		err = plm.ms.Delete(ctx, path)
		if err != nil {
			return err
		}
		return nil
	}
}

func (plm *PermissionLogicManager) GetACEs(ctx context.Context, path string, cred string) ([]*metadata.Annotation, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, err
	}
	visible, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return nil, err
	}

	if !visible {
		return nil, fmt.Errorf(`%s not found`, path)
	}
	return plm.getAces(ctx, path, callerID)
}

func (plm *PermissionLogicManager) getAces(ctx context.Context, path string, callerID string) ([]*metadata.Annotation, error) {
	p, err := plm.ms.Get(ctx, path, metadata.WithType("ace"))
	if err != nil {
		return nil, err
	}
	// copy the results and redact any invisible roles
	var tmp []*metadata.Annotation
	for _, perm := range p {
		ace, err := perm.AsACE()
		if err != nil {
			return nil, err
		}
		var tmpPerm []*metadata.ACL
		for _, acl := range ace.Acls {
			var tmpACL []string
			for _, role := range acl.Roles {
				flag, err := plm.checkPermissions(ctx, role, callerID, []metadata.Operation{metadata.View}, true)
				if err != nil {
					// This can't really happen because error only happens if caller doesn't exist.
					// We checked that up above.
					return nil, err
				}
				if flag {
					tmpACL = append(tmpACL, role)
				} else {
					tmpACL = append(tmpACL, common.RedactedRole)
				}
			}
			tmpPerm = append(tmpPerm, &metadata.ACL{Roles: tmpACL})
		}

		ace.Acls = tmpPerm
		ann, err := ace.AsAnnotation()
		if err != nil {
			return nil, err
		}
		tmp = append(tmp, ann)
	}
	return tmp, nil
}

func (plm *PermissionLogicManager) getUserAnnotations(ctx context.Context, path string) ([]*metadata.UserAnnotation, error) {
	p, err := plm.ms.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var tmp []*metadata.UserAnnotation
	for _, perm := range p {
		if perm.Tag != "ace" && perm.Tag != "applied-role" && perm.Tag != "dir" && perm.Tag != "leaf" {
			ua, err := perm.AsUserAnnotation()
			if err != nil {
				return nil, err
			}
			tmp = append(tmp, ua)
		}
	}
	return tmp, nil
}

func (plm *PermissionLogicManager) IsFolder(ctx context.Context, path, cred string) (bool, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return false, err
	}

	visible, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return false, err
	}
	if visible {
		return plm.ms.IsFolder(ctx, path)
	}
	return false, err
}

// ValidRoles checks to see if all of the elements of a slice are well-formed roles that exist in the metadata store.
func (plm *PermissionLogicManager) ValidateRoles(ctx context.Context, roles []string) error {
	for _, role := range roles {
		if !strings.HasPrefix(role, common.RolePrefix) {
			return fmt.Errorf("ill-formed role %s, must start with %s", role, common.RolePrefix)
		}

		annotations, err := plm.ms.Get(ctx, role, metadata.WithType("leaf"))
		if err != nil || len(annotations) == 0 {
			return fmt.Errorf("role %s does not exist %w", role, err)
		}
	}

	return nil
}

func (plm *PermissionLogicManager) GetChildren(ctx context.Context, path, cred string) ([]string, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, err
	}
	visible, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return nil, err
	}

	if visible {
		return plm.getChildren(ctx, path, callerID)
	} else {
		return nil, nil
	}
}

func (plm *PermissionLogicManager) getChildren(ctx context.Context, path string, callerID string) ([]string, error) {
	r, err := plm.ms.GetChildren(ctx, path)
	if err != nil {
		return nil, err
	}
	var children []string
	for _, child := range r {
		visible, err := plm.checkPermissions(ctx, child, callerID, []metadata.Operation{metadata.View}, true)
		if err != nil {
			return nil, err
		}
		if visible {
			children = append(children, child)
		}
	}
	return children, nil
}

func (plm *PermissionLogicManager) validateACERoles(ctx context.Context, ace *metadata.ACE) error {
	for _, acl := range ace.Acls {
		err := plm.ValidateRoles(ctx, acl.Roles)
		if err != nil {
			return err
		}
	}
	return nil
}

func (plm *PermissionLogicManager) UpdateACE(ctx context.Context, path string, perm *metadata.Annotation, cred string) (int64, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return 0, err
	}

	// TODO check
	if perm == nil {
		return 0, fmt.Errorf("nil ACE ... should never happen")
	}
	if perm.Tag != "ace" {
		return 0, fmt.Errorf("annotation is not an ACE, can't update it")
	}
	var ace *metadata.ACE
	if perm.Value == nil {
		ace = &metadata.ACE{}
		if err := anypb.UnmarshalTo(perm.Raw, ace, proto.UnmarshalOptions{}); err != nil {
			return 0, err
		}
	} else {
		ace = perm.Value.(*metadata.ACE)
	}

	if !strings.HasPrefix(path, common.StandardPrefix) {
		return 0, fmt.Errorf("ill-formed path %s, must start with %s", path, common.StandardPrefix)
	}

	if !strings.HasPrefix(callerID, common.WorkloadPrefix) && !strings.HasPrefix(callerID, common.UserPrefix) {
		return 0, fmt.Errorf("ill-formed caller path %s, must start with %s or %s", path, common.WorkloadPrefix, common.UserPrefix)
	}

	roles, err := plm.getEffectiveRoles(ctx, callerID)
	if err != nil {
		return 0, fmt.Errorf("caller %s does not exist in metadata store", callerID)
	}
	if err = plm.validateUsableRole(ctx, ace, roles, callerID); err != nil {
		return 0, fmt.Errorf("no right to use role in updated permission %w", err)
	}

	err = plm.validateACERoles(ctx, ace)
	if err != nil {
		return 0, fmt.Errorf("invalid role in update ACE %w", err)
	}

	// caller has to see the object being updated
	visible, _ := plm.checkRolePermissions(ctx, path, roles, []metadata.Operation{metadata.View}, true)
	if !visible {
		return 0, fmt.Errorf("principal %s not found", path)
	}

	// to change an ACE requires admin privileges on the path
	adminAllowed, _ := plm.checkRolePermissions(ctx, path, roles, []metadata.Operation{metadata.Admin}, true)
	if !adminAllowed {
		return 0, errors.New("principal " + callerID + " does not have permission to administer " + path)
	}

	// we are good to go based on path level permissions
	oldPermissions, err := plm.ms.Get(ctx, path, metadata.WithType("ace"))
	if err != nil {
		return 0, err
	}

	if perm.Unique == 0 {
		perm.Unique = common.SafeUnique()
	}

	// check to see if we are updating or adding a permission
	for _, p := range oldPermissions {
		if p.Unique == perm.Unique {
			// ... update, verify we can munch existing roles
			oldAce := p.Value.(*metadata.ACE)
			if oldAce == nil {
				oldAce = &metadata.ACE{}
				if err := anypb.UnmarshalTo(perm.Raw, oldAce, proto.UnmarshalOptions{}); err != nil {
					return 0, err
				}
			}
			if err := plm.validateUsableRole(ctx, oldAce, roles, callerID); err != nil {
				return 0, fmt.Errorf("unable to modify old permission due to role permissions %w", err)
			}

			perm.StartMillis = time.Now().UnixMilli()
			return plm.ms.Annotate(ctx, path, perm)
		}
	}

	// we need to add to existing permissions with a new unique key
	perm.StartMillis = time.Now().UnixMilli()
	perm.Unique = common.SafeUnique()

	return plm.ms.Annotate(ctx, path, perm)
}

// validateExistingPermissions function compares the ACLs of old and new permissions at their respective indices.
// it ensures that if a role is deleted from an ACL, the user must possess the "useRole" permission for the deleted role.
// additionally, if a new role is added to an ACL, it verifies whether the user has the "useRole" permission for all the
// roles within that ACL. Consequently, a user can remove a role, but reinstating it may be restricted if they lack "useRole"
// permissions for all the roles within the ACL.
//
//nolint:unused
func (plm *PermissionLogicManager) validateExistingPermissions(
	ctx context.Context,
	oldP, newP *metadata.ACE,
	roles []*metadata.Annotation,
) error {
	// TODO check
	for i := 0; i < len(oldP.Acls) || i < len(newP.Acls); i++ {
		var deletedRolesToCheck []string
		var aclToCheck *metadata.ACL

		if i < len(oldP.Acls) && i < len(newP.Acls) {
			r1 := mapset.NewSet[string]()
			for _, rx := range oldP.Acls[i].Roles {
				r1.Add(rx)
			}
			r2 := mapset.NewSet[string]()
			for _, rx := range newP.Acls[i].Roles {
				r2.Add(rx)
			}

			deletedRolesToCheck = append(deletedRolesToCheck, r1.Difference(r2).ToSlice()...)
			addedRoles := r2.Difference(r1).ToSlice()
			// if there is any new role added into an existing ACL, we have to validate
			// if the user has UseRole permission to that full ACL
			if len(addedRoles) > 0 {
				aclToCheck = newP.Acls[i]
			}
		} else if i < len(oldP.Acls) { // An ACL exist in old but not in new
			deletedRolesToCheck = append(deletedRolesToCheck, oldP.Acls[i].Roles...)
		} else if i < len(newP.Acls) { // An ACL exist in new but not in old
			aclToCheck = newP.Acls[i]
		}

		// Check if the user has the UseRole perm on deleted roles
		for _, r := range deletedRolesToCheck {
			useAllowed, _ := plm.checkRolePermissions(ctx, r, roles, []metadata.Operation{metadata.UseRole}, true)
			if !useAllowed {
				// in an existing permission, an unusable role was added or removed
				return fmt.Errorf("cannot use role %s", r)
			}
		}

		// Check if the user has the UseRole perm on the modified ACL
		if aclToCheck != nil {
			for _, r := range aclToCheck.Roles {
				useAllowed, _ := plm.checkRolePermissions(ctx, r, roles, []metadata.Operation{metadata.UseRole}, true)
				if !useAllowed {
					// in an existing permission, an unusable role was added or removed
					return fmt.Errorf("cannot use role %s", r)
				}
			}
		}
	}
	return nil
}

func (plm *PermissionLogicManager) GetRoles(ctx context.Context, path, cred string) ([]*metadata.Annotation, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, err
	}
	// first verify that the caller can see the path in question
	flag, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, fmt.Errorf("%s not found", path)
	}
	return plm.getRoles(ctx, path, callerID)
}

func (plm *PermissionLogicManager) getRoles(ctx context.Context, path string, callerID string) ([]*metadata.Annotation, error) {
	roles, err := plm.ms.Get(ctx, path, metadata.WithType("applied-role"))
	if err != nil {
		return nil, err
	}

	err = plm.redactRoles(ctx, roles, callerID)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (plm *PermissionLogicManager) GetAllRoles(ctx context.Context, path, cred string) ([]string, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, err
	}

	// first verify that the caller can see the path in question
	flag, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, fmt.Errorf("%s not found", path)
	}

	if !strings.HasPrefix(path, common.RolePrefix) {
		return nil, errors.New("invalid path, must be am://role/*")
	}

	roles, err := plm.getAllChildren(ctx, path, false)
	if err != nil {
		return nil, err
	}
	for i, role := range roles {
		flag, err = plm.checkPermissions(ctx, role, callerID, []metadata.Operation{metadata.View}, true)
		if err != nil {
			// This can't really happen because error only happens if caller doesn't exist.
			// We checked that up above.
			return nil, err
		}
		if !flag {
			// mutation is OK because we own the annotation structure
			roles[i] = common.RedactedRole
		}
	}

	return roles, nil
}

// Recursive function to get all children and children of children in top-down order.
func (plm *PermissionLogicManager) getAllChildren(ctx context.Context, path string, onlyDirs bool) ([]string, error) {
	return plm.getAllChildrenHelper(ctx, path, onlyDirs, nil)
}

func (plm *PermissionLogicManager) getAllChildrenHelper(ctx context.Context, path string, onlyDirs bool, children []string) ([]string, error) {
	c, err := plm.ms.GetChildren(ctx, path)
	if err != nil {
		return nil, err
	}
	if onlyDirs {
		children = append(children, path)
	} else {
		children = append(children, c...)
	}

	for _, child := range c {
		children, err = plm.getAllChildrenHelper(ctx, child, onlyDirs, children)
		if err != nil {
			return nil, err
		}
	}
	return children, nil
}

func (plm *PermissionLogicManager) GetInheritedRoles(
	ctx context.Context,
	path, cred string,
) ([]*metadata.Annotation, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, err
	}

	roles, err := plm.getEffectiveRoles(ctx, path)
	if err != nil {
		return nil, err
	}

	err = plm.redactRoles(ctx, roles, callerID)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

// redactRoles goes through a list of roles stored as annotations and checks that the caller
// is allowed to view each role. If not, the name of the role is replaced by a special string
func (plm *PermissionLogicManager) redactRoles(ctx context.Context, roles []*metadata.Annotation, callerID string) error {
	for _, rx := range roles {
		role, err := rx.AsAppliedRole()
		if err != nil {
			return err
		}
		flag, err := plm.checkPermissions(ctx, role.Role, callerID, []metadata.Operation{metadata.View}, true)
		if err != nil {
			// This can't really happen because error only happens if caller doesn't exist.
			// We checked that up above.
			return err
		}
		if !flag {
			// mutation is OK because we own the annotation structure
			rx.Value.(*metadata.RolePersist).Path = common.RedactedRole
			rx.Raw = nil
		}
	}
	return nil
}

// UpdateAppliedRole creates or updates an existing applied role. Updating really just means
// to change the end time.
func (plm *PermissionLogicManager) UpdateAppliedRole(
	ctx context.Context,
	appliedRole *metadata.Annotation,
	path, cred string,
) (int64, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return 0, err
	}

	// only principals have roles
	if !strings.HasPrefix(path, common.UserPrefix) && !strings.HasPrefix(path, common.WorkloadPrefix) {
		return 0, fmt.Errorf("invalid URI, must be %s/* or %s/*", common.UserPrefix, common.WorkloadPrefix)
	}

	roles, err := plm.getEffectiveRoles(ctx, callerID)
	if err != nil {
		return 0, err
	}

	// can our caller even see the principal in question?
	flag, _ := plm.checkRolePermissions(ctx, path, roles, []metadata.Operation{metadata.View}, true)
	if !flag {
		return 0, errors.New("path: " + path + " not found")
	}
	// we can see it, but can we muck with it?
	flag, _ = plm.checkRolePermissions(ctx, path, roles, []metadata.Operation{metadata.Admin}, true)
	if !flag {
		return 0, errors.New("not allowed to administer " + path)
	}
	// we are allowed to see and admin the roles on this path

	// grab the role we are updating in more convenient form
	role, err := appliedRole.AsAppliedRole()
	if err != nil {
		return 0, err
	}

	if err := plm.ValidateRoles(ctx, []string{role.Role}); err != nil {
		return 0, err
	}

	// verify that we have permissions on that role
	flag, _ = plm.checkRolePermissions(ctx, role.Role, roles, []metadata.Operation{metadata.ApplyRole}, true)
	if !flag {
		return 0, fmt.Errorf("not allowed to apply role %s", role.String())
	}

	// now we can get down to the real work
	existingAppliedRoles, err := plm.ms.Get(ctx, path, metadata.WithType("role"))
	if err != nil {
		return 0, err
	}

	// make sure we have a unique
	if appliedRole.Unique == 0 {
		appliedRole.Unique = common.SafeUnique()
	}

	// unique collisions are bad stuff
	// this is, of course, crazy unlikely but users do crazy stuff
	for _, existingRole := range existingAppliedRoles {
		if existingRole.Unique == appliedRole.Unique {
			rx, err := existingRole.AsAppliedRole()
			if err != nil {
				return 0, err
			}
			if rx.Role != role.Role {
				// strange collision ... let's avoid the question cowboy style
				return 0, fmt.Errorf("unique-ifier collision detected %s vs %s",
					rx.Role, role.Role)
			}
			break
		}
	}

	// now we look for this role already there
	for _, existingRole := range existingAppliedRoles {
		rx, err := existingRole.AsAppliedRole()
		if err != nil {
			return 0, err
		}
		if rx.Role == role.Role {
			// re-use existing unique
			existingRole.EndMillis = appliedRole.EndMillis
			existingRole.Version = appliedRole.Version
			v, err := plm.ms.Annotate(ctx, path, appliedRole)
			if err != nil {
				return 0, fmt.Errorf("unable to update role %s: %w", role, err)
			}
			return v, nil
		}
	}

	// Create ... will fail if our caller doesn't have version == 0
	appliedRole.StartMillis = time.Now().UnixMilli() //nolint:gosec
	return plm.ms.Annotate(ctx, path, appliedRole)
}

func (plm *PermissionLogicManager) GetDatasetCredential(
	ctx context.Context,
	path string,
	ops []metadata.Operation,
	cred string,
) (string, string, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return "", "", err
	}

	logger.GetLogger().Info().Ctx(ctx).Msg("verifying dataset and caller permissions")

	callerRoles, err := plm.getEffectiveRoles(ctx, callerID)
	if err != nil {
		return "", "", err
	}
	flag, err := plm.checkRolePermissions(ctx, path, callerRoles, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return "", "", err
	}
	if !flag {
		return "", "", fmt.Errorf("principal %s not found", path)
	}
	flag, err = plm.checkRolePermissions(ctx, path, callerRoles, ops, true)
	if err != nil {
		return "", "", err
	}
	if !flag {
		return "", "", fmt.Errorf("%v not allowed on %s", ops, path)
	}

	// get the JSON-encoded description of this particular dataset
	td, err := plm.ms.Get(ctx, path, metadata.WithType("data-info"))
	if err != nil {
		return "", "", fmt.Errorf("error getting data-info attribute for %s: %w", path, err)
	}
	if len(td) == 0 {
		return "", "", fmt.Errorf("no dataset info annotation for %s", path)
	}
	dataset, err := td[0].AsUserAnnotation()
	if err != nil {
		return "", "", err
	}
	var info DataInfo
	if err := protojson.Unmarshal([]byte(dataset.Data), &info); err != nil {
		return "", "", err
	}

	if len(info.DelegatedAttributes) == 0 {
		// if not a delegation asset, walk up the tree to find the first
		// inherited instance of a credential provider
		pieces := append(common.PathComponents(path), len(path))
		var provider *metadata.Annotation
		for i := len(pieces) - 1; i >= 0; i-- {
			tmpPath := path[0:pieces[i]]
			t, err := plm.ms.Get(ctx, tmpPath, metadata.WithType("data-credential-provider"))
			if err != nil {
				return "", "", err
			}
			if len(t) > 0 {
				provider = t[0]
				break
			}
		}
		if provider == nil {
			return "", "", fmt.Errorf("no credential provider found for %s", path)
		}
		// TODO generate access log record
		providerInfo, err := provider.AsUserAnnotation()
		if err != nil {
			return "", "", err
		}

		// TODO generate real access credential by calling provider
		return providerInfo.Data, dataset.Data, nil
	} else {
		// generate delegation token as JWT
		// TODO generate access log record
		delegatedAttributes := make([]string, 0)
		for _, r := range callerRoles {
			rx, err := r.AsAppliedRole()
			if err != nil {
				return "", "", err
			}
			if slices.Contains(info.DelegatedAttributes, rx.Role) {
				delegatedAttributes = append(delegatedAttributes, rx.Role)
			}
		}
		r, err := plm.ms.GetSignedJWTWithClaims(30*time.Minute, metadata.DefaultClaims(callerID, delegatedAttributes))
		if err != nil {
			return "", "", err
		}
		return r, "", nil
	}
}

// validateUsableRole confirms that we are allowed to use all the permissions in oldP
func (plm *PermissionLogicManager) validateUsableRole(
	ctx context.Context,
	perm *metadata.ACE,
	roles []*metadata.Annotation,
	cred string,
) error {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return err
	}
	for _, rx := range allRoles(perm) {
		visible, _ := plm.checkRolePermissions(ctx, rx, roles, []metadata.Operation{metadata.View}, true)
		if !visible {
			return errors.New("invalid role")
		}
		flag, _ := plm.checkRolePermissions(ctx, rx, roles, []metadata.Operation{metadata.UseRole}, true)
		if !flag {
			return errors.New("principal " + callerID + " does does not have admin permission for role " + rx)
		}
	}
	return nil
}

func (plm *PermissionLogicManager) getEffectiveRoles(ctx context.Context, path string) ([]*metadata.Annotation, error) {
	if !strings.HasPrefix(path, common.UserPrefix) && !strings.HasPrefix(path, common.WorkloadPrefix) {
		return nil, fmt.Errorf("invalid path for principal: %s", path)
	}

	// accumulate all roles for our caller from prefix paths
	roles := []*metadata.Annotation{}
	for _, offset := range common.PathComponents(path)[1:] {
		r, err := plm.ms.Get(ctx, path[0:offset], metadata.WithType("applied-role"))
		if err != nil {
			return nil, err
		}
		roles = append(roles, r...)
	}
	return roles, nil
}

/*
allRoles accumulates nested roles inside a ACE as a slice
*/
func allRoles(p *metadata.ACE) []string {
	r := []string{}
	for _, acl := range p.Acls {
		r = append(r, acl.Roles...)
	}
	return r
}

/*
checkPermissions compares the permissions on a `uri` against the `roles` to see if the specified `ops`
are all allowed. If `includeLast` is true, all components of the uri are checked, if not, all but the
last are checked. This is useful, for instance, when creating an element since we can't check permissions
on something that doesn't exist yet.
*/
func (plm *PermissionLogicManager) checkPermissions(
	ctx context.Context,
	path, caller string,
	ops []metadata.Operation,
	includeLast bool,
) (bool, error) {
	if !strings.HasPrefix(path, "am://") {
		return false, fmt.Errorf("invalid path %s, must start with 'am://'", path)
	}
	roles, err := plm.getEffectiveRoles(ctx, caller)
	if err != nil {
		return false, fmt.Errorf("bad caller: %w", err)
	}

	return plm.checkRolePermissions(ctx, path, roles, ops, includeLast)
}

/*
checkRolePermissions tests to see if a specified `uri` has permissions compatible with a list of `roles` for
all of a list of `operations` given. The permissions on `uri` include all of its inherited constraints.
*/
func (plm *PermissionLogicManager) checkRolePermissions(
	ctx context.Context,
	path string,
	roles []*metadata.Annotation,
	operations []metadata.Operation,
	includeLast bool,
) (bool, error) {
	permsOk := true
	checker := func(_ string, ace *metadata.ACE, done error) error {
		if slices.Contains(operations, ace.Op) {
			allowed, err := checkPermission(ace, roles)
			if err != nil {
				return err
			}
			if !allowed {
				permsOk = false
				return done
			}
		}
		return nil
	}

	err := plm.ms.ScanPath(ctx, path, includeLast, checker)
	return permsOk, err
}

func (plm *PermissionLogicManager) Exists(ctx context.Context, path string, cred string) (bool, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return false, err
	}

	r, err := plm.ms.Get(ctx, path, metadata.WithType("dir"), metadata.WithType("leaf"))
	if err != nil {
		if strings.HasSuffix(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	exists := len(r) > 0
	if exists {
		flag, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
		if err != nil {
			return false, err
		}
		// "not allowed to see" is just like "doesn't exist"
		if !flag {
			return false, nil
		}
	}
	return exists, nil
}

func (plm *PermissionLogicManager) GetAnnotations(
	ctx context.Context,
	path string,
	cred string,
	filters ...metadata.KeyOption,
) ([]*metadata.Annotation, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, err
	}

	flag, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return nil, err
	}
	// "not allowed to see" is just like "doesn't exist"
	if !flag {
		return nil, fmt.Errorf("object not found for %s", path)
	}

	return plm.getAnnotations(ctx, path, filters...)
}

func (plm *PermissionLogicManager) getAnnotations(ctx context.Context, path string, filters ...metadata.KeyOption) ([]*metadata.Annotation, error) {
	rx, err := plm.ms.Get(ctx, path, filters...)
	if err != nil {
		return nil, err
	}

	return rx, nil
}

/*
checkPermission tests `perm` to see if it is satisfied by a list of roles
*/
func checkPermission(perm *metadata.ACE, roles []*metadata.Annotation) (bool, error) {
	roleSet := mapset.NewSet[string]()
	for _, role := range roles {
		rx, err := role.AsAppliedRole()
		if err != nil {
			return false, err
		}
		roleSet.Add(rx.Role)
	}

	for _, acl := range perm.Acls {
		if mapset.NewSet(acl.Roles...).Intersect(roleSet).Cardinality() == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (plm *PermissionLogicManager) GetDetails(ctx context.Context, path string, includeChildren bool, cred string) (*NodeDetails, []string, error) {
	callerID, err := plm.checkCredential(ctx, cred)
	if err != nil {
		return nil, nil, err
	}

	flag, err := plm.checkPermissions(ctx, path, callerID, []metadata.Operation{metadata.View}, true)
	if err != nil {
		return nil, nil, err
	}
	if !flag {
		// we only bail out if we are not allowed to see the top-level
		return nil, nil, fmt.Errorf("object not found for %s", path)
	}

	details, err := plm.getDetails(ctx, path, callerID)
	if err != nil {
		return nil, nil, err
	}

	var children []string
	if includeChildren {
		children, err = plm.getChildren(ctx, path, callerID)
		if err != nil {
			return nil, nil, err
		}
	}
	return details, children, nil
}

func (plm *PermissionLogicManager) getDetails(ctx context.Context, path, caller string) (*NodeDetails, error) {
	roles, aces, _, err := plm.getLocalDetails(ctx, path, caller)
	if err != nil {
		return nil, err
	}

	inheritedRoles := []*metadata.AppliedRole{}
	inheritedAces := []*metadata.ACE{}
	pathComponents := common.PathComponents(path)
	for _, i := range pathComponents[:len(pathComponents)-1] {
		r, a, _, err2 := plm.getLocalDetails(ctx, path[:i], caller)
		if err2 != nil {
			return nil, err2
		}
		inheritedRoles = append(inheritedRoles, r...)
		for _, ax := range a {
			if !ax.Local {
				inheritedAces = append(inheritedAces, ax)
			}
		}
	}

	ax, err := plm.getAnnotations(ctx, path)
	if err != nil {
		return nil, err
	}

	annotations := []*metadata.UserAnnotation{}
	for _, a := range ax {
		if a.Tag != "applied-role" && a.Tag != "ace" && a.Tag != "dir" && a.Tag != "leaf" {
			ua, err := a.AsUserAnnotation()
			if err != nil {
				return nil, err
			}
			annotations = append(annotations, ua)
		}
	}

	allowChildren, err := plm.ms.IsFolder(ctx, path)
	if err != nil {
		return nil, err
	}
	return &NodeDetails{
		Path:           path,
		Roles:          roles,
		InheritedRoles: inheritedRoles,
		Aces:           aces,
		InheritedAces:  inheritedAces,
		IsDirectory:    allowChildren,
		Annotations:    annotations,
	}, nil
}

func (plm *PermissionLogicManager) getLocalDetails(ctx context.Context, path, caller string) ([]*metadata.AppliedRole, []*metadata.ACE, []*metadata.UserAnnotation, error) {
	r, err := plm.getRoles(ctx, path, caller)
	if err != nil {
		return nil, nil, nil, err
	}

	roles := []*metadata.AppliedRole{}
	for _, role := range r {
		r, err := role.AsAppliedRole()
		if err != nil {
			return nil, nil, nil, err
		}
		roles = append(roles, r)
	}

	rawAces, err := plm.getAces(ctx, path, caller)
	if err != nil {
		return nil, nil, nil, err
	}
	aces := []*metadata.ACE{}
	for _, ann := range rawAces {
		ace, err := ann.AsACE()
		if err != nil {
			return nil, nil, nil, err
		}
		aces = append(aces, ace)
	}

	userAnnotations, err := plm.getUserAnnotations(ctx, path)
	if err != nil {
		return nil, nil, nil, err
	}
	return roles, aces, userAnnotations, nil
}

func (plm *PermissionLogicManager) Bootstrap(bootstrap string, key string) error {
	log.Info().Msg("starting bootstrap attempt")
	tx, err := metadata.SampleMetaTree(bootstrap)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("unable to read bootstrap metadata")
		return fmt.Errorf("unable to read bootstrap metadata: %w", err)
	}
	if err := plm.ms.PutTree(context.Background(), tx); err != nil {
		if ex, ok := err.(metadata.TransactionFailure); ok {
			logger.GetLogger().Info().Msg(ex.Msg)
			return fmt.Errorf("data already exists: %w", err)
		} else {
			logger.GetLogger().Err(err).Msg("unable to insert bootstrap metadata")
			return fmt.Errorf("unable to insert bootstrap metadata: %w", err)
		}
	}

	if key != "" {
		keyAnnotation, err := metadata.UserAnnotationString("ssh-pubkey", key)
		if err != nil {
			logger.GetLogger().Err(err).Msg("unable to create ssh key annotation")
			return fmt.Errorf("unable to create ssh key annotation: %w", err)
		}
		_, err = plm.ms.Annotate(context.Background(), "am://user/the-operator", keyAnnotation)
		if err != nil {
			logger.GetLogger().Err(err).Msg("unable to add ssh key")
			return fmt.Errorf("unable to add ssh key to operator: %w", err)
		}
	}
	return nil
}
