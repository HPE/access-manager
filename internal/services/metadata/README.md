# Metadata persistence

Access manager metadata is expressed as a tree with annotations on the nodes of
the tree. Each node has a URL with the scheme `am://`. The annotations on the
nodes can be structural to indicate leaf nodes of the tree. The annotations can
also be permission expressions or applied roles that are used by the access
manager itself for making decisions about access. It is also possible to have ad
hoc annotations that store things such as public keys that are used by identity
plugins, URLs for objects in S3 or addresses of data repositories such as
databases or file systems. All annotations are versioned to allow safe update
semantics.

The nodes of the metadata tree together with all annotations are stored in a
table with key and data formats designed to facilitate some operations (such as
scanning for the children of a node) without forcing the storage layer to
understand anything about the annotations themselves.

# Key Structure

This tree of metadata is persisted in etcd in a namespace with prefix `/meta`.
Annotations are stored by a key formed using URL of the node with the `am://`
scheme removed and a fragment identifier added to distinguish different
annotations. This path is prefixed by a fixed length integer indicating the
number of components in the path so that range scans on keys traverse the
metadata tree in breadth-first order. For all entries representing annotations
on nodes, a fragment identifier is appended to the path to indicate the kind of
annotation.

The fragment identifier consists of `#` followed by a tag and an integer with
less than 63-bits expressed in hexadecimal form. This convention allows range
scans to be used to get all of the annotations on a node or all of the direct
children of a node. The unique integer is required on all annotations other than
`leaf` annotations.

We guarantee that all nodes also appear in etcd without any fragment. This means
that all of the nodes in the tree can be enumerated with a range scan even if
there are no annotations on a node or some child.

Here is an example of the keys used to store a metadata tree that has one user
and one role.

```
00000 /
00001 /data
00001 /data#ace-37af
00001 /role
00001 /role#ace-e392
00001 /user
00001 /user#ace-48a8
00001 /workload
00001 /workload#ace-5982
00002 /role/operator-admin
00002 /role/operator-admin#leaf-48a3
00002 /user/the-operator
00002 /user/the-operator#leaf-518c
00002 /user/the-operator#role-4857
```

Note how we can find all of the top-level directories and their annotations
using a prefix scan on `00001 /`. Similarly, we can find all children of the
`am://user` directory by scanning with a prefix of `00002 /user/`

# Kinds of Annotations

Each node can have multiple annotations with the same tag. Each annotation has a
different key distinguished by the fragment of the URL.

The four kinds of annotation include structural markers, applied roles,
permissions expressions, and internal data such as cryptographic public keys,
external URLs for credential factories and external identities such as Spiffe
IDs. There are conventions about where different kinds of metadata can appear,
but the storage layer doesn't care about or enforce these conventions.

The values of all metadata annotations are stored as protobuf encoded values
wrapped in a header structure. This header contains the start and end times for
the annotation. The actual content of the annotation is stored as a raw bytes
that are not interpreted by the storage layer.

A table with commonly used annotation tags is shown below

| Type      | Description                                                   |
|-----------|---------------------------------------------------------------|
| `ace`     | A single access control expression                            |
| `role`    | A role applied to a user or workload.                         |
| `leaf`    | Indicates that this node is a leaf-node in the metadata tree  |                               
| `svid`    | A SPIFFE id for a workload                                    |                               
| `ssh-key` | An ssh key to be used to authenticate a user, workload or key |                               
| `s3-info` | Information required to access and S3 table such as URL       |

The `leaf` type is used to signal that no children should be created below a
node. This is used to indicate any object in the tree such as a dataset, user,
role or workload. Other kinds of annotations can appear anywhere in the tree,
but, by convention, `role` is only allowed under the `am://user`,
`am://workload` and `am://key` top-level directories. No structural or
semantic conventions are enforced at the storage layer.

For all annotations, the annotation content is stored in a wrapper type
`metadata.Annotation` that has a header containing meta-metadata that
layers above the storage can use to determine how to interpret the annotation
data itself. 

For all annotations, the `unique` value is used to allow multiple metadata
values with the same type. This isn't useful for `leaf` annotations, but most
other kinds of annotation can have multiple values on the same node.

Setting the version to -1 allows a record to be created or overwritten without
regard to the version of any existing data. Setting version to 0 indicates that
there should not be a pre-existing value with the same key. All other values of
version require that there be a pre-existing record with exactly that version.