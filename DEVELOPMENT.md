# Development

## kubebuilder

For historic documentation, this project has been generated with [kubebuilder](https://book.kubebuilder.io/) with the following version information:

```sh
$ kubebuilder version
Version: main.version{KubeBuilderVersion:"4.3.1", KubernetesVendor:"unknown", GitCommit:"a9ee3909f7686902879bd666b92deec4718d92c9", BuildDate:"2024-11-09T09:58:43Z", GoOs:"darwin", GoArch:"arm64"}
```

The project has been initialised with:

```sh
$ kubebuilder init --domain operator.kcp.io --owner "The KCP Authors" --project-name kcp-operator
```


## Kind setup

```sh
kind create cluster
```

Install cert-manager:

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

Install etcd:

```sh
helm install etcd oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
helm install etcd-shard oci://registry-1.docker.io/bitnamicharts/etcd --set auth.rbac.enabled=false --set auth.rbac.create=false
```

Create issuer:

```sh
kubectl apply -f config/samples/cert-manager/issuer.yaml
```

Build the image:

```sh
make docker-build IMG=ghcr.io/kcp-dev/kcp-operator:1
kind load docker-image ghcr.io/kcp-dev/kcp-operator:1
```

Load the image into the kind cluster:

```sh
make deploy IMG=ghcr.io/kcp-dev/kcp-operator:1
```

Now you can create a root shard:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_rootshard.yaml       
```

Shards:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_shard.yaml
```

Frontproxy:

```sh
kubectl apply -f config/samples/operator.kcp.io_v1alpha1_frontproxy.yaml
```