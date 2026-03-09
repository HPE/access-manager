# Abstract

The Access Manager mediates access requests to data
resources. Applications or users request access to data resources
(which can be S3 objects, DAOS data stores, SQL servers or many other things)
and the access manager evaluates the request using the identity of
requestor against the access constraints of the requestor and the data. If the
constraints allow access, then an access pointer (typically a URL) and
associated credentials are returned. The credentials are short-lived
so that revocation does not need to be dealt with.

The Access Manager also manages all of the metadata that
encodes the access constraints. This metadata includes entries for all
of the user or workload identities known to the system, the roles that
are assigned to these users or workloads and the access control
expressions that define which roles are required to perform operations
on data. This same access control structure is used to define who (or
what) can create users, workloads, roles or data references, apply
roles, or modify access control expressions.

The Access Manager does not actually store any data. It only
records references to the actual data along with metadata that defines
access constraints. Typically, the access manager does not store
credentials either but creates them on demand using standard
mechanisms such as AWS Security Token Service or the equivalent. The workloads
that generate credentials are known as "credential factories".

The Access Manager also does not normally verify the identity of the requestor. 
It relies on external identity providers known as identity plugins for this. 
When an identity plugin verifies an identity for a process or user, it's verification
is accepted as valid by the Access Manager if the identity plugin satisfies the
`VouchFor` permission on the identity being verified. 

# Background

The primary goal of the Access Manager is to allow data to be
shared between applications and users
easily with very low barriers to usage while maintaining high levels
of security. One particular aspect of the Access Manager design is
that it is designed to make it easy to adopt incrementally so
that data that is already managed by other systems will be easy 
to integrate with the Access Manager for the control of incremental
kinds of access.

It is common in large organizations for it to bedifficult to share data even with a
relatively homogeneous data environment because each data micro-estate
is managed and controlled data as an isolated island. API
mediated access data could be supported, but that requires a bespoke
secure network connection to be established between any two entities
that needed to share data. Data could be copied from one system to
another but that requires that a
file export process be built and that data ingest processes to be
established on the receiving side. When the micro-estates use different
underlying techology, the problem can be much worse because it can be
difficult to reconcile differences in data models and access methods.
Even worse, this difficulty in sharing leads organizations to design
their overall data architecture with minimal sharing or even complete 
isolation between different data micro-estates.

Having polyglot data micro-estates leads to secondary impacts as well. 
Having multiple forms of access control mechanisms leads to high
operational complexity which leads, in turn, to errors in configuration.
A recent study by XM Cyber[@XMCyberDataSecurity] found that 50% of data
breaches were caused by misconfigured data access controls and it is
easy to draw a line from high complexity to misconfiguration.

The Access Manager is intended to solve for these complexity
and cost inhibitions on data sharing and to allow such data to stored
and accessed without having to establish data processing clusters that
are used solely to copy and store data. Further, the Access Manager is
intended to support multi-tenancy and recursive operator-tenant models.

# Business Requirements

We have observed within multiple large organizations that data
sharing is extremely difficult to achieve and that this difficulty
results in high costs, slow time-to-market and lost business.

The core business requirement for the Access Manager is to avoid
these problems by definition.

# Technical Requirements

The Access Manager provides a secure but very lightweight way to
share data between different projects, business units, or organizations.

This system also provides a simple and understandable way to specify
governance constraints on this sharing that has strong safe-by-default
behavior in the event of administrative inaction. These governance
constraints can be expressed in such a way that allows separation of concerns
 between different domains of concern. Thus, a DevOps team can
express limitations on how data can be shared from production instances
to staging and development instances, but not the reverse, compliance
teams can lock down PII data; project managers can limit how different
data pipelines can access (or not) data within their project and IT teams
can prevent external access to internal data. Each of
these different concerns can be handled by the people most expert in the
corresponding requirements without compromising other
concerns.

The Access Manager also governs access to its own metadata using the same
expressions as it uses to govern access to data. This has two main
effects. First, the system is understandable to users because they only
have to learn one simple mechanism of governance. Second, the system is
susceptible to formal proof techniques so that strong security assertions
can be formally proved all the way down to the implementation of the
internal permission manager.

Finally, the historical state of all metadata can be queried to not only
tell who can access or modify what, but also who did or could have
accessed or modified what during any period of time.

# Terminology

Please explain acronyms or identifiers that are used in the doc.

`attribute` - An attribute is an opaque token that is used to connect users or
workloads to access control expressions. Attributes are applied to users or
workloads and used in the access control lists that make up access
control expressions. Conversely, attributes can also be applied to data
and used in access control expressions on users or workloads.

`principal` - A principal or, more specifically, a security principal is a
user or workload.

`user` - A user corresponds to a human who needs to access data managed
by the Access Manager. Users are verified by using an identity plugin that
is configured by their organization and register with the Access Manager.

`workload` - A workload represents the identity of a process that can be
verified, typically using SPIFFE and the mutual TLS (mTLS) framework of
GLCP.

`access request` - An access request is the abstract representation of 
an action controlled by the Access Manager. Every request to the Access
Manager has one or more implied access requests. An access request
consists of three parts: the identity of the requestor (a user or workload),
the operation being requested (such as Read, Write, or View) and the object
being accessed (commonly data, but could be attributes, directories, users
or workloads. An access request is satisfied if the requestor has sufficient
attributes to satisfy the access control expression associated with the
operation on the object being accessed and the object has sufficient 
attributes to satisfy the access control expression associated with the
operation on the requestor.

`Access Manager` - The Access Manager is a policy enforcement service.
It stores metadata about
users, workloads, attributes and data that express who can do what to which
data or metadata. Users and workloads can request access to data from the access
manager. If access granted, the access manager returns a reference
(typically a URL) and short-lived credentials back to the requestor

`data` - Data in the Access Manager is given an abstract name in a 
directory-like hierarchy. Each data item has associated metadata that
associates it with some real data store (such S3, DAOS, or a SQL database)
and access control expressions that define who can do what to the data.
To access the data, a user or workload requests access from the Access
Manager which evaluates the request and, if access is granted, requests
the correct credential factory service to generate short-lived
credentials to access the data. The Access Manager then returns the data
reference and credentials to the requestor who accesses the data directly.

`SPIFFE` - The SPIFFE framework is a standard that defines how
workload idenities can be verified.

`mTLS` - Mutual TLS is a protocol for verifying the identity of
workloads using X.509 certificates issued by trusted certificate
authorities. 

`access control expression` - an access control expression contains an
operation such as Read, Write, or View together with a set of access
control lists. An access control expression is satisfied by a set
of attributes if every access control list in the expression is satisfied by
the attributes. As an example, for a workload to read an S3 object, the object
must be visible (controlled by the View operation) and readable
(controlled by the Read operation). This means that the attributes on the workload
must satisfy the access control expressions associated with `Read` and `View` on 
the object must be satisfied by attributes on the workload and the access control
expressions associated with `Read` and `View` on the workload must be satisfied
by attributes on the object. An access control expression
with no access control lists is considered to be trivially satisfied.

`access control list` - an access control list is a set of attributes and is
said to be satisfied by a set of attributes if the set has any of the attributes in the
access control list. Access control lists are only used within the Access Manager
as part of access control expressions.

# Additional References

# Scope

The Access Manager shall

* Allow management and storage of metadata for users, workloads, roles
and data via CLI, programmatic and web interface

* Store a history of all previous metadata states

* Support requests for access to data by users or workloads and 
return a data reference with credentials if access is granted or an
error if not. Requests from Python, Go and Spark will be supported

* Allow direct inspection of metadata associated with users, workloads,
roles and data. Also allow forensic examination of metadata to determine
which users and workloads can perform which operations on which data both
now or at any time in the past. These examinations will be allowed from
CLI, programmatic and web interfaces

* Use standard mechanisms for determination and verification of user
and workload identity and allow configuration of multiple identity
providers

* Use standard mechanisms for communication between internal
services within the Access Manager

* Formally define the permission model to allow formal verification of 
the implementation against the model and allow formal verification of
the access control configurations against organizational policies

# Non-Goals
The Access Manager shall not

* Store data

* Define or implement any authentication framework

* Define or implement any authorization framework

* Require data to traverse the Access Manager

* Be dependent on any particular cloud vendor except insofar as required to manage access to data stored by that cloud vendor

# Design Approach
System Context Diagram

The Access Manager accepts requests from applications that
specify the identity of the requestor and the request for access to data
or for metadata maintenance. It also interacts with platform level
services that actually provide the necessary short-lived credentials to
return to the applications.

All requests for data by the applications will then bypass the access
manager and go straight to the source of the data.

# Dependencies on other services

To the extent possible, the Access Manager is being implemented
in a dependency free form insofar as the software itself is concerned.
There are, however, several systemic dependencies that cannot be avoided.
These include:

* Any configured authentication frameworks for users or workloads.

* Any configured credential factories for generating short-lived
credentials to access data.

* An `etcd` metadata backend for storing metadata.

## Resiliency

The Access Manager will support resiliency on several levels.
Internally, the permission evaluator is a stateless service that
responds to requests. Requests for data access do not result in metadata changes, but
some administrative requests will result in metadata changes. All
such metadata changes are immediately persisted to a quorum-replicated
metadata store. Since no in-memory cache is retained by the permission
evaluator, multiple instances of the permission evaluator can be run
simultaneously to provide both load balancing and failure tolerance for
all requests including administrative requests.

The source of truth for the permission evaluator is the
metadata backend which maintains quorum-replicated persistent copies of
the metadata. 

Regarding external dependencies, outage of the AWS services for creating
credentials will affect the creation of new credentials and thus will
cause all applications to lose access to data over a period of about 30
minutes. Outages for those credential creation services are exceedingly
rare. 

# Technology stack

The Access Manager is written in Go and has good unit test coverage.

Incoming and internal requests are all processed using gRPC with
reflection enabled and with message validation and integrated breaking
change detector. All requests have RESTful HTTP/JSON endpoints as well.

# Technical Architecture

# Data Storage(s)

All changes to metadata are stored in an `etcd` cluster which uses
the Raft consensus protocol to maintain a strongly consistent,
replicated store of all metadata. The `etcd` cluster is normally run
as a three-node cluster to provide high availability and fault tolerance.
All metadata changes are made via transactions that are applied to the
`etcd` cluster which guarantees that all changes are atomic, isolated,
durable and strictly serialized.

The etcd transaction number is used to guarantee strict serialization of
all updates to metadata. Each update is applied within a transaction that
will only execute if the current version of the object being updated matches
the version specified in the update request. If simultaneous updates to the 
same object are attempted, one of the updates will arrive first and will
cause the object version to be updated. That causes the second update to
fail. When the permission manager detect such a failure, it will
retry the update by re-reading the current version of the object
and re-applying the update. 

# Data Schemas

Internally, the Access Manager maintains a dictionary of metadata objects 
indexed by path names. Objects can have multiple kinds of annotations added 
to them as access controls, attributes, or other more specialized information 
such as identity information for plugins. These objects and the annotations 
are stored in etcd using protobuf serialization with a key formed from the path
and additional information to allow fast breadth-first scanning of a directory
tree. For annotations, the key is formed by appending the annotation type
to the path name.

# Authentication
All API end points other than authentication endpoints require a `callerID`
parameter that identifies the caller. The callerID can be a user or
workload identity if the identity is set up not to require authentication, but
more commonly, the callerID will be a JWT issued by the Access Manager itself.
This JWT is obtained by first authenticating to a configured identity provider
which will provide a token that can be exchanged for a JWT from the Access Manager.

The Access Manager has a built-in fallback identity provider based on SSH 
credentials. This fallback is typically used only for initial bootstrap of
the system, as an emergency backdoor for corporate administrators, or as a
way for identity plugins to verify their own identity.

Data objects can be configured to allow delegation of a limited set of attributes.
This is normally used to allow a data repository to limit access to data based
on attributes that are delegated to the repository by the Access Manager. This 
delegation is done by requesting a delegation token from the Access Manager for
use with a particular data object. The delegation token is a JWT that encodes the
delegated attributes and is signed by the Access Manager so that the data repository
can verify the authenticity of the delegated attributes.

As an example, with DAOS, a high-performance data repository, it is typical that a
subset of a user's or workload's attributes are delegated to DAOS to be used as
group IDs for access control within DAOS. This delegation allows DAOS to enforce
fine-grained access control without having to query the Access Manager for each
access request.

This same sort of delegation can be used with other data repositories such as
SQL databases. Delegation can also be used to allow more complex policies to be
implemented based on attributes from the Access Manager. An example of this more
complex policy would be to use Cedar or Rego policies within a data repository
that refer to attributes from the Access Manager. The Access Manager would control
whether the data repository could be accessed at all and then would delegate a set
of attributes for final policy evaluation within the data repository.

# On-Premise differences

The Access Manager is designed to be deployable both on-premises and in
the cloud with minimal differences. For local debugging and development,
the Access Manager can be run as a single binary attached to a local
`etcd` instance that runs without any data redundancy. The Access Manager
has provisions for a one-time bootstrap of the system using a local image
of the initial metadata state. This bootstrap operation can also inject
SSH keys to secure the initial operator account.

# SLO & SLI

The Access Manager is designed so that it can be updated,
downgraded or patched incrementally using a rolling update with no user
visible disturbance, but it is expected that, in order to simplify
operational procedures, upgrades will be accomplished in short
maintenance windows that involve very short planned outages (typically
only a few seconds per quarter). Most clients will not observe these
outages in any case since the access manager is not in the normal data
path. There are no current provisions to make significant changes in the
persisted metadata schema without a maintenance window, however.

It is expected that most upgrades of the access manager will preserve
compatibility with earlier versions of the associated client software.
This will mean that applications will not be impacted by upgrades and it
will be possible to upgrade client versions lazily. Similarly, later
versions of the client software should have limited compatibility with
earlier versions of the access manager with some limitations on supported
functions. Automated testing for breaking changes to APIs will eventually be
incorporated in the continuous testing systems for the access manager but 
this has not yet been done.

The access manager inherently supports operation across availability
zones so no backup procedures are strictly necessary. Etcd is designed
to support wide-area replication and data backups. 

If the metadata scale grows very large, it may be desirable to migrate
the metadata store to a different backend that supports larger scale
databases. This is unlikely to be necessary in the near term. Such a
might be possible without an outage using a dual-write strategy.

The natural SLI of the system is 100% uptime and 100% accessibility
subject to network availability, but it is unrealistic to expect more
than 99.95% uptime (~1 hour per quarter of outage) until we get
experience running the system. After this experience is gained, 99.99%
uptime SLO should be feasible.

The definition of outage will remain to be defined, but a preliminary
definition would be that an outage is any 5 minute period during which a
connection to the Access Manager cannot be made, a request from a 
properly configured client takes more than 10 seconds or a request 
returns incorrect results to a well-formed request.

# HA/DR

The Access Manager together with the etcd metadata storeinherently 
provide both high availability and
disaster recovery with no operational actions required. The metadata
store maintains replicated copies of all metadata and updates these
copies using a quorum algorithm. Under normal operational scenarios,
elements of this quorum are run in physically separated locations which
makes the system inherently robust with respect to most kinds of server
and network faults. Based on this design, the system should exhibit zero
RPO and RTO behavior. 

The permission manager component is stateless and can be
replicated as needed to provide sufficient capacity and availability.
Moreover, the permission manager can handle all requests that do not 
require metadata mutation even if the metadata store cannot process 
updates so even if a quorum of metadata store instances is not 
available, it is unlikely that any applications will observe an outage.

The system will depend on the uptime of identity plugins and credential 
factories but the failure of one customer's identity plugin or credential
factories will not affect customers with their own plugins and factories. 
The implementation of highly available identity plugins and credential factories
is outside the scope of the Access Manager.

If a conventional cold backup is desired, the change-data-capture
capability of the system can be used to create an mirror of the metadata
state which can be used for small but non-zero RTO and very near zero
RPO.

With respect to logic errors or attacks where metadata is purposely
destroyed, the metadata store retains all previous states of the metadata
so the state of the system can be rolled by wholesale or selectively.
Most logic errors will only corrupt limited portions of the metadata in
the system so selective restoration will recover those situations. For
intentional data spoliation, the damage may be more widespread
which may make a more global restoration desirable. In any case, the fact
that the metadata is well governed should mean that the blast radius of 
any data corruption is likely to be much smaller than would otherwise be true.

# Failure modes

Identify possible failure situations that are not explicitly covered by
your design.

Credential generation

Bypass permission manager

Data compromise

Tight integration (mTLS) between permission manager and credential manager

Metadata store

Accidental or intentional metadata corruption

Data compromise or denial of service or loss of metadata

Snapshots, data validation, backup log, mTLS between permission manager and metadata store

New code bugs

Denial of service or data access

High unit and integration test coverage. Wide scale design and code review. Pen testing.

# Scale

There are three scale parameters of interest in the Access Manager:

* maximum practical size of metadata database including all history

* number of metadata-mutating requests per unit of time

* number of non-mutating requests

The first parameter is the integral of the second and is limited by the
database size for both the live metadata and the historical set of all
metadata states in the past. In the current implementation, a table
of the latest object versions are kept in the metadata store. The
number of metadata structures scales roughly as  θ(n + log_b n)
where _n_ is the number of structures and _b_ is at least 10 and may
be as large as 100. If there are 10,000 organizations using the 
Access Manager and each organization has 10,000 datasets (remember, 
a dataset may involve thousands of files), then the total number of 
datasets will be about a 100 million. If each organization has 10,000 
users and workloads, then the total number of users and workloads will 
also be about a 100 million. This leads to a total of about 200 million 
metadata structures. The size of each metadata structure in the database 
is less than a kilobyte so the total size of the metadata will be less 
than 200 GB. This is about 20x the recommended maximum size for a single 
etcd cluster but is not outside the range of a sharded implementation 
and there are many reliable key-value stores that can handle this scale.

In any case, the current implementation should provide sufficient runway
to redesign the system to keep metadata in a more advanced key-value store.

Regarding metadata mutation throughput, it is expected that the number of
metadata mutations per unit of time will be very low under normal
circumstances, even when loading large numbers of datasets. For instance,
even if 10,000 datasets are added to the system in a single batch, each
with half a dozen metadata updates (comparable to the entire steady state
size of the system) at the same time as 100,000 users are on-boarded,
also with several associated updates, it should be possible to handle
this load in a few tens of seconds at most. In practice, it is likely
that this volume of metadata updates would normally be spread out over
several years so the performance of the metadata store should be more
than adequate. Detailed performance benchmarks are important, but are
likely to be unimportant in terms of scaling the system.

Almost all of the requests to the system will be due to requests from
applications to read or write data. Evaluating these requests requires
reading the metadata for the data being accessed and the metadata for
the requestor and evaluating the permission expressions. The current
implementation is easily capable of handling thousands of such requests
per second on a single instance of the permission manager, but this read
load is passed to the metadata backend where read-only caches may be
required to handle very high request rates. Because of the very simple
update model for the metadata, it is also possible to include caching
within the permission manager itself to reduce load on the metadata
backend during flash requests from large-scale parallel applications.

A more binding limit may occur in the credential factories. Well-designed
applications will request credentials once and then distribute these 
credentials to all parallel threads within the application that need
access to the data, but poorly designed applications may request credentials
for each thread separately and cause request flash mobs. Caching credentials
within the credential factories can mitigate this problem.

# Security

TODO Violates guidelines by implementing an authorization scheme, but the need 
is critical. Alternatives are insufficient and the current system is very small.

Penetration or bypassing the system results in large-scale data exposure. 
Substantial topic of interest is split permissions somehow to limit blast radius.

System design matches current standards for authentication. Principle of least 
privilege is used ubiquitously (permission manager cannot persist metadata directly, 
credential factory is isolated and only does one thing, metadata store is small and 
based on standard software)

Critical questions about issues that have not been examined. What are the blind spots?

# References

* [RAFT](https://raft.github.io/)
* [ETCD-RAFT](https://github.com/etcd-io/raft)
* [Raft with gRPC](https://github.com/paralin/raft-grpc)
