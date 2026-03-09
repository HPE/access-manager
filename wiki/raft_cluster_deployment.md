
## Points to note:
* The uniqueness of a node in the Raft cluster is defined by two factors: the node's IP and the data directory.
* Even if you remove a node from the cluster, it's IP is retained. Which means you can't rejoin or create a new node with the same IP.
* Attempting to add a new node with an IP that already exists in the cluster, even if that IP is associated with a removed existing node, will result in failure.
* One solution would be to not use POD_IP and instead use HOSTNAME as it will be unique for each pod.
* It is crucial to maintain this uniqueness during rolling deployments to avoid the need for node removal.
* For instance, in Kubernetes StatefulSets, during a redeployment, the rolling update removes an existing pod and recreates it with the same IP and data volume.
* However, if a Deployment is used instead of StatefulSet, the rolling update might create a new pod with a different IP and data volume.
* In such cases, it is necessary to ensure that the terminating pod is removed from the cluster, when being replaced by an entirely new pod.
* This means that when we are scaling down, we'd need to remove those pods from manually(or script) using RemoveNode RPC.

## Steps to deploy
* To create a fresh raft cluster, we need to do a k8s StatefulSet deployment under a service(metadata) with one pod with RAFT_CLUSTER_ADDRESS as empty. 
  It will initialize the cluster with one pod as a leader. 
* Now, the next deployment will be usual and here we can assign 2 more pods under the same StatefulSet with RAFT_CLUSTER_ADDRESS as the service:port,
  it should add 2 new nodes and join the cluster.
* Few things to note - RAFT_ADDRESS is the address of each node, we should populate this env variable with the hostname of the pod. `spec.hostname`. It should not be
  the POD_IP because of the reason mentioned #Points to note section
* Provide the METADATA_DNS in access manager and audit manager as `multi:///dns://service:port`
* If we are not using StatefulSet and instead using Deployment in k8s then the new pods will always have new state(eg. hostname, volume etc)
  for this case, we should enable RAFT_REMOVE_NODE = true. It will remove the pod from the cluster on termination as the new pod will replace it.

## Scale up/down k8s pods
* Scaling up pods is a straightforward process – simply add new nodes, and they will seamlessly join the cluster.
* Scaling down, however, may necessitate a distinct pipeline. This pipeline first invokes the ListNode RPC to identify the nodes to be removed, followed by calling RemoveNode to eliminate the specified number of nodes.
  Subsequently, these pods are removed from Kubernetes. Note that this step might be omitted if we use normal Deployment instead of StatefulSet, and RAFT_REMOVE_NODE is enabled.
* An important consideration is that when scaling back up after a previous scale-down operation, Kubernetes Statefulset should avoid reusing the persistence volume of terminated pods from the scale-down. To ensure a clean scaling process,
  new pods initiated during scaling should always have a fresh data directory.

## Changes made in Raft library under pkg/raft
* Integrated the https://github.com/hpe/raft wrapper, based on etcd, into the codebase under pkg/raft.
* Introduced support for BoltDB as an additional storage implementation, expanding the storage options beyond the default Write-Ahead Logging (WAL) type.
* Extended the StateMachine interface by adding a new method named AppliedEntry. Unlike the existing Apply method, which only provides CacheUpdate data, this new method offers access to the actual entry.
* Addressed a limitation in the node removal process. Previously, a check was in place to ensure that any new node attempting to join the cluster had a different IP address and nodeID than existing nodes.
  This restriction caused issues when a node was removed, and subsequently, a fresh node with a distinct ID but the same IP address attempted to join. Modified the addressInUse function to include the condition m.Type() != removed,
  allowing a fresh node with a different ID to join even if the IP address was previously associated with a removed node.
* Make sure we import github.com/Jille/grpc-multi-resolver whenever we are using multi:/// in the DNS name
