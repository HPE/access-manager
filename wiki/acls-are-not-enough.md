# Access Control Lists aren't Enough (except when they are)
The traditional Unix permission scheme assigned rights to the owner, a group and all other users on a system has long been recognized to be inadequate for use in large-scale systems. Access control lists (ACLs) are a more general construct that allow considerably more flexibility in access control. ACLs were [used in VMS](https://www.mrynet.com/FTP/os/VMS/docs/ssb71/6346/6346p005.htm), [inherited from VMS by Windows/NT and successors](https://dl.acm.org/doi/10.1145/373256.373271)[@10.1145/373256.373271] and incorporated into Active Directory. In the late 1990s, there was an effort to standardize ACLs for Unix as part of the withdrawn POSIX.1e standard, but those capabilities have largely been [incorporated into Linux](https://www.usenix.org/conference/2003-usenix-annual-technical-conference/posix-access-control-lists-linux) [@270211] and the NFSv4 standard [@RFC3530] ([RFC 3530](https://datatracker.ietf.org/doc/html/rfc3530))is widely considered to be the successor to the POSIX effort. ACLs are also [used in DAOs](https://docs.daos.io/v2.6/overview/security/)[@DAOsV26Security] and the implementation is similar, though less complex than  POSIX ACLs. In all of these implementations, files (occasionally ACLs) have an owner and only the owner or the system administrator can change the ACL.

So how can it be asserted that ACLs are insufficient as a global mechanism for expressing security constraints if they are used so widely?

The key is that ACLs are sufficient at (relatively) small organizational scale. Within a single working group, the concept of central administration makes sense and an assumption of aligned goals makes sense. At a larger scale where you might have independent enterprises sharing a platform, these assumptions break down dramatically. The breakdown has multiple forms which are the subject of the following sections.

# Separation of Concerns
In a large organization, there are large-scale security concerns that are relatively simple (all confidential data will be accessible only by authorized employees and contractors, for instance). These policies are not subject to change by anybody but a few executives and the technical implementation of such policies should only be subject to change by a few administrators. On the other hand, on a smaller scale, there will be project-specific security constraints that are not universal across the enterprise (production data for product X must be produced only by production applications working on production data, for instance). In order to simplify lines of communication and coordination with the organization, these policies should be administered at the level where they have an impact.

This idea that there are separate kinds of concerns that should be administered by different people makes ACLs a serious problem. The simplest problem implied by a security architecture limited to ACLs is the problem of who should administer an ACL. It shouldn't be the project team (because they might disable enterprise level security by accident) and it shouldn't be the enterprise-level security team (because they will get flooded with change requests local changes that they cannot understand and thus will be susceptible to social engineering). In short, the problem is that there are separate concerns (local and global) that need to be kept separate, but which must both have technical means for enforcement.

A common way to attempt to work around for this need for separation of concerns is to implement perimeter controls at the network level and then hope that things will work out.  Such a solution has two major defects. The first is that global control via the perimeter and local control via ACLs ignores the existence of mid-scale concerns like horizontal transfer or the fact that multiple small scale groups may have divergent goals. The second big problem is that network controls are too ham-handed to selectively control access to data. This might be called the "VPN strategy" of security.

Another common work-around is to define and publish human readable policies defining security constraints that must be satisfied by all projects. This fails because comprehending and complying with these standards quickly becomes more than a full-time job and is subject to differing interpretations. That leads to the focus problem described in the next section, but it also means that even the owners of these policies will eventually wind up inserting logical self-contradictions into these policies. 

What is needed is
* the ability to have independently administered, but logically integrated technical controls at multiple (more than two) levels
* these controls need to be verified for consistency and correctness both by human audit and by automated formal verification
* high-level technical controls must be irrevocably inherited by local groups 

Since ACLs inherently function at a single level, they cannot possibly be the complete solution to this problem.

# Focus versus Perspicacity
Another problem that is related to the separation of concerns is that of focus. To do their job, individual contributors need to focus on the details of their work. In focussing, however, they will inevitably be unable to remember all the external compliance constraints they must satisfy. If these individual contributors, on the other hand, spend enough time to understand and track all applicable compliance constraints, they will be unable to do their job because they will be overwhelmed by the complexity of the compliance environment.

The only real solution here is that it local work must satisfy global constraints by default. 
The problem here is that is  in systems that don't have a strong mechanism for inheriting global constraints on data 
access, it is far too easy to accidentally defeat global controls.

# Multi-tenancy
In cloud-like environments where data must be shared symmetrically between different tenants with controls on further sharing, control via an ACL is problematic because if both parties can modify the ACL, the sharing guarantees cannot be enforced while if only one party can modify the ACL, that party can modify the sharing at will. What is needed is either some form of interlocking control that either allows only mutually agreed changes to the ACL or multiple ACLs. Either way, conventional ACLs are insufficient.

# Factoring out Common Concerns
Another problem with ACLs is that most implementations do not allow common components of ACLs to be factored out and effectively reused. Google's Zanzibar is an exception to this rule, but it has substantially more complex definition than is normally understood by the term "ACL". Without the ability to factor out common concerns, ACLs must contain high levels of repetitive content. Inevitably, not all copies of this common content will be kept in sync as requirements leading inevitably to an extremely complex pastiche of non-compliant ACLs.

# Solving the Problems
The Access Manager has five primary mechanisms for solving the problems outlined above.

* Permissions are inherited from virtual parent directories so that global constraints are automatically inherited across an organization without requiring local administrators to understand or even be aware of the details of these constraints. This also avoids large amounts of repetitive content in ACLs and allows separation of concerns between global and local administrators.
* Attributes are also inherited from virtual parent directories so that global attributes can be inherited. This enhances separation of concerns, but it also allows different attributes to be controlled by different administrators.
* For any action consisting of an Actor, Action and Object, permissions can be applied to the Object that must be satisfied by the attributes of the Actor but permissions can also be applied to the Actor that must be satisfied by the attributes of the Object. This duality allows access controls to be factored into separate concerns that exhibit different patterns of inheritance throught the Actor or through the Object hierarchies.
* Permissions in the Access Manager are composed of conjunctions of ACLs. Each ACL can be administered independently so that multiple independent parties can share control over access to data. This allows multi-tenancy, shared administration and separation of concerns.

