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

!!! warning
    Currently there is no support for persistence in kcp's cache-server, so whenever the Pod is
    restarted, it will have to be re-filled over time by the kcp shards.

## Architecture

The cache server is a purely passive server: It doesn't connect to any shard or proxy, but instead
all the kcp shards connect to it, in order to store their data on the cache. This makes the server
conceptually very simple, since it all it needs to function is a TLS serving certificate.

As of kcp 0.30, there is only a single cache server supported for a whole kcp installation
consisting of multiple shads).

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
