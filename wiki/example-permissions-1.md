# Worked Example - Permissions
This example illustrates how roles and permissions are inherited by users, workloads and data.

## Roles
For the sample data that is used in the unit tests, we have the following roles:
```
am://role/
       #perm[Admin-local:{{am://role/operator-admin }} ]
           hpe/
            #perm[Admin-local:{{am://role/hpe/hpe-admin }} ]
               bu1/
                #perm[View:{{am://role/hpe/hpe-user }} Admin:{{am://role/hpe/bu1/bu1-admin }} ApplyRole:{{am://role/hpe/bu1/bu1-admin }} UseRole:{{am://role/hpe/bu1/bu1-admin }} ]
                   bu1-admin
                    #role[am://role/hpe/bu1/bu1-admin]
                    #perm[View:{{am://role/hpe/bu1/bu1-admin }} ]
                   bu1-aggregator
                    #role[am://role/hpe/bu1/bu1-aggregator]
                   bu1-ingest
                    #role[am://role/hpe/bu1/bu1-ingest]
                   bu1-query
                    #role[am://role/hpe/bu1/bu1-query]
                   bu1-user
                    #role[am://role/hpe/bu1/bu1-user]
                   bu1-workload
                    #role[am://role/hpe/bu1/bu1-workload]
               bu2/
                #perm[View:{{am://role/hpe/hpe-user }} Admin:{{am://role/hpe/bu2/bu2-admin }} ApplyRole:{{am://role/hpe/bu2/bu2-admin }} UseRole:{{am://role/hpe/bu2/bu2-admin }} ]
                   bu2-admin
                    #role[am://role/hpe/bu2/bu2-admin]
                    #perm[View:{{am://role/hpe/bu2/bu2-admin }} ]
                   bu2-user
                    #role[am://role/hpe/bu2/bu2-user]
               hpe-admin
                #role[am://role/hpe/hpe-admin]
               hpe-user
                #role[am://role/hpe/hpe-user]
               hpe-workload
                #role[am://role/hpe/hpe-workload]
           operator-admin
            #role[am://role/operator-admin]
```
In this dump of the meta-data, we see the role definitions themselves (marked by `#role`) 
as well as the permissions that control who can use or change the roles.

The `Admin` permission at the top level is particularly interesting. 
It allows users or workloads with the `operator-admin` role to make changes at 
the `am://user` level. Note, however, this permission is a local one and is not
inherited into the `am://user/hpe` directory. At that level there is another 
local permission that restricts `Admin` permissions to users with the `hpe-admin` 
role (i.e. the HPE corporate admins). Again, that lets those administratively 
privileged users manipulate things at the level of `am://user/hpe`, but because 
the privilege is local and thus not inherited, neither the `operator-admin` 
nor the `hpe-admin` can administer anything in the `am://user/hpe/bu1` 
or `am://user/hpe/bu2` directories.

On the other hand, the `View` permission at this level is inherited so that no
matter what else is configured for `bu1` or `bu2`, only users or workloads with
the `hpe-user` role can see the `am://role/hpe` directory or any contents below
it. As we see in the next section, that role is applied at the `am://user/hpe` as 
well so that any user defined here will be allowed to see the whole hierarchy 
unless there are additional restrictions
in directories below `am://role/hpe`. In this example, there are no such 
local restrictions on `View` although there are restrictions on `Admin` permissions.

There are additional `Admin` permissions defined at the `hpe/bu1` and `hpe/bu2` 
levels. This illustrates an important way that systems can be set up to delegate
administrative responsibilities within an organization while maintaining certain
boundary conditions. This example also shows how the normal way that permissions
get more and more restrictive as to you descend into directories can be slightly
adjusted to that the top few levels of directories can have tighter controls than
lower level ones.
## Users and Workloads
These roles are applied to various users and workloads like this:
``` 
am://user/
       #perm[Admin-local:{{am://role/operator-admin }} ]
           hpe/
            #applied-roles[am://role/hpe/hpe-user ]
            #perm[Admin-local:{{am://role/hpe/hpe-admin }} ]
            #perm[View:{{am://role/hpe/hpe-user }} ]
               bu1/
                #applied-roles[am://role/hpe/bu1/bu1-user ]
                #perm[Admin:{{am://role/hpe/bu1/bu1-admin }} ]
                   alice
                    #applied-roles[am://role/hpe/bu1/bu1-admin ]
                    #principal[spiffe://hpe.com/ajones]
                   bob
                    #principal[spiffe://hpe.com/bsmith]
                   invisible-man
                    #applied-roles[am://role/hpe/bu1/bu1-admin ]
                    #perm[View:{{am://role/hpe/bu1/bu1-admin }} ]
                    #principal[spiffe://hpe.com/ajones]
                   x/y/buried
                    #principal[spiffe://hpe.com/ajones]
               bu2/
                #applied-roles[am://role/hpe/bu2/bu2-user ]
                   alice
                    #applied-roles[am://role/bu2/hpe/bu2-admin ]
                    #principal[spiffe://hpe.com/akronecker]
       #perm[Admin-local:{{am://role/operator-admin }} ]
           hpe/
            #applied-roles[am://role/hpe/hpe-workload ]
            #perm[Admin-local:{{am://role/hpe/hpe-admin }} ]
            #perm[View:{{am://role/hpe/hpe-user }} ]
                bu2/
                   #applied-roles[am://role/hpe/bu1/bu1-workload ]
                       dashboard
                        #applied-roles[am://role/hpe/bu1/bu1-query ]
                        #principal[spiffe://hpe.com/wz1]
                       ingester
                        #applied-roles[am://role/hpe/bu1/bu1-ingest ]
                        #principal[spiffe://hpe.com/wz3]
                       monthly-summary
                        #applied-roles[am://role/hpe/bu1/bu1-aggregator am://role/hpe/bu1/bu1-query ]
                        #principal[spiffe://hpe.com/wz4]
```
From this we see that `am://user/hpe/bu1/alice` has the following roles:
```
am://role/hpe/hpe-user
am://role/hpe/bu1/bu1-user
am://role/hpe/bu1/bu1-admin 
```
while `am://user/hpe/bu2/alice` (a different user!) has the following roles:
```
am://role/hpe/hpe-user 
am://role/hpe/bu2/bu2-user 
am://role/bu2/hpe/bu2-admin 
```

and workload `am://workload/hpe/bu1/ingester` has these roles:
```
am://role/hpe-workload
am://role/hpe/bu1/bu1-query
am://role/hpe/bu1/bu1-ingest
am://role/hpe/bu1/bu1-aggregator 
am://role/hpe/bu1/bu1-query 
```

while workload `am://workload/hpe/bu1/dashboard` has these roles:
```
am://role/hpe/bu1/bu1-workload 
am://role/hpe/bu1/bu1-query
```
Referring back to what we saw in the `am://role` hierarchy, we can see that
`am://user/hpe/bu1/alice` can see all of the roles in the `hpe` directory and
can administer all of the roles in `hpe/bu1` but has no administrative permission
in `hpe` or `hpe/bu2`.
## Data Permissions
Looking at the data, we see this for data permissions:
```
am://data/
 #perm[Admin-local:{{am://role/operator-admin }} ]
           dir/bar
            #data[s3://bucket6/1]
           hpe/
            #perm[Admin-local:{{am://role/hpe/hpe-admin }} View:{{am://role/hpe/hpe-user am://role/hpe/hpe-workload }} ]
               bu1/
                #perm[Admin:{{am://role/hpe/bu1/bu1-admin }} View:{{am://role/hpe/bu1/bu1-user am://role/hpe/bu1/bu1-workload }} ]
                   hot-data
                    #perm[Write:{{am://role/hpe/bu1/bu1-aggregator }} Read:{{am://role/hpe/bu1/bu1-query am://role/hpe/bu1/bu1-user }} ]
                    #data[s3://bucket7/hot]
                   raw-data
                    #perm[Write:{{am://role/hpe/bu1/bu1-ingest }} Read:{{am://role/hpe/bu1/bu1-aggregator }} ]
                    #data[s3://bucket7/raw]
                   special
                    #data[s3://bucket1/3]
               foo
                #data[s3://bucket1/3]
               pig
                #data[s3://bucket5/13]
```
After accounting for inheritance, this boils down to these permissions on `bu1/hot-data` and `bu1/raw-data`
```
am://data/
           dir/bar
           hpe/
               bu1/
                   hot-data
                    Write:{{am://role/hpe/bu1/bu1-aggregator }} 
                    Read:{{am://role/hpe/bu1/bu1-query am://role/hpe/bu1/bu1-user }} 
                    Admin:{{ am://role/hpe/bu1/bu1-admin }} 
                    View:{{ am://role/hpe/hpe-user am://role/hpe/hpe-workload} {am://role/hpe/bu1/bu1-user am://role/hpe/bu1/bu1-workload }}
                   raw-data
                    Write:{{ am://role/hpe/bu1/bu1-ingest }} 
                    Read:{{ am://role/hpe/bu1/bu1-aggregator }} ]
                    Admin:{{ am://role/hpe/bu1/bu1-admin }} 
                    View:{{ am://role/hpe/hpe-user am://role/hpe/hpe-workload} {am://role/hpe/bu1/bu1-user am://role/hpe/bu1/bu1-workload }}
```
# What It All Means
So now we can analyze this to see who can do what to which data.

Starting with user `bu1/alice`, she can see that `bu1/hot-data` and `bu1/raw-data` 
both exist (because she has both `hpe-user` and `bu1-user` roles), but she can only 
read `bu1/hot-data` (because she has `bu1-user`) but not `bu1-raw` (because she 
doesn't have the role `bu1/bu1-aggregator`).

The workload `bu1/ingester`, on the other hand, can see `bu1/raw-data` exists 
(because it has `hpe-workload`) and can read `bu1-raw` (because it has 
`bu1-aggregator` role). The `bu1/ingester` role can see none of the roles any 
workload, or any user might have. It can't even see anything about itself. 

It should also be noted that none of these roles on users or workloads take 
precedence over any others except in the context of a particular permission 
expression.

