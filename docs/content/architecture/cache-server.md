---
description: >
    Explains how to use an external cache server in a kcp installation.
---

# kcp cache-server

The kcp cache-server is used to coordinate all shards within one kcp installation. It holds data used
by all of them, like `APIExports`, `Workspaces` and RBAC to some degree. If no standalone cache is
configured, then the kcp root shard will run an in-process cache that uses the root shard's etcd as
its storage backend. However once multiple shards are involved, a standalone cache server should be
provisioned.

## Architecture

The cache server is a purely passive server: It doesn't connect to any shard or proxy, but instead
all the kcp shards connect to it, in order to store their data on the cache. This makes the server
conceptually very simple, since all it needs to function is a TLS serving certificate.

## Usage

To deploy a cache server, create a `CacheServer` object:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: CacheServer
metadata:
  name: my-cache-server
  namespace: example
spec:
  certificates:
    issuerRef:
      group: cert-manager.io
      kind: ClusterIssuer
      name: selfsigned
```

The kcp-operator will provision a root CA and a serving certificate for the cache server. For this,
similar to how `RootShards` work, you need to configure either your desired `ClusterIssuer` or provide
a pre-existing CA certificate and key as a `Secret`.

The CacheServer must be referenced in the RootShard and Shard objects, otherwise the embedded cache
is enabled:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: RootShard
metadata:
  name: my-root
  namespace: example
spec:
  # ...

  cache:
    ref:
      name: my-cache-server

  # ...
```

Once applied, the kcp-operator will reconfigure the shard `Deployments` accordingly and briefly after
that, the cache server will be filled with data.

By default, the cache-server is run with an embedded (in-process) etcd store which has implications
on persistance and availability:

* embedded etcd uses ephemeral store, and will not retain its data across cache-server Pod deletions,
* CacheServer deployment with an embedded etcd store may not scale to more than one replice.

This is usually sufficient for development environments. For production environments, please see
the [High availability](#high-availability) section below.

## High availability

It is recommended to run the cache-server with more than a single replica for high availability and
load balancing.

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: CacheServer
metadata:
  name: my-cache-server
  namespace: example
spec:
  # ...

  replicas: 2

  etcd:
    tlsConfig:
      secretRef:
        name: cache-etcd-client-tls
    endpoints:
    - https://cache-etcd-0.cache-etcd.example.svc.cluster.local:8379

  # ...
```

With multiple replicas, the CacheServer must be configured with an external etcd store. Each replica then
communicates with the same etcd store.

!!! warning
    Currently, all CacheServer deployments in a kcp installation must use the same instance of the etcd store.
    The user is responsible for making sure the instance is able to span the whole kcp installation.
