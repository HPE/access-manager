# How Things Go Wrong

In [this article](https://www.wiz.io/blog/38-terabytes-of-private-data-accidentally-exposed-by-microsoft-ai-researchers), a recent data leak from a Microsoft research group was described. 

The touble all started when SAS tokens (Shared Access Signature) were shared for training data. These tokens were not, however, either limited in scope (they gave access to the entire storage account), nor limited in time (they didn't expire). Both scope limitation and short expiration time are options with SAS tokens, but neither was used due to user error. The impact was that private chat sessions, full system backups, proprietary codebases and lots of other things were exposed. Not only that, nearly 40 TB of data was exfiltrated by ... somebody.

The core problem is that SAS tokens are a fine way to implement low-level security constraints but expecting users to configure them correctly is foolish. The core issue is that users cannot comply with multiple kinds of constraints while they are engaged in their ordinary tasks. The tasks of compliance is even harder since these users probably don't even *know* about all of the applicable limitations.

# Root Causes

The core problem here is that humans are fallible and being thus fallible should not be expected to perform perfectly in a system that defaults to dangerous settings.

Unfortunately, the system described in this article defaults to dangerous in many ways. There are, however, a number of mechanisms that would have allowed these users to share the data appropriately without screwing up.

These include:

1. **Dataset as primitive** They should have thought of their training data files as a single dataset and should have been allowed to express their needs in terms of that dataset. The structure of SAS tokens (and AWS' equivalent) defaults to tokens that give access to the entire storage account (or bucket in AWS S3). Users often don't even realize that there is an option to limit the scope of tokens. Giving users a way to talk about granting rights to a dataset helps change from failure by default into correct by default.

2. **Workloads, not users** They should have been granting access to the data for workloads, not users. These workloads should be running in a secure environment, not in a web browser on a non-secure machine. In general, this distinction between workloads and users is important to make.

3. **Short expiration** Tokens or credentials should be issued at the point and moment of use, not far in advance. What this means is that the users should be specifying rules and conditions of granting these credentials, not granting the credentials themselves. If this is done, then the credentials can have a very short life (typically 30 minutes or less). If the credentials have to be granted long before the data is read, most users simply remove the limit entirely. Short token lifespan also solves the problem of revocation.

     Clearly, however, credentials cannot have too short a lifespan. This particularly applies to delegation credentials given for an administrative purpose (where you may be waiting on a human) or for an inherently slow process such as AI agents diagnosing the misbehavior of a large system or running a large job. In many cases, this can be solved by credential refreshes, but that doesn't much work for delegated credentials.

4. **Logging of access** Issuance of tokens should be logged and attributed back to a cryptographically verified program identity which should itself have its launch logged with attribution back to a valid source. With short-lived credentials issued to hardware-locked workloads, logging issuance is essentially the same as logging access.

5. **Comprehensive audit** It should be possible to audit data access by asking who (or which workloads, really) can access particular data. In fact, this should be a standard part of any granting UI and should be done by default for the entire system whenever permissions are changed. If my system backup suddenly becomes world readable, I should be notified and it should be clear who made the change. If I make some training data available to the world, I should be notified if 10,000 other datasets are affected. Audit should be an everyday task, not an extraordinary event.

6. **Secure multi-level delegation** It should be possible to delegate control over particular data within global
constraints. Thus, the owner of the Azure storage account in question should have been able to define that no data
was world readable and then delegate control of the backups sub-class of data to the admin team. Separately,
they should have been able to delegate control over training data to the machine learning team. This delegation
should, however, not have allowed either of these teams to violate the prime directive of no outside sharing.

# How Access Manager Solves this Problem

The situation described in this article isn't particularly novel. This sort of leak has happened many times and
in many places. Each of the root causes listed above is an example of a class of problems that have collectively 
motivated specific design features in the Access Manager.

Here are the root causes again, but this time with the specific access manager features that help avoid this
kind of incident:

1. **Dataset as primitive** In the access manager, every dataset is given a unique URL so that access to it
can be controlled and granted as a unit. Since the system itself generates access tokens, these tokens can
be scoped to precisely the desired level automatically. This means that access token generation is done
correctly by default.

2. **Workloads, not users** The access manager manages workloads and users as first class entities that each
have roles and thus access permissions but which are kept entirely distinct. This means that you can apply
constraints to users but not to the parallel workloads. That allows, for instance, normal users to be granted
rights to data products, but not to raw data. It allows administrators to have the right to administer workloads,
but not the right to see their outputs.

3. **Short expiration** Since access tokens are generated by the access manager at the moment that a workload runs, the default lifetime of 30 minutes is typically sufficient for most workloads. Further, there is no provision to generate a permanent token so all access must be accompanied by a request.

4. **Logging of access** All requests for access are logged. More than this, all changes to any permissions
or role assignments are also logged so that root cause analysis can be done to determine when metadata
is changed, who made the change, and, most importantly, whether any data access was actually granted.

5. **Comprehensive audit** Audit capabilities are built into the Access Manager as a core function.
When access controls are set on a dataset, this can show everybody who has access (perhaps with a warning). Conversely, the scope of access for any user, workload or group of users or workloads can be computed with the same API. This can also be done for any historical time range. Further, the entire permission model is subject to formal mathematical proof techniques so that the overall configuration of the access manager can be verified to comply with general security policies whenever the configuration is changed.

6. **Secure multi-level delegation** The Access Manager allows constraints to be set as an
enterprise level but also allow full control within the limits imposed by those constraints to be delegated
so units and their administrative teams. Further, the units can impose further constraints and delegate
control to groups and so on. At each level, all of the constraints imposed at higher levels still apply
without any explicit action by administrators or users. This idea of separation of concerns between global
and local interests is crucial to building an environment that helps users succeed by default.


