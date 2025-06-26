---
description: >
    Create your first objects after installing kcp-operator.
---

# Quickstart

kcp-operator has to be installed according to the instructions given in [Setup](./index.md) before starting the steps below.

## etcd

!!! warning
    Never deploy etcd like below in production as it sets up an etcd instance without authentication or TLS.

Running a root shard requires a running etcd instance/cluster. A simple one can be set up with Helm and the Bitnami etcd chart:

```sh
helm install etcd oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
```

## Create Root Shard

In addition to a running etcd, the root shard requires a reference to a cert-manager `Issuer` to issue its PKI. Create a self-signing one:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
```

Afterward, create the first `RootShard` object. API documentation is available in the [CRD reference](../reference/crd/operator.kcp.io/rootshards.md).

The main change to make is replacing `example.operator.kcp.io` with a hostname to be used for the kcp instance. The DNS entry should not be set yet.

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
    # this references the issuer created above
    issuerRef:
      group: cert-manager.io
      kind: Issuer
      name: selfsigned
  cache:
    embedded:
      # kcp comes with a cache server accessible to all shards,
      # in this case it is fine to enable the embedded instance
      enabled: true
  etcd:
    endpoints:
      # this is the service URL to etcd. Replace if Helm chart was
      # installed under a different name or the namespace is not "default"
      - http://etcd.default.svc.cluster.local:2379
```

kcp-operator will create the necessary resources to start a `Deployment` of a kcp root shard and the necessary PKI infrastructure (via cert-manager).

## Set up Front Proxy

Every kcp instance deployed with kcp-operator needs at least one instance of kcp-front-proxy to be fully functional. Multiple front-proxy instances can exist to provide access to a complex, multi-shard geo-distributed setup.

For getting started, a `FrontProxy` object can look like this:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: FrontProxy
metadata:
  name: frontproxy
spec:
  rootShard:
    ref:
      # the name of the RootShard object created before
      name: root
  serviceTemplate:
    spec:
      # expose this front-proxy via a load balancer
      type: LoadBalancer
``` 

kcp-operator will deploy a kcp-front-proxy installation based on this and connect it to the `root` root shard created before.

### DNS Setup

Once the `Service` `<Object Name>-front-proxy` has successfully been reconciled, it should have either an IP address or a DNS name (depending on which load balancing integration is active on the Kubernetes cluster). A DNS entry for the chosen external hostname (this was set in the `RootShard`) has to be set and should point to the IP address (with an A/AAAA DNS entry) or the DNS name (with a CNAME DNS entry).

Assuming this is what the `frontproxy-front-proxy` `Service` looks like:

```sh
kubectl get svc frontproxy-front-proxy
```

Output should look like this:

```
NAME                     TYPE           CLUSTER-IP     EXTERNAL-IP                          PORT(S)          AGE
frontproxy-front-proxy   LoadBalancer   10.240.30.54   XYZ.eu-central-1.elb.amazonaws.com   6443:32032/TCP   3m13s
```

Now a CNAME entry from `example.operator.kcp.io` to `XYZ.eu-central-1.elb.amazonaws.com` is required.

!!! hint
    Tools like [external-dns](https://github.com/kubernetes-sigs/external-dns) can help with automating this step to avoid manual DNS configuration.

## Initial Access

Once deployed, a `Kubeconfig` object can be created to generate credentials to initially access the kcp setup. An admin kubeconfig can be generated like this:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: kubeconfig-kcp-admin
spec:
  # the user name embedded in the kubeconfig
  username: kcp-admin
  groups:
    # system:kcp:admin is a special privileged group in kcp.
    # the kubeconfig generated from this should be kept secure at all times
    - system:kcp:admin
  # the kubeconfig will be valid for 365d but will be automatically refreshed
  validity: 8766h
  secretRef:
    # the name of the secret that the assembled kubeconfig should be written to
    name: admin-kubeconfig
  target:
    # a reference to the frontproxy deployed previously so the kubeconfig is accepted by it
    frontProxyRef:
      name: frontproxy
```

Once `admin-kubeconfig` has been created, the generated kubeconfig can be fetched:

```sh
kubectl get secret admin-kubeconfig -o jsonpath="{.data.kubeconfig}" | base64 -d > admin.kubeconfig
```

To use this kubeconfig, set the `KUBECONFIG` environment variable appropriately:

```sh
export KUBECONFIG=$(pwd)/admin.kubeconfig
```

It is now possible to connect to the kcp instance and create new workspaces via [kubectl create-workspace](https://docs.kcp.io/kcp/latest/setup/kubectl-plugin/):

```sh
kubectl get ws
```

Initially, the command should return that no workspaces exist yet:

```
No resources found
```

To create a workspace, run:

```sh
kubectl create-workspace test
``` 

Output should look like this:

```
Workspace "test" (type root:organization) created. Waiting for it to be ready...
Workspace "test" (type root:organization) is ready to use.
```

Congratulations, you've successfully set up kcp and connected to it! :tada:

<!-- TODO(embik):
## Optional: Additional Shards

kcp can be sharded, so kcp-operator supports joining additional kcp instances to the existing setup to act as shards.
-->

## Further Reading

- Check out the [CRD documentation](../reference/index.md) for all configuration options.
