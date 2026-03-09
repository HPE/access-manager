# Access Manager

This code implements the Access Manager which allows express governance
constraints to be layered over a variety of storage substrates such as S3.

The general way that this works is that virtual path names are constructed that
have permissions set on them in the form of access control expressions that
allow or deny various operations on objects referenced via these path names.
These access control expressions are evaluated in terms of the roles that have
been assigned to the user or workload attempting to perform some operation.
Similar access control expressions limit who can manage the roles assigned to
users or workloads, or to change access expressions or who can even see various
entities.

You can find more detailed information in the wiki directory starting
at [this page](./wiki/Home.md)

# System Architecture

In general, the Access Manager is implemented in three major parts. The first is
the actual access manager that handles requests from users and workloads to
manage permissions or get access to data. The second part, known as the metadata
server, is simply an instance of `etcd`. The third major part of the access
manager consists of auxiliary services used by the access manager for generating
credentials, and vouching for user identities.

Different implementations of the access manager can co-exist cleanly since all
changes to metadata go through the metadata service and from there to all access
manager instances. This interoperability is useful when version upgrades are
necessary.

# Compile and Run

To compile and run the Access Manager, you will need at least version 1.25 of go
and a copy of `etcd`. To rebuild the protobuf definitions, you will need podman.

Given these, you should be able to build the Access Manager using make:

```shell
make build
```

The Access Manager uses `etcd` to store metadata. On MacOS, you can install etcd
using

```shell
brew install etcd
```

See [the etcd page on installation](https://etcd.io/docs/v3.5/install/) for
other platforms such as Linux. Do not user Linux package managers since they
typically have very old versions.

For testing and playing around, you can use a single, unsecured instance of
`etcd`:

```sh
etcd
```

If you want to remove all metadata in order to start with an empty system again,
use this command:

```sh
etcdctl  del --from-key "/" 
```

You can reload a test or bootstrap universe using the access manager command
line utility as described below. No special synchronization with the access
manager is required.

In a production setting, you would need multiple instances of `etcd` and would
need to secure connections to these instances using client and server
certificates as described in the `etcd` documentation.

Etcd produces a fair bit of logging output by default so running it in a
separate window may be preferable. That's true for the Access Manager as well.

## Access Manager signing key

At this point, you can create a key pair for the Access Manager to sign
credentials with. Make sure you create this credential with no passphrase.

```shell
ssh-keygen -f host.key
```

## Trying out your system

You now have all you need to run the Access Manager

```shell
./bin/access-manager
```

This program will produce a fair bit of log output so running in a separate
window may be useful.

The following environment variables can be used to configure the access manager.
The defaults should be sufficient for testing and demos.

| Environment | Description | Defaults | 
|------------ | ---------------------------------- | --------------| 
| LOG_LEVEL | Logging detail level | "info", "debug", "error" |
| AM_Port | Access manager port for GRPC access | 4000 |
| AM_MetricServerPort | Access manager port for metrics | 2114 |
| AM_GatewayPort | Access manager port for REST over HTTP | 8080 |
| AM_MetadataUrls | Hostnames or URLs for `etcd` | localhost |
| AM_ServerKey | File containing ssh host key | host.key |
| - | Port for ssh credential access | 2222 |

By default, the Access Manager looks for its private key under the name
`./host.key` and connects to a single instance of `etcd` via
`localhost`. In a more realistic scenario, you would likely have 3 or 5
instances of `etcd` and many instances of the Access Manager and they would use
cryptographic certificates to secure their communication with each other.

At this point, there is no metadata in the system so the Access Manager is
pretty useless. To fix that, you can insert the bootstrap metadata

```shell
export AM_USER=am://user/the-operator
export ACCESS_MANAGER_URL=http://localhost:8080
./bin/am boot bootstrap
```

This should print "Loaded bootstrap" on completion.

You can verify that there is now metadata in the system by running

```shell
./bin/am ls -r am://
```

This will produce a recursive enumeration of the entire universe as the Access
Manager knows about it.

```json
{
  "path": "am://",
  "children": [
    {
      "path": "am://data"
    },
    {
      "path": "am://role",
      "children": [
        {
          "path": "am://role/operator-admin"
        }
      ]
    },
    {
      "path": "am://user",
      "children": [
        {
          "path": "am://user/the-operator"
        }
      ]
    },
    {
      "path": "am://workload"
    }
  ]
}
```

This universe has a single user `am://user/the-operator` who has a single
attribute `am://role/operator-admin`.

Note that we ran the ls command claiming to be `the-operator` without any
evidence that we had the right to do so. That worked because `the-operator` has
not been configured to use any of the available verification methods. Such an
identity is known as a demonstration identity and, as the name implies, is only
useful for demos.

Such a lax setup is handy for demos, but it isn't very interesting. To make
things one notch more realistic, we can use ssh keys to authenticate
`the-operator`. In practice, ssh keys are a good backup for critical identities
in case you have a problem with more elaborate identity systems.

First, install your public key as an annotation on `the-operator`:

```shell
./bin/am annotate am://user/the-operator ssh-public-key="ssh-rsa ... your key here ..."
```

After this, you will find that the `ls` command we used before will now fail
because we can no longer successfully claim to be the `the-operator` without any
kind of proof.

To get that proof, we can `ssh` to the Access Manager. If everything checks out,
the Access Manager will return a string containing an Access Manager credential
which we will use instead of the user URL. We can set this up in a single step:

```shell
export AM_USER=$(ssh -l am://user/the-operator -p 2222 localhost)
```

This will give us the necessary credential to act as `the-operator` for a
limited period of time (configurable, but typically 15 minutes).

## A more elaborate universe

Let's start over with a fancier universe. First erase the content on the
metadata server and load a sample metadata universe that has a number of users
divided between different companies.

```sh
etcdctl  del --from-key "/" 
./am boot new_sample
```

Set up your identity as Alice who works for HPE:

```sh
export AM_USER=am://user/hpe/bu1/alice
export ACCESS_MANAGER_URL=http://localhost:8080
```

For demos, the `new_sample` universe is good since it contains a variety of
users in different top-level domains. For real use, you probably would rather
start with the `boostrap` universe which contains a minimum configuration. You
can repeat this sequence at any time to reload the universe. Accidentally using
the `boot` command twice will result in an error, but the metadata that is in
the metadata server will not be damaged.

Once you have a universe loaded, you can explore:

```console
$ ./am ls am://user
{
   "path": "am://user",
   "children": [
      {
         "path": "am://user/hpe"
      },
      {
         "path": "am://user/small"
      }
   ]
}
```

Note that different users see different content. Alice has the attribute
`am://role/hpe/hpe-user` which matches up the `View` permission on all of the
`am://*/hpe` top-level directories.

```console
$ ./am ls -l am://user/hpe/bu1/alice
{
   "Details": {
      "path": "am://user/hpe/bu1/alice",
      "roles": [
         {
            "role": "am://role/hpe/bu1/bu1-admin",
            "tag": "applied-role",
            "unique": 7461446305135842,
            "version": 1
         }
      ],
      "inheritedRoles": [
         {
            "role": "am://role/hpe/hpe-user",
            "tag": "applied-role",
            "unique": 1425641780505653,
            "version": 1
         },
         ... stuff omitted for clarity ...
      ],
      "inheritedAces": [
         {
            "op": "VIEW",
            "acls": [
               {
                  "roles": [
                     "am://role/hpe/hpe-user",
                     "## Redacted role ##"
                  ]
               }
            ],
            "tag": "ace",
            "unique": 79234067,
            "version": 1
         },
         ... stuff omitted for clarity ...
   },
}
```

Other users may see different things. For instance, in the `new_sample` demo
universe, we have a user `am://user/demo-god` who has the
`am://role/panopticon` role and that allows them to see everything (that role is
not visible to alice except in redacted form). Here is what `demo-god` sees
under `am://user`:

```sh
./am ls am://user  
{
   "path": "am://user",
   "children": [
      {
         "path": "am://user/demo-god"
      },
      {
         "path": "am://user/hpe"
      },
      {
         "path": "am://user/small"
      },
      {
         "path": "am://user/the-operator"
      },
      {
         "path": "am://user/yoyodyne"
      }
   ]
}
```

The `am` command has extensive built in help. Try `am help` or `am ls help`
for examples. One particularly useful option is the `-v` option which shows you
exactly which URLs the command is using to get the results it is showing you.

# User Authentication

So far, we have played with identities that don't need to be authenticated
(alice and demo-god). That isn't realistic, of course, and in practice we need
some way to authenticate users in a more secure way. You saw a hint of that
earlier when setting up `the-operator` with ssh keys.

There are three classes of user authentication in the access manager:

a) use an identity plugin. An identity plugin is just a workload that is allowed
to vouch for a particular user. As a result, the plugin can request a signed
credential for that user. The plugin itself must be authenticated as well which
is commonly done using the second method.

b) use ssh credentials assigned to an identity. If you annotate a user identity
with an ssh public key that corresponds to an ssh private key accessible to a
user id on Linux-like system, then processes running under that user id will be
able to ssh to the access manager to get a credential. That credential can then
be used instead of a username.

c) use the URL of the identity directly as a credential. This is what we did in
the examples above and is only allowed if neither (a) nor (b) apply. That
happens when there is no `VouchFor` permission on or inherited by the identity
and there is no `ssh-key` annotation on that identity.

## Demo of user authentication

In the `new_sample` demo universe, there is a company Yoyodyne which has set up
an identity plugin `am://workload/yoyodyne/id-plugin` with `VouchFor`
permission on all users under `am://user/yoyodyne`. We can add an ssh public key
to the identity plugin:

```shell
./am  annotate  am://workload/yoyodyne/id-plugin "ssh-pubkey=$(cat ~/.ssh/id_rsa.pub)"
```

At this point, any program that can authenticate itself with the corresponding
private key will be allowed to get an Access Manager credential for anybody at
Yoyodyne. To make it easier to see how this works, the `am` command line utility
has a `vouch` command to allow it to act like an identity plugin. That means
that if we set the `am` command up with an Access Manager credential for the
`id-plugin` identity, the `am` can retrieve a credential for any Yoyodyne user.

To do this, we use `ssh` to get the credential for the plugin and pass that 
to `am` via the `AM_USER` environment variable.
```console
ssh -l am://workload/yoyodyne/id-plugin localhost -p 2222  | tee id
eyJhbGciOiJFUzI ... crypto stuff deleted ... 1cAeUFSIVT0myen_m7iYMSL7DiTSkuiBR5SYA
Connection to localhost closed.
export AM_USER=@id
```

Using this identity for the `id-plugin`, we can get a credential for `bob`

```console
$ ./am vouch am://user/yoyodyne/bob | tee bob_cred  
eyJhbGciOiJFUzI ... crypto stuff deleted ... fcPvNldb-likO7I1SRiVB3MQg```
```

And at this point, we can see the world through Bob's eyes
```shell
export AM_USER=@bob_cred
./am ls am://user  
```
