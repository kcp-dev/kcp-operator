---
description: >
    Step by step guide to get kcp-operator up and running for development purposes in a kind cluster.
---

# Local Setup

The files in [config/samples](https://github.com/kcp-dev/kcp-operator/tree/main/config/samples) showcase kcp-operator resources but have been crafted to get a kind setup up and running quickly.

Also check out the [Quickstart guide](../setup/quickstart.md) for more information on getting a first kcp setup up and running.

## Prerequisites

- A local copy of the kcp-operator repository
- [kind](https://kind.sigs.k8s.io/)

To make DNS working from your local machine, it is necessary to create an entry in your `/etc/hosts` (or corresponding OS mechanism):

```
127.0.0.2 example.operator.kcp.io
```

## Prepare Environment

First, create a kind cluster if you do not have one yet:

```sh
kind create cluster
```

Install cert-manager, it is required to create kcp's PKI:

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

Set up two etcd instances, one for the root shard and one for a supplementary shard:

```sh
helm install etcd oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
helm install etcd-shard oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
```

Create a "self-signed" cert-manager issuer:

```sh
kubectl apply -f config/samples/cert-manager/issuer.yaml
```

## Run Operator

Now the operator needs to be started. You can either deploy the operator into the cluster to ensure
the built container image behaves as intended (e.g. has the necessary RBAC, etc) or -- for rapid development --
run the operator as a binary.

### Option 1: Deploy Operator

Build the image:

```sh
export IMG=ghcr.io/kcp-dev/kcp-operator:local
make docker-build
```

Load the image into the kind cluster:

```sh
kind load docker-image "$IMG"
```

Deploy the operator manifests into the cluster:

```sh
make deploy
```

### Option 2: Run Operator Directly

Alternatively, apply the CRDs to the cluster:

```sh
kubectl apply -k ./config/crd/
```

Then start the operator via `go run`:

```sh
go run ./cmd/operator/
```

## Create kcp Instance

Now you can create a root shard:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_rootshard.yaml
```

Create the additional shard:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_shard.yaml
```

Create the front-proxy instance:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_frontproxy.yaml
```

Finally, let's create a kubeconfig that we can use to access the kcp
environment via its front-proxy:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_kubeconfig.yaml
```

## Connect to kcp

Once the kubeconfig above has been reconciled, we can use it to connect to kcp.

First, fetch the created kubeconfig:

```sh
kubectl get secret sample-kubeconfig -o jsonpath="{.data.kubeconfig}" | base64 -d > admin.kubeconfig
```

Create a port-forwarding in a second terminal:

```sh
kubectl port-forward svc/frontproxy-sample-front-proxy 6443 --address=127.0.0.2
```

Use the new 'admin.kubeconfig` to connect:

```sh
export KUBECONFIG=$PWD/admin.kubeconfig
kubectl get ws
```
