/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"context"
	"errors"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"github.com/hpe/access-manager/pkg/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	v3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AnnotationStore interface {
	// Annotate adds an annotation to an existing node
	Annotate(ctx context.Context, path string, ann Notable) (int64, error)

	// ScanPath walks the metadata tree from root to `path` and calls the checker
	// function at each prefix. If the checker returns the sentinel as the error,
	// the scan will stop. It is expected that the checker will have a side effect
	// that achieves the goal of the scan; the scanner does not do anything but
	// provide a control flow.
	ScanPath(ctx context.Context, path string, includeLast bool, checker func(path string, ace *ACE, abortScan error) error) error

	// PutNode creates a node or updates one
	PutNode(ctx context.Context, path string, version int64) (int64, error)

	// PutTree writes and entire tree or subtree for bootstrapping or testing.
	// An error should be thrown if the data already exists
	PutTree(ctx context.Context, tree *MetaTree) error

	// GetTree serializes the entire metadata state in a form suitable for preserving
	GetTree(ctx context.Context) (*MetaTree, error)

	// Get reads the metadata for a path with an optional filter. Filters are of the form
	// `metaType/subType` or `metaType` and only metadata annotations that matches at
	// least one of the filters will be returned. If no filters are given, then all
	// metadata annotations are returned.
	Get(ctx context.Context, path string, filters ...KeyOption) ([]*Annotation, error)

	// GetChildren returns the list of the names of the children of the current node
	GetChildren(ctx context.Context, path string) ([]string, error)

	// Delete deletes an object along with all children and annotations
	Delete(ctx context.Context, path string) error

	// DeleteAnnotation deletes a single annotation. Version can be -1 to force the deletion, but tag and unique must be valid
	DeleteAnnotation(ctx context.Context, path string, tag string, version int64, unique int64) error

	// IsFolder returns true if a path names a directory, false if it names a leaf
	// node like a role or a user. An error is thrown if nothing exists with that
	// name.
	IsFolder(ctx context.Context, path string) (bool, error)
}

type annotationStore struct {
	meta rawAnnoStore // handles low-level metadata operations
}

func (e *annotationStore) Annotate(ctx context.Context, path string, ax Notable) (int64, error) {
	if ax == nil {
		return 0, fmt.Errorf("nil annotation makes no sense")
	}

	ann, err := ax.AsAnnotation()
	if err != nil {
		return -1, err
	}

	if ann.Unique == 0 {
		ann.Unique = common.SafeUnique()
	}

	// overwrite existing or create new record
	key, err := buildKey(path, WithUnique(ann.Unique), WithType(ann.Tag))
	var value *anypb.Any
	if ann.Raw == nil {
		if ann.Value != nil {
			value = &anypb.Any{}
			if err := anypb.UnmarshalTo(value, ann.Value, proto.UnmarshalOptions{}); err != nil {
				return 0, err
			}
		}
	} else {
		value = ann.Raw
	}

	// don't fill version and unique here because they come from the
	// storage layer and the key, respectively
	wrapped := AnnotationWrapper{
		StartMillis: ann.StartMillis,
		EndMillis:   ann.EndMillis,
		Tag:         ann.Tag,
		Raw:         value,
	}

	buf, err := proto.Marshal(&wrapped)
	if err != nil {
		return 0, err
	}

	return e.meta.putWithVersion(ctx, key, buf, ann.Version)
}

func (e *annotationStore) ScanPath(ctx context.Context, path string, includeLast bool, checker func(path string, ace *ACE, abortScan error) error) error {
	done := errors.New("marker")
	components := common.PathComponents(path)
	n := len(components)
	if !includeLast {
		n -= 1
	}
	// the effective permissions for a Path is computed by examining all the inherited
	// constraints
	if n >= 1 {
		for i, step := range components[0:n] {
			p := path[0:step]
			perms, err := e.Get(ctx, p, WithType("ace"))
			if err != nil {
				return err
			}
			if perms == nil {
				break
			}
			for _, perm := range perms {
				ace, err := perm.AsACE()
				if err != nil {
					logger.GetLogger().Error().Msgf("failed to unmarshal permission for %s", path)
					return nil
				}
				// local permissions are not inherited, but are applied at last step
				if !ace.Local || i == len(components)-1 {
					e := checker(p, ace, done)
					if e == done {
						return nil
					}
					if e != nil {
						return e
					}
				}
			}
		}
	}
	return nil
}

func (e *annotationStore) PutNode(ctx context.Context, path string, version int64) (int64, error) {
	parent := common.Parent(path)
	ok, err := e.IsFolder(ctx, parent)
	if err != nil {
		return 0, fmt.Errorf("parent of %s doesn't exist or is not a folder: %w", path, err)
	}
	if !ok {
		return 0, fmt.Errorf("parent directory %s does not exist", parent)
	}
	key, err := buildKey(path)
	if err != nil {
		return 0, err
	}
	return e.meta.putWithVersion(ctx, key, nil, version)
}

func (e *annotationStore) PutTree(ctx context.Context, tree *MetaTree) error {
	err := tree.Walk(TOP_DOWN, func(tx *MetaTree) error {
		// lay down the node itself
		_, err := e.PutNode(ctx, tx.Path, 0)
		if err != nil {
			return err
		}
		// and then add annotations
		for _, annotation := range tx.Meta {
			if annotation.Raw == nil {
				if annotation.Value != nil {
					err = anypb.MarshalFrom(annotation.Raw, annotation.Value, proto.MarshalOptions{})
					if err != nil {
						return err
					}
				}
			}

			annotation.Version = -1
			_, err = e.Annotate(ctx, tx.Path, annotation)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (e *annotationStore) GetTree(ctx context.Context) (*MetaTree, error) {
	r := &MetaTree{Path: common.StandardPrefix}
	err := build(e, ctx, r, common.StandardPrefix)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (e *annotationStore) IsFolder(ctx context.Context, path string) (bool, error) {
	if path == common.StandardPrefix {
		return true, nil
	}
	m, err := e.Get(ctx, path, WithType("leaf"))
	if err != nil {
		return false, fmt.Errorf("error getting %s: %w", path, err)
	}
	if len(m) > 0 {
		// found leaf annotation, not a directory
		return false, nil
	} else {
		return true, nil
	}
}

func (e *annotationStore) Delete(ctx context.Context, path string) error {
	key, err := buildKey(path)
	if err != nil {
		return err
	}
	err = e.meta.deleteWithPrefix(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

func (e *annotationStore) DeleteAnnotation(ctx context.Context, path string, tag string, version int64, unique int64) error {
	key, err := buildKey(path, WithType(tag), WithUnique(unique))
	if err != nil {
		return err
	}

	if version == 0 {
		return fmt.Errorf("can't delete with version = 0")
	}
	err = e.meta.deleteWithVersion(ctx, key, version)
	if err != nil {
		return err
	}
	return nil
}

func (e *annotationStore) GetChildren(ctx context.Context, path string) ([]string, error) {
	var childPattern = regexp.MustCompile(`^[[:xdigit:]] ([^#]+)$`)
	childKey, err := buildKey(path, WithDepthOffset(1))
	if err != nil {
		return nil, err
	}
	rows, err := e.meta.getWithPrefix(ctx, childKey)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0)
	for _, r := range rows {
		match := childPattern.FindStringSubmatch(string(r.Key))
		if match != nil {
			values = append(values, match[1])
		}
	}
	return values, nil
}

func build(e AnnotationStore, ctx context.Context, m *MetaTree, path string) error {
	annotations, err := e.Get(ctx, path, WithType("ace"), WithType("applied-role"))
	if err != nil {
		return err
	}
	m.Meta = annotations
	if dir, err := e.IsFolder(ctx, path); err != nil {
		return err
	} else {
		if !dir {
			if strings.HasPrefix(path, common.WorkloadPrefix) || strings.HasPrefix(path, common.UserPrefix) {
				annotations = append(annotations, &Annotation{Tag: "principal"})
			} else if strings.HasPrefix(path, common.DataPrefix) {
				annotations = append(annotations, &Annotation{Tag: "data"})
			} else if strings.HasPrefix(path, common.RolePrefix) {
				annotations = append(annotations, &Annotation{Tag: "role"})
			} else if strings.HasPrefix(path, common.KeyPrefix) {
				annotations = append(annotations, &Annotation{Tag: "key"})
			} else {
				return fmt.Errorf("invalid top-level directory: %s", path)
			}
		}
	}

	rx, err := e.GetChildren(ctx, path)
	if err != nil {
		return err
	}
	for _, r := range rx {
		child := &MetaTree{Path: r}
		m.Children = append(m.Children, child)
		err := build(e, ctx, child, r)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get returns all of the metadata stored at a particular path in a raw form. The
// keys for the metadata returned is examined to ensure that it is in a valid
// format, but is otherwise uninterpreted.
func (e *annotationStore) Get(ctx context.Context, path string, filters ...KeyOption) ([]*Annotation, error) {
	if strings.Contains(path, " ") {
		return nil, fmt.Errorf("metadata Path cannot contain spaces")
	}
	key, err := buildKey(path)
	if err != nil {
		return nil, err
	}
	r, err := e.meta.getWithPrefix(ctx, key)
	if err != nil {
		return nil, err
	}
	result, err := parseValues(r, path, filters...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *annotationStore) Close() {
	e.Close()
}

func parseValues(r []*mvccpb.KeyValue, path string, filters ...KeyOption) ([]*Annotation, error) {
	result := make([]*Annotation, 0)

	t0 := time.Now()
	foundRoot := false
	for _, kv := range r {
		k := kv.Key
		bare, metaType, unique, err := parseKey(k, path)
		if err != nil {
			return nil, err
		}
		if bare {
			foundRoot = true
			continue
		}
		rz := AnnotationWrapper{}
		if err := proto.Unmarshal(kv.Value, &rz); err != nil {
			return nil, err
		}
		if rz.Tag != metaType {
			return nil, fmt.Errorf("invalid metadata type: %s (expected) vs %s (found)", metaType, rz.Tag)
		}
		if rz.EndMillis == 0 || time.UnixMilli(rz.EndMillis).After(t0) {
			rx := &Annotation{
				Unique:      unique,
				Version:     kv.Version,
				Global:      kv.ModRevision,
				Tag:         rz.Tag,
				Raw:         rz.Raw,
				StartMillis: rz.StartMillis,
				EndMillis:   rz.EndMillis,
				Value:       nil,
			}
			if len(filters) == 0 {
				result = append(result, rx)
			} else {
				uniqueSpecified := false
				uniqueMatched := false
				typeSpecified := false
				typeMatched := false
				for _, filter := range filters {
					switch filter.tag {
					case Unique:
						uniqueSpecified = true
						if rx.Unique == filter.unique {
							uniqueMatched = true
							continue
						}
					case MetaType:
						typeSpecified = true
						if rx.Tag == filter.metaType {
							typeMatched = true
							break
						}
					case DepthOffset:
						// don't care
					}
				}
				var k1, k2 = true, true
				if typeSpecified {
					k1 = typeMatched
				}
				if uniqueSpecified {
					k2 = uniqueMatched
				}
				if k1 && k2 {
					result = append(result, rx)
				}
			}
		}
	}
	if !foundRoot {
		return nil, fmt.Errorf("%s not found", path)
	}
	return result, nil
}

var keyPattern = regexp.MustCompile(`([[:xdigit:]]+) ([^#]+)(#([[:alpha:]\-]+)-([[:xdigit:]]+))?`)

func parseKey(k []byte, path string) (bool, string, int64, error) {
	match := keyPattern.FindStringSubmatch(string(k))
	if match == nil {
		return false, "", 0, fmt.Errorf("metadata key has invalid format %s", k)
	}
	if path != match[2] {
		return false, "", 0, fmt.Errorf("metadata Path does not match %s", path)
	}
	if match[3] == "" {
		return true, "", 0, nil
	} else {
		metaType := match[4]
		if metaType == "" {
			return false, "", 0, nil
		}
		unique, err := strconv.ParseInt(match[5], 0, 64)
		if err != nil {
			return false, "", 0, fmt.Errorf("metadata key has invalid uniqueifier format %s", match[3])
		}
		return false, metaType, unique, nil
	}
}

type KeyOption struct {
	tag         KeyOptionKind
	unique      int64
	metaType    string
	depthOffset int
}

type KeyOptionKind int

const (
	Unique KeyOptionKind = iota
	MetaType
	DepthOffset
)

func WithUnique(unique int64) KeyOption {
	return KeyOption{tag: Unique, unique: unique}
}

func WithType(metaType string) KeyOption {
	return KeyOption{tag: MetaType, metaType: metaType}
}

func WithDepthOffset(offset int) KeyOption {
	return KeyOption{tag: DepthOffset, depthOffset: offset}
}

func buildKey(path string, options ...KeyOption) (string, error) {
	format := "%d %s"
	offset := 0
	i := slices.IndexFunc(options, func(opt KeyOption) bool { return opt.tag == MetaType })
	if i >= 0 {
		j := slices.IndexFunc(options, func(opt KeyOption) bool { return opt.tag == Unique })
		if j >= 0 {
			format = fmt.Sprintf("%%d %%s#%s-%d", options[i].metaType, options[j].unique)
		} else {
			format = fmt.Sprintf("%%d %%s#%s-", options[i].metaType)
		}
	} else {
		j := slices.IndexFunc(options, func(opt KeyOption) bool { return opt.tag == Unique })
		if j >= 0 {
			return "", fmt.Errorf("should not have unique without type")
		}
	}
	i = slices.IndexFunc(options, func(opt KeyOption) bool { return opt.tag == DepthOffset })
	if i >= 0 {
		offset = options[i].depthOffset
	}

	depth := len(common.PathComponents(path)) - 1 + offset
	return fmt.Sprintf(format, depth, path), nil
}

// These are the minimal operations that we need from our data store
// they are broken out as an interface to allow the data store to be
// replaced for testing with minimal amount of untested code
type rawAnnoStore interface {
	putWithVersion(ctx context.Context, key string, value []byte, version int64) (int64, error)
	deleteWithVersion(ctx context.Context, key string, version int64) error
	deleteWithPrefix(ctx context.Context, key string) error
	getWithPrefix(ctx context.Context, prefix string) ([]*mvccpb.KeyValue, error)
}

type etcAnnoStore struct {
	meta v3.KV
}

func (e *etcAnnoStore) deleteWithPrefix(ctx context.Context, key string) error {
	response, err := e.meta.Delete(ctx, key, v3.WithPrefix())
	if err != nil {
		return err
	}
	if response.Deleted == 0 {
		return fmt.Errorf("no rows deleted")
	}
	return nil
}

func (e *etcAnnoStore) putWithVersion(ctx context.Context, key string, value []byte, version int64) (int64, error) {
	v := ""
	if value != nil {
		v = string(value)
	}

	if version == -1 {
		// unconditional write
		if response, err := e.meta.Put(ctx, key, v, v3.WithPrevKV()); err != nil {
			return 0, err
		} else {
			if response.PrevKv == nil {
				return 1, nil
			}
			return response.PrevKv.Version + 1, nil
		}
	} else if version == 0 {
		// create
		ctx1, cancel := context.WithTimeout(ctx, 10*time.Second)
		r, err := e.meta.Txn(ctx1).
			If(v3.Compare(v3.CreateRevision(key), "=", 0)).
			Then(v3.OpPut(key, v)).
			Commit()
		cancel()
		if err != nil {
			return 0, fmt.Errorf("failed to put metadata value: %w", err)
		}
		if !r.Succeeded {
			return 0, TransactionFailure{Msg: "data already exists", Path: key}
		}
		return 1, nil
	} else {
		// update existing version
		r, err := e.meta.Txn(ctx).
			If(v3.Compare(v3.Version(key), "==", version)).
			Then(v3.OpPut(key, v, v3.WithPrevKV())).
			Commit()
		if err != nil {
			return 0, err
		}
		if !r.Succeeded {
			return 0, TransactionFailure{Msg: "version mismatch on update", Path: key}
		}
		return r.Responses[0].GetResponsePut().PrevKv.Version + 1, nil
	}
}

func (e *etcAnnoStore) deleteWithVersion(ctx context.Context, key string, version int64) error {
	var (
		deleteCount int64
	)
	if version == -1 {
		response, err := e.meta.Delete(ctx, key)
		if err != nil {
			return err
		}
		deleteCount = response.Deleted
	} else {
		response, err := e.meta.Txn(ctx).
			If(v3.Compare(v3.Version(key), "=", version)).
			Then(v3.OpDelete(key)).
			Commit()
		if err != nil {
			return err
		}
		if !response.Succeeded {
			return TransactionFailure{Msg: "delete failed", Path: key}
		}
		deleteCount = response.Responses[0].GetResponseDeleteRange().Deleted
	}
	if deleteCount == 0 {
		return fmt.Errorf("could not delete %s", key)
	}

	return nil
}

func (e *etcAnnoStore) getWithPrefix(ctx context.Context, key string) ([]*mvccpb.KeyValue, error) {
	r, err := e.meta.Get(ctx, key, v3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if r.Count == 0 {
		return nil, nil
	}
	return r.Kvs, nil
}

type testAnnoStore struct {
	mu     sync.Mutex
	global int64
	tbl    map[string]vType
}

type vType struct {
	version int64
	value   []byte
}

func (t *testAnnoStore) putWithVersion(_ context.Context, key string, value []byte, version int64) (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if version == -1 {
		v, ok := t.tbl[key]
		if ok {
			v.version += 1
			v.value = value
			t.tbl[key] = v
			t.global += 1
		} else {
			v = vType{
				version: 1,
				value:   value,
			}
			t.tbl[key] = v
			t.global += 1
		}
		return v.version, nil
	} else if version == 0 {
		_, ok := t.tbl[key]
		if !ok {
			t.tbl[key] = vType{version: 1, value: value}
			return 1, nil
		} else {
			return 0, fmt.Errorf("metadata store already has value for %s", key)
		}
	} else {
		v, ok := t.tbl[key]
		if ok {
			if v.version != version {
				return 0, fmt.Errorf("version mismatch %d vs %d", version, v.version)
			}
			v.version += 1
			v.value = value
			t.tbl[key] = v
			t.global += 1
			return v.version, nil
		} else {
			return 0, fmt.Errorf("value does not exist, cannot update")
		}
	}
}

func (t *testAnnoStore) deleteWithPrefix(_ context.Context, key string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for k := range t.tbl {
		if strings.HasPrefix(k, key) {
			delete(t.tbl, k)
			count++
		}
	}
	if count == 0 {
		return fmt.Errorf("no rows deleted")
	}
	return nil
}

func (t *testAnnoStore) deleteWithVersion(_ context.Context, key string, version int64) error {
	if version == -1 {
		if _, ok := t.tbl[key]; ok {
			delete(t.tbl, key)
			return nil
		} else {
			return fmt.Errorf("no such element, cannot delete %s", key)
		}
	} else if version == 0 {
		return fmt.Errorf("cannot delete version 0")
	} else {
		v, ok := t.tbl[key]
		if !ok {
			return fmt.Errorf("no such element, cannot delete %s", key)
		}
		if v.version != version {
			return fmt.Errorf("version mismatch, cannot delete %s", key)
		}
		delete(t.tbl, key)
	}
	return nil
}

func (t *testAnnoStore) getWithPrefix(_ context.Context, key string) ([]*mvccpb.KeyValue, error) {
	// simulate etcd's returned values for prefix search
	values := make([]*mvccpb.KeyValue, 0)
	for k, v := range t.tbl {
		if strings.HasPrefix(k, key) {
			values = append(values, &mvccpb.KeyValue{
				Key:     []byte(k),
				Version: v.version,
				Value:   v.value,
			})
		}
	}
	return values, nil
}

type TransactionFailure struct {
	Msg  string
	Path string
}

func (t TransactionFailure) Error() string {
	return fmt.Sprintf("transaction failure on %s: %s", t.Path, t.Msg)
}
