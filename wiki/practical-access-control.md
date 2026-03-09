# Mechanics of Access Control

This document is a terse and fairly technical introduction to the mechanics of
how access control works in the Data Access Manager without saying much about
why it works that way. For a friendlier introduction with more text about the
motivation and ideas behind the design, check out the
[introductory document](Home.md).

# Core ideas

The basic ideas in the Data Access Manager can be boiled down to a few basic
ideas that have to do with objects and their names, operations, attributes, and
permissions. In general, the access manager controls actions by users and
workloads on users, workloads, attributes and data. Conceptually, all such
actions can be expressed as a triple containing an agent (a user or workload
initiating the action), an action (the operation being performed) and a target (
the object or directory being acted upon). The access manager determines whether
the action is allowed by checking the permissions on the target object or
directory and comparing these permissions to the attributes of the agent. In
addition, the accesss manager also checks the permissions on the agent versus
the attributes of the target.

Decomposing the access control problem into permissions on the objects as well
as the agents allows the access manager to user inheritance to simplify the
management of permissions. Inheritance allows permissions to be set at a
directory level and then inherited by all objects or directories below that.
This inheritance allows separation of concerns between the administration of
higher level directories from the administration of lower level directories.

Inheritance is also key in expressing access control policies in ways that are
amenable to formal verification.

## Objects and Path names

The universe of the access manager consists of users, workloads, attributes (or
roles) and data. Each of these objects has a path name (actually,
a [uniform resource locator](https://en.wikipedia.org/wiki/Uniform_Resource_Identifier))
that starts with `am://`. The path name for an object consists of the scheme (
the `am://` part) followed by one of  `user`, `workload`, `role` or `data`
followed by a string composed of alternating delimiters (`/`) and path
components. A path component is a non-empty string composed of upper and lower
case characters, decimal digits or any of `-`, `.`, `_`, or `~`
(that is, matched by the regular expression `[a-zA-Z0-9\-._~]+`).

A path name can refer to a directory or to a user, workload, role or data.
Directories are used to organize and group objects and to define paths of
inheritance for common characteristics such as access controls or attributes.

The policies that define which operations the access manager allows are defined
by attaching permissions and attributes to objects or directories. Other
metadata such as public keys, URLs for data access, or external usernames can
also be attached to objects or directories.

All of these kinds of metadata are generally referred to as annotations. Each
annotation has a unique identifier consisting of the path where the annotation
is attached, the class of annotation and a 53-bit integer. There are two
reserved annotation classes ("applied-role" and "ace") but other names can be
used as well.

### Internal Metadata

Users, workloads and data contain additional metadata used by the access
manager. In the case of users and workloads, the additional metadata may contain
information that allows the access manager to verify the identity of the user or
workload invoking some operation. For workloads, this is likely a SPIFFE
identity while for users, it may be the public half of an `ssh` key-pair.
Identity plug-ins can be integrated into the system and new kinds of plugins may
require new kinds of annotations.

The internal metadata for data describes how to access the actual dataset. For
object stores such as AWS S3 or Azure Blob Storage, this typically includes a
URL for the root of the dataset and a regular expression for additional
components of the dataset. In addition, there may be additional identity
information that allows the access manager to generate short-lived access
credentials for the dataset when needed. The specific details of the internal
metadata for datasets depends on the technology used to store the dataset.

## Operations

Fundamentally, users and workloads can operate on users, workloads, roles and
data. These operations might consist of reading or writing data, or they might
be administrative actions that create or delete objects or that change roles or
permissions. Identity plugins are themselves workloads that are given the right
to vouch for the identity of users or workloads.

The allowed operations are `View`, `Admin`, `Read`, `Write`, `ApplyRole`,
`UseRole` and `VouchFor`. For the most part, the meaning of these roles are
relatively self-evident from their names but there are some nuances. Typically,
determining whether an operation is allowed involves checking multiple
permissions. For instance, to read data, you need to have `View` and `Read`
permission on the data object.

### View

The view operation determines whether an object or directory can be seen. You
have to have `View` permission on an object or directory to operate on it or
test whether it exists or to see annotations on it. Additionally, any role you
don't have `View` permission for will appear in redacted form if it is used in a
permission or applied to a user or workload. The rationale for allowing this
partial visibility is to make it easier to understand why an operation might or
might not be allowed.

### Admin

The `Admin` operation refers to any change to annotations on an object or
director. This includes permissions, roles applied to a user or workload, or any
other kinds of annotations. Note that just having the `Admin` permission isn't
enough. To make changes need to be able to `View` and `Admin`
the object and you need to have `UseRole` on all the roles in the before and
after versions of a permission. To change the roles on a user or workload, you
need `View` and `Admin` on the user or workload and need `ApplyRole` on every
role that you are adding or deleting.

### Read and Write

The `Read` and `Write` operations are pretty much what they sound like. You need
`View` and `Read` permission to read data and you need `View` and `Write`
permission to write or update data. In order to create a directory or object,
you will need `View` and `Write` permission on the parent directory.

### ApplyRole and UseRole

The use and application of roles is limited by the `ApplyRole` and `UseRole`
operations. Adding or removing a role to/from a user or workload or any
directory under the user or workload roots requires the `View` and `Admin`
permission on the object, and the `ApplyRole` for the role or roles being added
or removed.

To use create or modify a permission expression on any object or directory, you
will need to have `View` and `Admin` on the object and
`UseRole` for all the roles in either the old or new versions of the permission
expression.

### VouchFor

The `VouchFor` permission is used to control which identity plugins can be
trusted to verify the ownership of an identity for a user or workload. Multiple
plugins can have the right to vouch for a user and different organizations can
have different identity plugins based on which administrative domains they
belong to.

The way that this works is that a user will contact an identity plugin and
provide a sufficient proof of their identity. The user or workload specific
information needed to validate this proof may be stored externally or the plugin
may look for this information in an annotation on the user or workload object in
the access manager.

Once satisfied that the user or workload is authentic, the plugin will request a
credential for that user from the access manager. If the plugin (which is itself
a workload) has `View` and `VouchFor` permission for the user or workload, the
access manager will create a signed JWT and return it to the plugin.

There are two special cases that allow access to the access manager upon
initialization. The first special case is that users with no `VouchFor`
permission can simply claim their identity without the mediation of a plugin.
This insecure access mode is only used in demonstration systems.

The other special case is that if the reserved attribute
`am://role/native-login` allows `VouchFor` permission for a user or workload
and if that user or workload has an `ssh-pubkey` annotation with a suitable
public key in it, then the access manager itself will serve as an identity
plugin. This special case is normally used to provide initial or emergency
access to the access manager when external identity systems are not available
because they are down or have not yet been configured. This second special case
is also used to allow identity plugins to authenticate themselves.

## Roles

Roles in the access manager are simply symbols (or attributes) that are attached
to users or workloads and referenced in permission expressions. To be a valid
role a role must have been created ahead of time below the `am://role` root. It
is unusual to attach any annotations other than permission expressions to roles.

In spite of the fact that roles are handled as atomic entities with no internal
structure, it is common to group them in directories and sub-directories to
simplify the management of the permissions on the roles themselves by using
inheritance. Restricting permissions on directories containing roles is an
economical way to limit who can change access controls. Restricting visibility
of roles gives the effect of classified keyword access control.

In addition to the roles applied directly to users or workloads, roles can be
applied to directories containing users or workloads. The effective roles that
apply to any user or workload include both roles that are directly applied in
addition to any roles on any parent or super parent directory above that user or
workload. This inheritance is commonly used to apply a common role to all users
in an enterprise or working group.

## Permissions

The right to perform any operation is defined by logical expressions that are
attached to objects or directories. These logical expressions are known as
access control expressions (or ACE for short).

An access control expression consists of an operation (one of the 7 listed
above), a unique identifier (used to make updates to the access control
expression unambiguous), a list of access control lists (ACLs), a `local` flag
that controls whether the ACE is inherited by child objects or directories, and
a `agent` flag that controls whether the ACE is used to control operations on
the controlled object (if false) or used to control operations initiated by the
controlled object (if true).

The effective ACE for an action is the combination of all the ACEs for
operations that are implied by the operation for both the agent of the action
and the target of the action subject to the inheritance rules described below.

A user or workload is said to satisfy an ACE if the ACE is satisfied by the
roles applied directly to or inherited by that user or workload.

An ACE is said to be satisfied by a set of roles if all the ACLs in the ACE are
satisfied by that set of roles.

An ACL consists of a list of roles. An ACL is said to be satisfied by a set of
roles there is some role that is in both the ACL and the set of roles.

In general, ACEs can be applied to any object or directory but ACEs controlling
the `ApplyRole` and `UseRole`operations can only be attached to directories and
roles under `am://role`. Roles can be applied to any object or directory.

### Inheritance

When an ACE is applied to a directory and the `local` flag is false, that ACE
will apply to all direct or indirect children of that directory (either
directories or objects). This allows general, broad-reaching ACEs to be set at
directories near the top of the directory with more fine-grained controls to be
set at lower-level directories. For instance, it is common to set an ACE that
limits the visibility of a company's top-level directory for users, workloads,
roles and data to only employees of the company and apply a role at the top
level of the company's users and workloads. Users and workloads from outside the
company will be unable to see any evidence that the company even exists in the
access manager as a result and no user in the company will be able to change
this unless they have `ApplyRole` and/or `UseRole`
permission for the company-wide role and `Admin` permission for the top-level
directories. That `Admin` permission is set with a local ACE so that it is not
inherited by lower level directories.

Similarly, an ACE with the target flag set may be applied to a directory
containing a number of workloads to prevent those workloads from being able to
read or write data outside of a limited set of directories.

At the current time, only roles and permissions are inherited. Other annotations
such as public keys or physical URLs are not inherited. The rationale for the
current approach is that some kinds of annotations such as user-specific ssh
public keys are not what you would want to inherit, but there may be operational
value in allowing inheritance of other kinds of information that we don't know
about. The current design is slightly more conservative and this decision may be
reconsidered as more experience is gained in using the system.

### Direct, local, and effective ACEs

As a matter of terminology, an ACE that is applied to a directory or object as
opposed to being inherited is called a "direct" ACE. An ACE applied to a parent
or super-parent of an object or directory that does not have the `local` flag
set is called an "inherited" ACE. ACEs with the `local` flag set are referred to
as "local" ACEs and only have effect if they are direct. ACEs with the `agent`
flag set to true are referred to as "agent" ACEs while ACEs without that flag
are referred to as "target" ACEs.

Most actions on objects, even just getting access credentials for a dataset
require that permission be available for multiple operations and across multiple
levels of inheritance. For convenience in discussing the internals of how the
decision is made to grant permission for an operation, it is common to refer to
an "effective" ACE for a particular operation. The effective ACE for a
particular operation is formed by combining the ACLs for all ACEs both inherited
and direct for the required permissions for both the agent and target of the
desired action. For instance, the effective ACE for reading a dataset involves
the direct and inherited target ACEs for `View` and `Read`
permissions on the dataset as well as direct and inherited agent ACEs for
`View` and `Read` on the agent of the action. The effective ACE is constructed
from this list of ACLs.

