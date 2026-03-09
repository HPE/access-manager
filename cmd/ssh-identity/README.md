# SSH identity

The idea of the ssh identity is that a user can get a signed JWT from the ssh
identity service if they have a private key whose corresponding public key is
listed in the metadata for the identity in question (assuming that ssh identity
is allowed for that user).

Internally, this works by establishing an ssh connection to the identity service
which knows all of the authorized keys and their associated identity. The login
name in this request is the URI for the requested identity without the `am:`
prefix. If the session is established correctly, the identity server returns a
signed JWT for that identity that can be used to authenticate requests to the
access manager. Typically, this process is wrapped up in a client program that
handles putting the JWT in a file with correct permissions.

One special use of this JWT is so that a FUSE filesystem service can read the
JWT to get a list of delegated attributes for the user which can be treated as
the groups for the user. That allows the use of a shared file system without
having to have synchronized usernames or identities. As long as you have the
private key, you can assume any identity associated with the corresponding
public key.

