# Access Manager versus Mitre's Top 25

Mitre has published an analysis of reported vulnerabilities and classified them into the [top 25 most dangerous software weaknesses](https://cwe.mitre.org/top25/archive/2024/2024_cwe_top25.html#top25list). The access manager combined with the patterns of usage that it encourages addresses more than half of these weaknesses.

Here is Mitre's list of top weaknesses with links back to their descriptions and to access manager features that address each weakness.

| Rank | ID | Name | Impact |
|--|--|--|-:|
| 1 | [CWE-79](https://cwe.mitre.org/data/definitions/79.html) | Improper Neutralization of Input During Web Page Generation ('Cross-site Scripting') |  [access control expressions](#access-control-expressions), [delegation](#delegation) |
| 2 | [CWE-787](https://cwe.mitre.org/data/definitions/787.html) | Out-of-bounds Write | |
| 3 | [CWE-89](https://cwe.mitre.org/data/definitions/89.html) | Improper Neutralization of Special Elements used in an SQL Command ('SQL Injection') |  [access control expressions](#access-control-expressions), [delegation](#delegation) |
| 4 | [CWE-352](https://cwe.mitre.org/data/definitions/352.html) | Cross-Site Request Forgery (CSRF) | [access control expressions](#access-control-expressions), [delegation](#delegation) |
| 5 | [CWE-22](https://cwe.mitre.org/data/definitions/22.html) | Improper Limitation of a Pathname to a Restricted Directory ('Path Traversal') | [delegation](#delegation) |
| 6 | [CWE-125](https://cwe.mitre.org/data/definitions/125.html) | Out-of-bounds Read | |
| 7 | [CWE-78](https://cwe.mitre.org/data/definitions/78.html) | Improper Neutralization of Special Elements used in an OS Command ('OS Command Injection') | [delegation](#delegation) |
| 8 | [CWE-416](https://cwe.mitre.org/data/definitions/416.html) | Use After Free | |
| 9 | [CWE-862](https://cwe.mitre.org/data/definitions/862.html) | Missing Authorization | [identity plug-ins](#identity-plug-ins), [delegation](#delegation) |
| 10 | [CWE-434](https://cwe.mitre.org/data/definitions/434.html) | Unrestricted Upload of File with Dangerous Type | |
| 11 | [CWE-94](https://cwe.mitre.org/data/definitions/94.html) | Improper Control of Generation of Code ('Code Injection') | [delegation](#delegation) |
| 12 | [CWE-20](https://cwe.mitre.org/data/definitions/20.html) | Improper Input Validation |  |
| 13 | [CWE-77](https://cwe.mitre.org/data/definitions/77.html) | Improper Neutralization of Special Elements used in a Command ('Command Injection') | [delegation](#delegation) |
| 14 | [CWE-287](https://cwe.mitre.org/data/definitions/287.html) | Improper Authentication | [delegation](#delegation), [identity plug-ins](#identity-plug-ins) |
| 15 | [CWE-269](https://cwe.mitre.org/data/definitions/269.html) | Improper Privilege Management | [delegation](#delegation), [identity plug-ins](#identity-plug-ins) |
| 16 | [CWE-502](https://cwe.mitre.org/data/definitions/502.html) | Deserialization of Untrusted Data | |
| 17 | [CWE-200](https://cwe.mitre.org/data/definitions/200.html) | Exposure of Sensitive Information to an Unauthorized Actor | [delegation](#delegation) |
| 18 | [CWE-863](https://cwe.mitre.org/data/definitions/863.html) | Incorrect Authorization | [delegation](#delegation), [identity plug-ins](#identity-plug-ins) |
| 19 | [CWE-918](https://cwe.mitre.org/data/definitions/918.html) | Server-Side Request Forgery (SSRF) | |
| 20 | [CWE-119](https://cwe.mitre.org/data/definitions/119.html) | Improper Restriction of Operations within the Bounds of a Memory Buffer | |
| 21 | [CWE-476](https://cwe.mitre.org/data/definitions/476.html) | NULL Pointer Dereference | |
| 22 | [CWE-798](https://cwe.mitre.org/data/definitions/798.html) | Use of Hard-coded Credentials | [short-lived credentials](#short-lived-credentials), [identity plug-ins](#identity-plug-ins) |
| 23 | [CWE-190](https://cwe.mitre.org/data/definitions/190.html) | Integer Overflow or Wraparound | |
| 24 | [CWE-400](https://cwe.mitre.org/data/definitions/400.html) | Uncontrolled Resource Consumption | |
| 25 | [CWE-306](https://cwe.mitre.org/data/definitions/306.html) | Missing Authentication for Critical Function | [access control expressions](#access-control-expressions), [delegation](#delegation) |

# Access control expressions
Access control expressions (ACEs) are a core component of the access manager. The express the logical combinations of access attributes a user or workload must have in order to access data. The logical form of an ACE as a conjunction of access control lists combined with inheritance, make ACEs a key enabler for the separation of concerns and the limitation of the ability to modify access controls.
# Identity plug-ins
The access manager can be connected to a variety of identity proof schemes such as OAUTH or SPIFFE. Typically, there will be several supported plugins and a single identity in the access manager may have multiple ways forms of proving supported at the same time. This is particularly true during transitions between identity provider or if a workload can be executed on different platforms that support different identity frameworks.

This open-ended support of multiple identity frameworks makes it easier to adopt the overall access manager framework. Once an application uses the access manager to access data, the privileges for the application itself can be dropped or even nearly eliminated. This follows the principle of least privilege, but, more importantly, it makes it impossible for the application to function unless proper authentication of incoming user requests is done.

Adopting the access manager as the sole arbiter of data access allows all or most privilege management code to be removed from the application as well.
# Delegation
The attribute delegation feature of hte access manager handles a range of issues related to the confused deputy problem as well as command or query injection.

A wide swathe of critical security problems occur when a software such as a web server is given high privileges so that it can satisfy user requests. That can work as long as the web server always verifies the user identity correctly and makes correct decisions about which operations it should perform. The phrase "as long as" is doing too much work here, however, since we shouldn't be encoding access control in an application and we can't expect developers of such applications to always get things like user authentication just right.

It is much better if the application in question has a minimum level of access which is insufficient to cause serious problems. All access to sensitive data should require a combination of the attributes of the application and the attributes of a user making a request ([see delegation credentials](./delegation-credentials.md)). These attributes can only be combined by getting a delegation credential for the user and combining it with the application identity. If authentication is not done, the application will fail safe by being unable to access sensitive data. Further, the application will not be responsible from implementing access controls to ensuring policy compliance. 

Using the access manager properly does not prevent all impacts of this kind of issue, however, since invalid operations may be within the proper scope of a users' access. In particular, a user is allowed to write to particular data elements, query injection could allow invalid values to be written.

CWE-79, CWE-89, CWE-22, CWE-78, CWE-862, CWE-94, CWE-77, CWE-287, CWE-269, CWE-200, CWE-863, CWE-306

# Short-lived credentials
Embedding credentials in source code or in source code repositories is mitigated by the universal use of short-lived credentials in the access manager. Storing credentials in source code is still a mistake, but the consequences are much less severe if the credentials have expired by the time that they are committed. Also, if short-lived credentials are used everywhere, developers will have much less incentive to embed them since they will quickly have no value.

CWE-798

# Excessive user access
# Memory management
The access manager really can't help with basic issues like buffer overruns, null pointer dereferencing and such. 