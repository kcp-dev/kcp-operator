---
description: >
    Take your first steps after installing kcp-operator.
---

# Quickstart

Make sure you have kcp-operator installed according to the instructions given in [Setup](./index.md).

## RootShard

!!! warning
    Never deploy etcd like below in production as it sets up an etcd instance without authentication or TLS.

Running a root shard requires a running etcd instance/cluster. You can set up a simple one via Helm:

```sh
$ helm install etcd oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
```

In addition, the root shard requires a reference to a cert-manager `Issuer` to issue its PKI CAs. You can create a self-signing one:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
```

Afterward, create a `RootShard` object. You can find documentation for it in the [CRD reference](../reference/crd/operator.kcp.io/rootshards.md).

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: RootShard
metadata:
  name: root
spec:
  external:
    # replace the hostname with the external DNS name for your kcp instance
    hostname: example.operator.kcp.io
    port: 6443
  certificates:
    issuerRef:
      group: cert-manager.io
      kind: Issuer
      name: selfsigned
  cache:
    embedded:
      enabled: true
  etcd:
    endpoints:
      - http://etcd.default.svc.cluster.local:2379
```

kcp-operator will create the necessary resources to start a `Deployment` of a kcp root shard.
