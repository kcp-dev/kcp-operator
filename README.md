# kcp-dev/kcp-operator

This is a work-in-progress operator to deploy and manage kcp instances. Please check back later!

## Requirements

- [cert-manager](https://cert-manager.io/)

## Quickstart

### RootShard

Running a root shard requires a running etcd instance/cluster. You can set up a simple one via Helm (THIS IS INSECURE, NEVER DEPLOY ETCD LIKE THIS IN PRODUCTION):

```sh
$ helm install etcd oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
```

In addition, the root shard requires a reference to a cert-manager `Issuer` to issue its PKI CAs. You can create a self-signing one:

```sh
$ kubectl apply -f ./config/samples/cert-manager/issuer.yaml
```

Afterward, create the `RootShard` sample object:

```sh
$ kubectl apply -f ./config/samples/v1alpha1_rootshard.yaml
```

kcp-operator will create the necessary resources to start a `Deployment` of a kcp root shard.
