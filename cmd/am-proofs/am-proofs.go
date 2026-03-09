/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/JonahRosenblumHPE/dafny-go-transpile/AMDatatypes"
	"github.com/JonahRosenblumHPE/dafny-go-transpile/MetaTreeFns"
	"github.com/JonahRosenblumHPE/dafny-go-transpile/MetaTreeInvariants"
	"github.com/JonahRosenblumHPE/dafny-go-transpile/dafny"
	"github.com/hpe/access-manager/internal/services/metadata"
)

func am_tree_to_dafny_tree(m *metadata.MetaTree) (AMDatatypes.MetaTree, error) {
	var dafny_mtree AMDatatypes.MetaTree
	var roles []interface{}
	var aces []interface{}
	var err error

	for _, ann := range m.Meta {
		switch ann.Tag {
		case "ace":
			var dafny_ace AMDatatypes.Data_ACE_
			var dafny_op AMDatatypes.Operation

			ace, err := ann.AsACE()
			if err != nil {
				return dafny_mtree, err
			}

			switch ace.Op {
			case metadata.Operation_INVALID:
				return dafny_mtree, errors.New("invalid operation found in ace")
			case metadata.Operation_READ:
				dafny_op = AMDatatypes.Companion_Operation_.Create_Read_()
			case metadata.Operation_WRITE:
				dafny_op = AMDatatypes.Companion_Operation_.Create_Write_()
			case metadata.Operation_VIEW:
				dafny_op = AMDatatypes.Companion_Operation_.Create_View_()
			case metadata.Operation_ADMIN:
				dafny_op = AMDatatypes.Companion_Operation_.Create_Admin_()
			case metadata.Operation_USEROLE:
				dafny_op = AMDatatypes.Companion_Operation_.Create_UseRole_()
			case metadata.Operation_APPLYROLE:
				dafny_op = AMDatatypes.Companion_Operation_.Create_ApplyRole_()
			case metadata.Operation_VOUCHFOR:
				dafny_op = AMDatatypes.Companion_Operation_.Create_VouchFor_()
			default:
				return dafny_mtree, errors.New("unknown operation found in ace")

			}

			acls := make([]interface{}, len(ace.Acls))
			for acl_idx, acl := range ace.Acls {
				dafny_acl := make([]interface{}, len(acl.Roles))
				for role_idx, role := range acl.Roles {
					dafny_acl[role_idx] = role
				}
				acls[acl_idx] = dafny.SetOf(dafny_acl...)
			}

			dafny_ace = AMDatatypes.Companion_ACE_.Create_ACE_(dafny.SeqOf(acls...), dafny_op, ace.Local)
			aces = append(aces[:], dafny_ace)
		case "applied-role":
			role, err := ann.AsAppliedRole()
			if err != nil {
				return dafny_mtree, err
			}
			roles = append(roles[:], role.Role)
		}
	}
	if len(m.Children) == 0 {
		dafny_mtree = AMDatatypes.Companion_MetaTree_.Create_Leaf_(dafny.UnicodeSeqOfUtf8Bytes(m.Path), dafny.SetOf(aces...), dafny.EmptySet, dafny.SetOf(roles...))
		return dafny_mtree, nil
	}
	dafny_mtree_children := make([]interface{}, len(m.Children))
	for idx, c := range m.Children {
		dafny_mtree, err = am_tree_to_dafny_tree(c)
		if err != nil {
			return dafny_mtree, err
		}
		dafny_mtree_children[idx] = dafny_mtree
	}
	dafny_mtree = AMDatatypes.Companion_MetaTree_.Create_Node_(dafny.UnicodeSeqOfUtf8Bytes(m.Path), dafny.SetOf(aces...), dafny.EmptySet, dafny.SetOf(roles...), dafny.SeqFromArray(dafny_mtree_children, false))
	return dafny_mtree, nil
}

func main() {
	metaStore, err := metadata.OpenEtcdMetaStore([]string{"http://localhost:2379"})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	m, err := metaStore.GetTree(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("Got tree from etcd")

	a, err := am_tree_to_dafny_tree(m)
	if err != nil {
		panic(err)
	}
	b := MetaTreeFns.Companion_Default___.Inherit(a, dafny.SetOf(), dafny.SetOf())

	d := &MetaTreeInvariants.CompanionStruct_Default___{}
	fmt.Println(d.Descendants__inherit__roles(b))
}
