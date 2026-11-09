# Access Manager Semantics
## Purpose and Scope
### Goal of this document
Describe the semantics of the Access Manager state and operations precisely enough to support proofs, audits, and implementation checks. This document reflects the past vision of the Access Manager along with agreed upon "present" changes.

Focus on the semantics of:
* the hierarchical universe of nodes,
* the permission language that constrains changes,
* the interface and commands that read or mutate the state,
* the permissions that are required for operations
* how identity is acquired and demonstrated
### In-scope questions
* What is the abstract state of the Access Manager?
* How are effective attributes and effective permissions computed?
* When is an operation allowed?
* What does each externally visible command do to the abstract state?
## Semantic Vocabulary and Notation
### Core sorts
* Access Manager state
* Path
* Node
* Leaf
* Directory
* Dataset
* Attribute
* Operation
* ACL
* cACL
* Credential
* DelegationToken
## Abstract State Model
The Access Manager's state is composed of atomic nodes. Nodes are either represent a user, a workload, a dataset, etc. or are attributes which have a deeper meaning within the Access Manager. Nodes can have any unique name.
### Tree-structured namespace
All nodes are structured in a tree, keyed by path names. Relationships between nodes are ancestor/descendant. These relationships inform the path names. See Prefix and Ancestry for more detail. All nodes in the tree are typed as Directory or Leaf depending on if they are "containers" (capable of having children) or are terminals respectively.
### Metadata attached to nodes
Applied-role annotations may appear on any node. See Effective attributes for more detail.
cACLs may appear on any node. See Effective cACLs for more details.
Auxiliary metadata (e.g. ssh keys, dataset annotations, etc.)
### Well-formedness invariants
There are no cycles in the tree structure (i.e. it is an actual tree)
* All nodes in a path name must exist (no references to non-existent nodes)
* All paths derive from am:// at the root
* All attributes in a cACL must exist in the access manager (no references to non-existent attributes). 
## Hierarchical Semantics
### Prefix and Ancestry
* Path Representation: A path is a sequence of node names separated by /, starting with am://: path(n) = am://n₁/n₂/.../nₖ where k ≥ 0
* Prefix Relation: Path P is a prefix of path Q (written P ⊑ Q)
* P ⊑ Q ⇔ Q = P / R for some R (possibly ∅)
* Ancestor/Descendant defintion
```
ancestor(n) = { m | path(m) ⊑ path(n) ∧ m ≠ n } 
descendants(n) = { m | path(n) ⊑ path(m) ∧ m ≠ n }
```
* Parent/Child relationship
```
parent(n) = m ⇔ path(m) ⊑ path(n) ∧ | path(n) | = | path(m) | + 1 
child(n) = m ⇔ parent(m) = n
```
Note: $|path|$ denotes path cardinality/the number of components in the path (e.g., $|am://user/alice| = 3$).
Tree Invariant: $∀ n ∈ N: ancestors(n)$ forms a chain under $⊑$ (totally ordered by prefix)
### Effective Attributes
A node has directly annotated attributes attached at its path. Attributes inherited by an node n consist of all attributes made inheritable by the parent of n. The effective attributes of an node are the union of direct and inherited attributes.

Note 1: attributes can affect authorization even if the node cannot view that role.

Note 2: Effective attributes accumulate via union at the node level, but inheritance is filtered by the parent's inheritable set.

Note 3: delegated attributes are permission-based (see Sections 5.4.1 and 6.4). Delegated attributes can be unioned with the effective attributes.

An actor A can delegate attribute attr to actor R iff: attr ∈ effective(A) ∧ authorized(A, Delegate, attr) ∧ authorized(R, Redeem, attr)
The Delegate and Redeem permissions are annotations on the attribute node itself, not derived from hierarchy.

### Formal Definitions for Attributes
Let $N$ be the set of nodes in the Access Manager tree, `Attribute` the set of attributes, and `Time` the set of possible times. Let $P(X)$ denote the power set of $X$.

Each node has a locally configured set of attributes that it permits its children to inherit. We write this set as:
```
n.inheritableAttributes
```
This is local node configuration. It is not necessarily the same as the computed set $inheritable(n)$.

We define the following functions:
```
 direct : N → P(Attribute)
 inherited : N → P(Attribute)
 inheritable : N → P(Attribute)
 delegated : N x Time → P(Attribute)
 effective : N x Time → P(Attribute)
```
The parent function returns the direct parent of a non-root node:
```
parent : N \ {root} → N
```
The root is excluded from the domain because it has no parent.

```
direct(n) is the set of attributes directly annotated on node n:
direct(n) = { a ∈ P(Attribute) | a is directly annotated on n }
```
The root has no inherited attributes:
```
inherited(root) = ∅
```
For every non-root node, the inherited attributes are the attributes that the parent makes inheritable:
```
inherited(n) = inheritable(parent(n))
```
The inheritable attributes of a node are the attributes that are either directly applied to the node or inherited by it, restricted to the attributes that the node has configured for inheritance:
```
inheritable(n) = ( direct(n) ∪ inherited(n) ) ∩ n.inheritableAttributes
```
This definition allows a node to make an inherited attribute inheritable again. In particular, an inherited attribute may be included in `inheritable(n)` when the node includes it in `n.inheritableAttributes`.

Delegated attributes are defined separately as they are transient and are not inherited through the tree. At time t:
$delegated(n, t) = \lbrace a | \exists d - a delegator, d delegated a to n, t < a.expirationTime \rbrace$

The effective attributes of a node at time t are:
```
effective(n, t) = direct(n) ∪ inherited(n) ∪ delegated(n, t)
```

Thus, delegated attributes affect authorization for their recipient while valid, but they do not become inherited attributes merely because they appear in effective(n, t).

The use of $direct(n) ∪ inherited(n)$ rather than $effective(n, t)$ in the definition of $inheritable(n)$ is intentional. It prevents a delegated attribute from being propagated through the hierarchy.
## Effective cACLs
An authorization `triple (A, op, T)` for an actor $A$, operation $op$ on target $T$ has effective cACLs consisting of 
* outbound cACLs for op attached to A or non-local cACLs for op attached to any ancestor of A and resolved against the effective attributes of T
* inbound cACLs for op attached to T or non-local cACLs for op attached to any ancestor of T and resolved against the effective attributes of A
### Formal Definition
The structure of a cACL is as follows
```
(op: Operation, acls: P(ACL), isLocal: bool, isOutbound: bool)
```
An ACL $L$ is satisfied by attributes $X$ iff $L ∩ X \ne ∅$. A cACL $c$ is satisfied by attributes $X$ iff $\forall L ∈ c.acls: L ∩ X ≠ ∅$.

The function that returns all cACLs associated with node $n$ for an operation is
```
 opcacls: N × Operation → P(cACL)
```
The function that returns the set of ancestors of $n$ is
```
ancestors(n) = { parent(n) } ∪  ancestors(parent(n)), if n ≠ root
     ∅, if n = root
```
The effective cACL function can be recursively defined for any node $n \in N$: 
```
effectivecacls(n, op) = opcacls(n, op) ∪ { c | ∃ n' ∈ ancestors(n): c ∈ opcacls(n', op) ∧ ¬c.isLocal }
```
The effective inbound and outbound cACLs for $n$ can then be defined by
```
effoutbound(n, op) = { c ∈ effectivecacls(n, op) | c.isOutbound }
effinbound(n, op) = { c ∈ effectivecacls(n, op) | ¬c.isOutbound }
```

An operation is authorized if
```
authorized(A, op, T) ≡ ( ∀ c ∈ effinbound(T, op): satisfied(c, effective(A)) ) ∧ 
( ∀ c ∈ effoutbound(A, op): satisfied(c, effective(T)) )
```
## Visibility and redaction
View controls 
a) general visibility (whether nodes and metadata elements are visible)
b) whether roles are redacted when retrieved as a list of applied roles or in an ACL
c) as a qualifier on all other operations
## Permission Language Semantics
### Attributes
Attributes are nodes in the Access Manager state.

`ApplyAttribute`, `UseAttribute`, `Delegate` and `Redeem`  are the permissions specific to attributes

Attributes are used in cACLs to make decisions about whether a permission can be satisfied.

Attributes may be packaged into credentials used for external systems (e.g. DAOS)

### ACL syntax and semantics
An ACL is a set of attributes. An ACL is said to be satisfied by a set Ax of attributes if any of the attributes in the ACL are in Ax (i.e. ACL ∩ Ax ≠ ∅).
### Conjunction of ACL (cACL) syntax and semantics
A cACL is an annotation applied to an node used to determine if a permission can be obtained.

A cACL contains:
* an operation,
* a set of ACLs,
* a flag isLocal to mark it as local. If the cACL is local it will not be inherited by descendants of the node.
* a flag isOutbound which marks the cACL as outbound (if true) or inbound (if false).

A cACL is satisfied by a set of attributes when every ACL in it is satisfied. 
### Idnodes
An identity is a path in the state that represents an actor. All idnodes can be insecure or secure.  Idnodes are secure if they have or inherit a VouchFor permission, or have annotations specifying a SPIFFE registry and identity, or have annotations specifying ssh public keys. Idnodes are insecure if none of those conditions apply.

All requests to the Access Manager must include a proof of identity for the actor known as an Access Manager credential. Secure idnodes must use a secure credential. Insecure idnodes may use any kind of credential. Such a credential may simply specify the identity or it may be the result of redeeming a delegation token and thus contain additional attributes and permissions. 

The simplest Access Manager credential is known as an insecure credential and is simply the path of the identity.  A secure credential contains the identity path, an optional list of delegated attributes and an optional list of any delegated permissions. Secure credentials are signed by the Access Manager.
## Primitive authorization relation
A primitive authorization relation is a `triple (A, op, T)` consisting of actor $A$, an operation $op$ and a target $T$. $A$ is allowed to perform $op$ on $T$ if every effective inbound cACL for op at $T$ is satisfied by the attributes of $A$ and every effective outbound cACL for $A$ is satisfied by the attributes of $T$.

This means implicitly that authorization begins as "allow-all" and becomes more restrictive with additional ACL/cACLs. Put simply, if an actor and target have no cACLs associated with them, all operations will be permitted from the actor to the target.

## API authorization
Many API calls require multiple primitive authorization relations to be satisfied in order to complete. For instance, adding a cACL to a dataset requires View and Admin on the dataset and View and UseRole on every attribute in the new cACL.
## Meaning of each operation
View: controls metadata visibility and is required by all operations.
Admin: controls "annotation". Adding or removing annotations from nodes requires Admin permissions.
ApplyAttribute: to apply an attribute annotation to a principal or path, require ApplyAttribute on the attribute and `Admin` on the target path; also require visibility as needed.
UseAttribute: to use an attribute inside a cACL, require UseAttribute on each referenced attribute and Admin on the path where the cACL is attached; also require visibility as needed.
VouchFor: controls whether an identity plugin may obtain a credential on behalf of a principal.
Delegate: to temporarily apply an attribute to another node (delegate), require `Delegate` on the attribute path. Also requires visibility of the attribute, and of the target.
Redeem: to temporarily receive an attribute from another node (delegated to), require `Redeem` on the attribute path. Also requires visibility of the attribute (maybe not the principal?)
Read: controls read access to a target, also requires visibility of target.
Write: controls a) write access to a target and b) the ability to create new nodes under a Directory. Requires visibility of target.
## Meta-circularity/Self-governance
The same permission language that governs operational and governance actions such as changes to roles, cACLs, and identity-control metadata.
# Interface and Command Semantics
## API model
A request carries a credential that specifies the identity of the actor along with any additional attributes or permissions.
## "Read-only" commands
Candidate commands
* List a path (ls functionality). 

Requires: View of target's entire path. If View is not satisfied for a child of of the target, that child will not be returned.

* Inspect all metadata for a target.

Requires: View of target's entire path. 

## Mutation commands
Note: If I have permission to modify a target (permissions or attributes) in a way that affects a child via inheritance, I do not explicitly need permission for that child to make the change so long as I can satisfy the parent's requirements.

All requirements for commands are evaluated on the pre-mutation state, and can be thought of as pre-conditions (P) in a Hoare-triple (P, S, Q)
Candidate command families:
* Create node
Requires: View of target's entire path. Write permission for the target. Target must be a Directory. Must specify if new node is of type Directory or Leaf. The created node must be unique, i.e. if am://A/B/ already exists I cannot create a new node B under am://A/.
* Delete node
Requires: View of target's entire path. Write permission for the target. Must satisfy all requirements for removing annotations (including permissions and attributes) on the node. If node is an attribute, all references to the node must be removed from cACLs.
* Add annotation
Requires: View of target's entire path. Admin permission for the target.
* Remove annotation
Requires: View of target's entire path. Admin permission for the target.
* Add permission annotation
Requires: View of target's entire path. Admin permission for the target. UseAttribute permission for attribute used in new permission annotation.
* Remove permission annotation
Requires: View of target's entire path. Admin permission for the target. UseAttribute permission for every attribute used in the permission annotation.
* Add attribute annotation
Requires: View of target's entire path. Admin permission for the target. ApplyAttribute permission for attribute being annotated.
* Remove attribute annotation
Requires: View of target's entire path. Admin permission for the target. ApplyAttribute permission for annotated attribute being remove.
## Credential-issuing commands
Candidate command families:
* Obtain a credential
Requires: View of target's entire path. Principal must be able to satisfy the inbound and outbound cACLs with the target (see section 5.4).
* Identity vouching
Requires: View of vouchee's entire path. Voucher must be able to satisfy the VouchFor permission placed on the vouchee.
* Requesting a delegation token
Requires: View of delegator's entire path, and entire path of delegatable attribute. Delegator must be able to satisfy the Delegate permission placed on the attribute.
* Redeeming a delegation token
View of entire delegatable attribute path. Delegation token from valid delegator (see above). Redeemer must be able to satisfy the Redeem permission placed on the attribue.
# Security and Correctness Properties
## Basic safety properties
No principal can obtain a permission unless the effective policy allows it.

No principal can mutate metadata unless the effective policy allows it.

## Policy-mutation safety
No principal can attach an attribute or cACL that they are not authorized to use.

No principal can widen visibility or authority outside the self-governing policy rules.

## Identity safety
No plugin can obtain a credential for a principal for which it does not have VouchFor  permission

No plaintext identity can be used if a stronger identity mechanism is configured.

## Delegation safety
No principal can delegate an attribute it doesn't have permission to delegate.

No principal can redeem an attribute it doesn't have permission to receive by delegation.

When redeeming a delegation token, some, none, or all of the attributes may be delegated.

# Access-Manager Credential Factory, SPIFFE Registry Trust Store, and Link License.
For an organization to issue valid X.509 SVIDs for its users and workloads, it must operate an independent SPIFFE registry backed by a certificate authority whose root is anchored, per trust domain, in the Access Manager's trust store. Additionally, for cross organization access management, the Access Manager must (1) map the SPIFFE URIs, i.e., spiffe:// registered by each org into its am:// principal namespace, and (2) issue SPIFFE-bound Link Licenses that the target's policy enforcement point can verify. This requires certain changes in the access manager which includes: 
SPIFFE Registry Trust Store.

Access Manager Recognized Certificate for native mTLS and ES256 Signature.

Credential Factory for Link Licenses.
# SPIFFE Registry Trust Store
The Access Manager must have a trust store in which each organization root CA is anchored by the organization's trust domain.
The trust domain of user/workload SPIFFE URI, i.e., access.acme.org is used to pin the root CA in the trust store that validates the user/workload certificate chain against only the relevant root CA during mutual TLS handshake.
## Access Manager Native mTLS Support and Link License Signature 

To support mutual TLS, the Access Manager needs to present a certificate whose root CA is trusted by all entities that connects to it. This requires that root certificate of the Access Manager cert be distributed either out-of-band or via remote attestation. For remote attestation, entities prove their identities to a TAS server after which the TAS server delivers the entity's SVID, the root CA and the Access Manager root CA (so the entity can verify the Access Manager's TLS certificate).
The Access Manager must have a distinct signing key used solely for the Link License. Policy Enforcement Points (PEP) can either receive the Access Manager's public key for Link License verification via remote attestation as part of the authenticated bundle delivery or mutual TLS such that endpoints that present a valid PEP SVID receives the Access Manager key. 
## Link License Specification
The Link License is a SPIFFE-bound access token. It design is inspired by RFC 8705/RFC 9068. The Access Manager issues this license to entities requesting access to another organization's endpoint. The Link License is a proof-of-possession token. It can only be used by a peer that completes an mTLS handshake with a valid SVID whose SPIFFE ID matches the license's cnf binding, i.e., the holder of the private key for a certificate that both carries that SPIFFE URI and chains to the trust domain's trusted root. A stolen license is useless without that key, and a forged certificate is useless without a valid chain. 

Header for the Link License:
```
{ "typ": "link+jwt", "alg": "ES256", "kid": "am-key-1" }
```
Access Manager can have multiple keys or one single key for signing Link Licenses. Policy Enforcement Points (PEP) can select the corresponding public key based on the kid .
PEP can enforce use of alg == ES256 to prevent the use of alg == none .

Claims/Payload:
```
{
  "iss": "https://accessmanager.com",
  "sub": "am://user/acme/alice",
  "aud": "https://edge.yoyodyne.example.org/daos1/",
  "iat": 1783723675,
  "nbf": 1783723675,
  "exp": 1783723975,
  "jti": "0f8b1c9d2a4e6f70",
  "cnf": {
    "spiffe#S256": "7u-oPdG0rHgPvlMFP1P7mN1J48JJmWozvA76goSFMIQ"
  }
}
```
### Link License Properties
The Link License can only be provisioned or issued by the Access Manager, hence, iss = acccessmanager.com or similar URL.

The subject, i.e., `sub` of the license is the bearer whose Access Manager policy allows access to the endpoint.

The `aud` is the resource/policy enforcement point `am://` path or URL, where the Link License will be enforced based on signature, SVID, and confirmation token. URL can be used outside of the token while `am://` path is used in `aud`.

Link Licenses are short lived, and must carry `iat` for time when the license was issued, `nbf` to indicate license validity is not before (optional), and `exp` to show when license expires. License should be provisioned to last for 5 minutes.
`jti` can be used to keep track of Link Licenses used at the enforcement point, to enforce use once and prevent replay attacks.

The Link License allows for cross-organization access, and it guarantees that the specific bound holder from ACME (`sub=="am://user/acme/alice"`) can connect with an endpoint at YOYODYNE (`aud=="https://edge.yoyodyne.example.org/daos1/"`). 

The Link License will be provisioned only after the Access Manager policy checks are completed, i.e., if the policy specified in the Access Manager allows an ACME entity to connect with a YOYODYNE entity then and only then is a connection license issued by the Access Manager to the ACME entity.

A Link License must contain a confirmation token, which is the base64 encoding of the digest of the SVID SPIFFE URI for the subject the license is issued to, i.e.,  `cnf.spiffe#S256 == base64url(SHA-256(utf8(SpiffeID(leaf))))`. We use digest of the SPIFFE URI as confirmation token rather than the SVID certificate digest due to the rotational nature and short TTL of SVIDs, ensuring the Link License is valid even post SVID expiration/rotation.

## Policy Enforcement Point (PEP)
On each connection the PEP enforces the following, such that if any failure occur on any check, the connection is rejected and license is deemed invalid.
* Verifies the peer's certificate chain against the trust domain's root only (trust-domain pin); 
* Verifies the license signature via kid and rejects `alg != ES256`; 
* Checks `exp/nbf/iat`; 
* Checks `sub == ` the originating peer's own `am://` identity and `aud ==` this PEP's identity;  
* Checks cnf.spiffe#S256 against the presented SVID. 
## Attested Access Management

Figure 1 - TEE Attestation Service.jpg
Figure 2 - ID plugin and User:Workload Credentials.jpg
FIgure 3 - Connection License and PEP.jpg

