/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	v3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"
	"math/big"
	rnd "math/rand/v2"
)

// MetaStore allows access to annotations and keys. This version hides all
// implementation details and is what is advertised.
type MetaStore interface {
	AnnotationStore
	KeyStore
}

// MetaStoreX is a MetaStore which uses unexported implementations so that
// internal tests can mess with things like time slipping or see through to
// the underlying persistence interface. Not advertised for normal use and
// only available via OpenTestStore
type MetaStoreX struct {
	annotationStore
	keyStore
}

// OpenEtcdMetaStore connects to etcd and sets up a production style metadata store
func OpenEtcdMetaStore(urls []string) (MetaStore, error) {
	c, err := v3.NewFromURLs(urls)
	if err != nil {
		return nil, err
	}

	return newMetaStore(
		&etcAnnoStore{namespace.NewKV(c, "/meta")},
		&etcKeyStore{meta: namespace.NewKV(c, "/key")},
	)
}

// OpenTestStore creates an in-memory metadata store useful for testing. The
// data parameter is the name of an embedded file that contains a
// MetaTree in JSON format. This is used to bootstrap the store with some
// known data. The file is expected to be in the "metadata/sample" directory.
func OpenTestStore(data string) (*MetaStoreX, error) {
	r, err := newMetaStore(
		&testAnnoStore{tbl: make(map[string]vType)},
		&testKeyStore{tbl: make(map[string]vType)},
	)

	tree, err := SampleMetaTree(data)
	if err != nil {
		return nil, err
	}
	tree.adjust()
	if err := r.PutTree(context.Background(), tree); err != nil {
		return nil, fmt.Errorf("insertion of test data failed: %w", err)
	}

	return r, nil
}

/*
adjust modifies the data in a MetaTree to be more realistic. This involves
munging the `unique` flag in ACEs and setting versions to realistic values. Any
values that are already set to non-zero values will be unmolested.
*/
func (t *MetaTree) adjust() {
	uniqueTable := map[int64]int{}
	uniqueTable[0] = 1
	adjust_helper(t, uniqueTable)
}

func adjust_helper(t *MetaTree, uniqueTable map[int64]int) {
	for _, m := range t.Meta {
		if m.Unique == 0 {
			m.Unique = common.SafeUnique()
		}
		_, taken := uniqueTable[m.Unique]
		for taken {
			m.Unique = common.SafeUnique()
			_, taken = uniqueTable[m.Unique]
		}
		uniqueTable[m.Unique] = 1
		if m.Version == 0 {
			// this magic number is just to generate reasonable test data
			//nolint:gosec,mnd
			m.Version = 2 + rnd.Int64N(4)
		}
		if m.Version <= 1 {
			m.Version = 2
		}
	}
	for _, child := range t.Children {
		adjust_helper(child, uniqueTable)
	}
}

// newMetaStore is the common glue between test and production implementations of MetaStore
func newMetaStore(ms rawAnnoStore, ks rawKeyStore) (*MetaStoreX, error) {
	mask := big.NewInt(0)
	mask.SetUint64(1<<64 - 1)
	id, err := rand.Int(rand.Reader, mask)
	if err != nil {
		return nil, err
	}
	r := &MetaStoreX{
		annotationStore: annotationStore{meta: ms},
		keyStore: keyStore{
			internalId: fmt.Sprintf("internal-%016x", id.Uint64()),
			keys:       ks,
		},
	}
	return r, nil
}
