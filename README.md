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
Make sure you have the pre-requisites installed as listed below.

```sh
sudo apt install -y protobuf-compiler
go get -tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go get -tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
go get -tool google.golang.org/protobuf/cmd/protoc-gen-go
go get -tool google.golang.org/grpc/cmd/protoc-gen-go-grpc
go install tool
```

Then generate a key for the server and build all of the key executables

```sh
ssh-keygen -t rsa -f host.keygen
for x in cmd/*/*.go
do
  go build $x
  echo $x
done
```

This will build the access manager, the metadata server, the credential manager
and the command line interface and put them into the `cmd` directory.

If you modify any service definitions in the various `*.proto` files, you will
need to regenerate the associated go code using this command:

```shell
sh ./scripts/proto-gen.sh
```

## Trying out your system

To try things out, you need to run the access manager and at least one metadata
server instance. It helps to run each of these in a separate window so that you
can separate the log outputs for each one. Each command requires some
configuration which is simplest to put into environment variables.

## Metadata server

The Access Manager uses `etcd` to store metadata. On MacOS, you can install etcd
using

```shell
brew install etcd
```

See [the etcd page on installation](https://etcd.io/docs/v3.5/install/) for
other platforms such as Linux.

For testing and playing around, you can use a single, unsecured instance of
`etcd`:

```sh
etcd
```

To remove all metadata, use this command:

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
separate window may be preferable.

## Access manager

Build the access manager and associated utilities as described above. To start
the access manager using the default address for client access
(`http://localhost:8080`) and default address for etcd
(`http://localhost:2114`) use this

```sh
./access-manager &
```

This program will produce a fair bit of log output so running in a separate
window may be useful.

The following environment variables can be used to configure the access manager.
The defaults should be sufficient for testing and demos.

| Environment | Description |
Defaults | |---------------------+----------------------------------------+------------------------------------------| |
LOG_LEVEL | Logging detail level | "info", allowed "debug", "error"         | |
AM_Port | Access manager port for GRPC access | 4000 | | AM_MetricServerPort |
Access manager port for metrics | 2114 | | AM_GatewayPort | Access manager port
for REST over HTTP | 8080 | | AM_MetadataUrls | Hostnames or URLs for
`etcd`           | localhost | | AM_ServerKey | File containing ssh host key |
host.key | | - | Port for ssh credential access | 2222 |

## Try it out

You can send some sample commands to the server using the `am` command-line
utility which was built using the commands above. You will need some environment
variables to define where the access manager is and what your user path is

```sh
export AM_USER=am://user/hpe/bu1/alice
export ACCESS_MANAGER_URL=http://localhost:8080
```

Add some dummy data on metadata server, first erase the content on the metadata
server and load a bootstrap image

```sh
etcdctl  del --from-key "/" 
./am boot new_sample
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
some way to authenticate users in a more secure way.

There are three classes of user authentication in the access manager:

a) use a identity plugin. An identity plugin is just a workload that is allowed
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

At this point, we can get a credential for the id-plugin and write it to a file
using ssh. At the same time, we can set the `AM_USER` environment variable to
point to this file.

```console
$ ssh -l am://workload/yoyodyne/id-plugin localhost -p 2222  | tee id
eyJhbGciOiJFUzI ... crypto stuff deleted ... 1cAeUFSIVT0myen_m7iYMSL7DiTSkuiBR5SYA
Connection to localhost closed.
$ export AM_USER=@id
```

Using this identity for the `id-plugin`, we can get a credential for `bob`

```console
$ ./am vouch am://user/yoyodyne/bob | tee bob_cred  
eyJhbGciOiJFUzI ... crypto stuff deleted ... fcPvNldb-likO7I1SRiVB3MQg```
```

