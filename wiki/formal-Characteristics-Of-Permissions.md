# Formal semantics for the Access Manager

## Objects
All objects in the access manager including users, workloads, roles (as attributes are called) and datasets
are leaves in a tree with four sub-trees `user`, `workload`, `role` and `data` respectively for each of these
categories of objects. Intermediate nodes in the tree are called directories. 

## Paths

Each object or directory has a path associated with it that has a textual form of a URL. 
This URL is limited to the following form:

- scheme is `am`
- the domain can be one of `workload`, `user`, `role` or `data`.
- the path consists of "/"-separated non-empty components and is defined by the following grammar
- no query or fragment are allowed
```
path ::= /* empty */ | <component>+ <tail>
component ::= re"/[a-zA-Z0-9_\-]*"
tail ::= re"(/([a-zA-Z0-9_\-]*)?)?"
```

Formally, the URL structure is discarded and paths are considered to be the a vector of strings
containing the domain and subsequent "/"-delimited strings.
For convenience the regular set of legal strings in a path is called $\mathcal T$. 

The `<` relation is defined on paths such that $a < b$ iff 
$a$ is a proper prefix of $b$. Addition is defined as concatenation. The direct parents of an object $a$ 
are $\mathtt{parents}(a) = \lbrace p \mid x \in \mathcal T \wedge p + x = a \rbrace$. The ancestors
of an object $a$ are $\mathtt{ancestors}(a) = \lbrace p \mid p < a \rbrace$. 

Similarly, the direct children of a directory $d$ are $\mathtt{children}(d) = \lbrace c+x \mid x \in \mathcal T \rbrace$ and
the descendants of a directory $d$ are $\mathtt{descendants}(d) = \lbrace c \mid d < c \rbrace$ 

## Principals, roles, data and operations

We use caligraphic font for the various universal sets of interest

* $\mathcal P$ is the set of all principals (users and workloads). $\mathcal P = \mathtt{descendants}([\mathtt{user}]) \cup \mathtt{descendants}([\mathtt{workload}])$.
* $\mathcal R$ is the set of all roles. $\mathcal R = \mathtt{descendants}([\mathtt{role}])$.
* $\mathcal D$ is the set of all data objects. $\mathcal D = \mathtt{descendants}([\mathtt{data}])$.
* $\mathcal O$ is the set of all operations. $\mathcal O = \lbrace \mathtt{Read}, \mathtt{Write}, \mathtt{Admin}, \mathtt{View}, \mathtt{ApplyRole}, \mathtt{UseRole}, \mathtt{VouchFor}\rbrace$. 
* $\mathcal A$ is the set of all access control expressions
* $\mathcal T$ is the set of valid strings in paths

## Inheritance of roles

All objects and directories can have roles applied to them. The set of roles applied to a directory or
object $a$ is referred to as $\mathtt{appliedRoles}(a)$. In addition, a directory or object $a$ inherits
a set of roles $\mathtt{inheritedRoles}(a)$ and has a (possibly empty) list of roles that should 
be removed from inheritance. These sets are combined so that the effective roles for an object $a$
are

$\mathtt{roles}([]) = []$
$\mathtt{roles}(a) = \left( \mathtt{roles}(\mathtt{parent}(a)) \cup \mathtt{appliedRoles}(a) \right) - \mathtt{disinherited}(a)$

Note that this means that roles applied to $a$ while simultaneously being disinherited from $a$ are not effective.

## Access control expressions, paths and implied operations

An access control expressions (ACE) is defined as a structure with an operation and a set of access control lists. 

An access control list (ACL) is a set of roles. The roles in an access control list are written as $\mathtt{roleset}(l) \rightarrow \mathtt{Set}[\mathcal R]$. 

A set of roles $r \subseteq \mathcal R$ satisfies an access control list $l$ if some role in $r$ appears in $l$. 

$\mathtt{satACL}(r, l) \equiv \exists \rho \in r, \rho \in l$.

An ACL with no roles cannot be satisfied, $\neg \exists s, \mathtt{satACL}([], s)$.

An access control expression $a$ is satisfied by a set of roles $r$ if and only if each access control list in $a$ is satisfied. 

$\mathtt{satACE}(r, a) \equiv \bigwedge_{l \in a} \mathtt{satACL}(r, l)$

Access control expressions with the same operation can be combined $a \wedge b = \left \lbrace a.\mathtt{op}, a.\mathtt{acls} \cup b.\mathtt{acls} \right \rbrace$. Access control expressions with no ACLs are satisfied by any set of roles, $\forall s, \mathtt{satACE}(\lbrace \_, []\rbrace, s)$.

## Critical sets
 A set of roles $r\subset \mathcal R$ is said to be critical with respect to an access control expression $a:ACE$ if any set of roles that satisfies $a$ will have $r$ as a subset, that is $\forall s: \mathtt{satACE}(s, a) \implies r \subset s$. From the definition of an access control expression, a critical set of $a_1$ is also critical to $a_1 \wedge a_2$. There are often many distinct critical sets for an access control expression.

 Critical sets are often useful in proofs about accessibility because a critical set for the access control at the top of a sub-tree is critical for all effective access control anywhere in the sub-tree due to inheritance. If you  can prove that some set of users or workloads cannot have some or all of the members of that critical set, you can prove that nobody in that set of users or workloads can satisfy the access control for anything in the sub-tree.

## Paths and permissions
Any path in the access manager can have zero or more access control expressions associated with it. Each such associated access control expression is either _direct_ or _local_ according to whether the access control expression is inherited by the children of the path. Multiple access control expression with the same operation can be associated with the same path. In addition, each association is has a direction which is _outbound_ or _inbound_. This direction  which affects how the access control expression is used to determine whether an operation is allowed.

These direct and local controls are combined to form the *effective* access control expression for an operation by using a system of inheritance and accounting for interactions between operations. The way that this is done is by combining the direct controls for all ancestors of a path (including the path itself) with any local controls for the path. This gives the accumulated controls for a path $x$, a specific operation $\Omega$ in direction $d$ as

$\mathtt{accumulated}(x, \Omega:\mathcal O, d) = \mathtt{local}(x, \Omega, d) \wedge \left(\bigwedge_{s < x} \mathtt{direct}(x, \Omega), d\right)$

The *effective* access control for an object $x$ and operation $\Omega$ further qualifies the `accumulated` access control with implied operations. For instance, the effective access control for a data object and the `Read` operation actually consists of the combination of the accumulated access control for both `Read` and `View` operations. The effective access control for adding a role to a principal (or removing it) consists of the accumulated `View` and `Admin` controls for the principal as well as the `View` and `ApplyRole` controls for the role in question. The effective access control for object $x$, operation $\Omega$ and direction $d$ is written as $\mathtt{ace}(x, \Omega, d)$

To summarize,

* $\mathtt{satACL}(r:\mathtt{Set}[\mathtt{Role}], l:\mathcal ACL) \rightarrow \mathtt{bool}$ is a function that determines whether the given roles satisfy the ACL $l$ because some role in $r$ is in $l$. 
* $\mathtt{satACE}(r::\mathtt{Set}[\mathtt{Role}], a::\mathcal A) \rightarrow \mathtt{bool}$ is a function that determines whether the given roles satisfy the ACE $a$.
* $s:\mathtt{Set}[\mathtt{Role}]$ is considered critical to $a:\mathcal A$ if $\mathtt{satACE(a, x)} \implies x \subset s$. 

* $\mathtt{direct}(x, \Omega::\mathcal O, d)$ is the set of direct access control expressions for path, operation $\Omega$ and direction $d$
* $\mathtt{local}(x, \Omega::\mathcal O, d)$ is the local access control list for an object, operation and direction
* $\mathtt{accumulated}(x, \Omega::\mathcal O, d)$ is the combination of local and inherited access controls.
* $\mathtt{ace}(x, \Omega::\mathcal O, d)$ is the combination of all accumulated controls for all operations implied by $\Omega$.

# An demonstration that a set is sub-critical

As an example of how these concepts can demonstrate that a set of principals $S$ is not critical with respect to a particular operation $\Omega$ on an object $x$, extend previous definitions to include principals:

* $\mathtt{allowed}(u::\mathtt{Principal}, \Omega::\mathtt{Operation}, x) = \mathtt{allowed}(\mathtt{roleset}(u::\mathtt{Principal}), \mathtt{ace}(\Omega::\mathtt{Operation}, x))$ this states that the roles on a user or workload $u$ are sufficient to perform operation $\Omega$ on object $x$. Note that $\mathtt{ace}(\Omega::\mathtt{Operation}, x)$ will likely include access control expressions for operations  such as `View` and may involve `Admin`, `Use` or `Apply` depending on the particular nature of $\Omega$.
* $\mathtt{roleset}(p::\mathtt{Principal}]) \rightarrow \mathtt{Set}[\mathtt{Role}]$ this is the set of roles on the principal $p$
* $\mathtt{roleset}(S::\mathtt{Set}) = \bigcup_{z \in S} \mathtt{roleset}(z)$ 

Clearly, $\left(\mathtt{roleset}(u::\mathtt{Principal}) \cap \mathtt{roleset}(\mathtt{ace}(\Omega, x)) = \varnothing\right) \Rightarrow \neg \mathtt{allowed}(u, \Omega, x)$. From this, we can range over of all members of a set of principals $u \in S$ to see $\mathtt{roleset}(S::\mathtt{Set}[\mathtt{Principal}) \cap \mathtt{roleset}(\mathtt{ace}(\Omega, x)) = \varnothing$ implies $S$ is sub-critical with the permissions on $x$ and roles of members of $S$ as they stand. 

This allows an important bound on the criticality of $S$. The roles of $S$ cannot be changed unless some member of $S$ has `Apply` permissions for some role. Similarly the permissions cannot be changed if no member has effective `Admin` permissions on any object, $x$ included. Thus, we have sufficient conditions that $S$ is sub-critical if

* $\mathtt{roleset}(\mathtt{ace}(\mathtt{\Omega}, x)) \cap \mathtt{roleset}(S) = \varnothing$
* $\forall_{z} \mathtt{roleset}(\mathtt{ace}(\mathtt{Admin}, z)) \cap \mathtt{roleset}(S) = \varnothing $
* $\forall_{r\in \mathcal R} \mathtt{roleset}(\mathtt{ace}(\mathtt{Apply}, r)) \cap \mathtt{roleset}(S) = \varnothing $

The conditions here are far more stringent than strictly necessary, but they have the virtue of being easy to verify and relatively common in practice (for instance, almost no workloads or users will have any `Apply` permissions on any role whatsoever). Effectively what they are saying is that users who have no administrative powers and who cannot currently satisfy an access control for an operation cannot perform that operation without outside help.

We can make the opposite kind of statement as well. Any set $S$ that includes a principal that has `Admin` rights on every object and `Apply` rights on every role is critical. More restricted statements can be made which ultimately demonstrate that the delegation model is sound.

# Outline of Required Proofs

## Preliminaries
1. Define ACE semantics and path structure with simplified ops
2. Show that ACEs can be collected at terminal of path
3. Show that an object protected by an ACE for an operation cannot have that operation performed by a user without sufficient roles
4. Define role use and reference semantics
5. Define admin process
6. Show no sequence of admin steps can lead to successful operation unless admin has use or refer permissions on minimal set of roles

## Every set of users with no administrators has static capabilities

## Empty roleset intersection implies no access

## Disjoint delegation is possible and secure

## Indexed form of metadata is equivalent to theoretical form

## Audit algorithm that finds users with access to specific data is correct

## Audit algorithm that finds data access for specific users is correct



# Other work

See [this paper in SecDev](https://people.scs.carleton.ca/~paulv/papers/secdev2018.pdf) for a discussion of what formal methods mean in this context (BP stands for "Best Practice", by the way). See the [SeL4](https://sel4.systems/About/seL4-whitepaper.pdf) project for an example of formal methods done well. We won't come near that standard, but it provides a worth aspiration.