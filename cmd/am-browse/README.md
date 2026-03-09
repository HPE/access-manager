# Access Manager Web Interface
This directory contains a simple web UI that lets you browse metadata
in the access manager.

To run this UI, make sure you have `etcd` and the access manager running. 
The access manager should be using default port. Then use
```shell
go build am-browse
```
from this directory and open `http://localhost:8081` and follow on-page
documentation from there.