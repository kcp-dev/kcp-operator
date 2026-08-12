---
description: >
    Explains how virtual workspace servers are deployed, including servers that are not part of kcp.
---

# Virtual Workspaces

A `VirtualWorkspace` deploys a virtual workspace server as its own `Deployment`, separate from the
shards. By default it runs kcp's own virtual-workspace server, but the same object can deploy any
server built on the virtual workspace framework.

## Architecture

A virtual workspace server is an aggregated apiserver placed between clients and kcp: it terminates
TLS with its own serving certificate, identifies callers, and talks to a shard with its own
credentials. The kcp-operator provisions all of that from the CA hierarchy of the `RootShard` named
in `spec.target`:

* a serving certificate, issued by the shard's server CA and valid for the server's in-cluster
  service name,
* the client CA and the requestheader client CA, so callers can be authenticated either by their
  client certificate or by the identity headers the front-proxy forwards,
* a client certificate, mounted at the path the target's logical-cluster-admin kubeconfig expects,
  so the server can reach kcp's APIs.

The resulting `Service` listens on port 6443.

`spec.target` decides what the server is connected to. Use `shardRef` for a per-shard virtual
workspace, which means one `VirtualWorkspace` object per shard plus one for the root shard. Use
`rootShardRef` for a singleton virtual workspace: a single deployment that serves the whole
installation and connects to the root shard to discover the other shards.

## Custom virtual workspaces

Servers that are not kcp's own — the ones under
[kcp-dev](https://github.com/orgs/kcp-dev/repositories?q=virtual-workspace), or your own — need two
things beyond a different image.

Their binary is not at `/virtual-workspaces`, so `spec.command` has to name it. That is all
`spec.command` does — it picks the binary and nothing else.

The operator generates only the arguments every aggregated apiserver accepts, so a server plugged
in here starts without having to know anything about kcp:

```
--tls-cert-file         --requestheader-client-ca-file        --client-ca-file
--tls-private-key-file  --requestheader-allowed-names         --kubeconfig
--bind-address          --requestheader-username-headers      -v
--secure-port           --requestheader-group-headers
                        --requestheader-extra-headers-prefix
```

Anything beyond that is `spec.extraArgs`, including flags that kcp's own server takes. The operator
does not pass them, because it cannot know which binary `spec.command` names or which flags that
binary understands — and an apiserver built on `pflag` exits on a flag it does not know, before it
ever serves.

!!! note
    This applies to kcp's own virtual-workspace server too. It takes `--shard-external-url` on kcp
    releases before 0.31, where the flag was required, and `--cache-kubeconfig` where a cache
    server is configured. Both go in `spec.extraArgs`:

    ```yaml
    spec:
      extraArgs:
        - --shard-external-url=https://127.0.0.1:6443    # kcp < 0.31 only; unused, any URL will do
        - --cache-kubeconfig=/etc/cache-server/kubeconfig/kubeconfig
    ```

The cache server's kubeconfig, CA and client certificate are still mounted whenever a cache server
is configured, at the fixed paths above, so `--cache-kubeconfig` has something to point at without
you having to name the Secret yourself.

Many of these servers also have to bootstrap themselves in kcp before they can serve, typically by
creating a workspace and installing an `APIExport` in it. `spec.initContainers` runs that first.
Init containers default to the server's own image, which is the common case for servers shipping
their bootstrapping binary alongside the server, and they inherit the certificates and CAs the
operator manages so they can reach them without knowing where the operator mounted them.
Bootstrapping usually needs different credentials than serving, though — see
[Choosing the credentials](#choosing-the-credentials).

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: VirtualWorkspace
metadata:
  name: access
  namespace: example
spec:
  target:
    rootShardRef:
      name: my-root
  external:
    hostname: kcp.example.com
    port: 6443

  image:
    repository: ghcr.io/kcp-dev/contrib-access-virtual-workspace
    tag: latest
  command:
    - /access-vw

  initContainers:
    - name: init
      command:
        - /access-vw-init
      args:
        - --workspace-prefix=root:access
        - --controllers-workspace=controllers

  extraArgs:
    - --apiexport-endpointslice=access.contrib.kcp.io
    - --endpoint-base=https://kcp.example.com:6443/clusters/
```

## Choosing the credentials

Two fields decide what each container authenticates to kcp as: `spec.kubeconfigSecretRef` for the
server, and `kubeconfigSecretRef` on an individual init container. Each Secret, normally produced by
a `Kubeconfig` object, is mounted only into the container that asked for it — at
`/etc/kcp/server-kubeconfig/kubeconfig` and `/etc/kcp/init-kubeconfig/kubeconfig`. The server's
`--kubeconfig` is repointed automatically; an init container names the path in its own args.

Set them, because one credential rarely suits both containers. Bootstrapping walks the workspace
tree and so has to go through the front-proxy, which is the only thing that resolves workspace paths
across shards; serving usually only reads a few objects and should hold far less than an
administrator. A `kcp-admin` in the `system:kcp:admin` group covers the first, since kcp's bootstrap
policy binds that group to `cluster-admin`.

Left unset, both containers fall back to the logical-cluster-admin kubeconfig, which is a broadly
privileged credential aimed straight at one shard. Retargeting *that* at the front-proxy is not a
fix — the proxy strips its privileged group on ingress, so it arrives with no rights at all.

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: access-vw-bootstrap
  namespace: example
spec:
  target:
    frontProxyRef:            # the workspace tree only resolves here
      name: my-front-proxy
  username: kcp-admin
  groups:
    - system:kcp:admin
  validity: 8766h
  secretRef:
    name: access-vw-bootstrap-kubeconfig
---
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: access-vw-server
  namespace: example
spec:
  target:
    frontProxyRef:
      name: my-front-proxy
  targetWorkspace: root:access:controllers
  username: access-vw         # no groups at all
  validity: 8760h
  secretRef:
    name: access-vw-server-kubeconfig
  authorization:
    clusterRoleBindings:
      clusterRoles:
        - access-vw-controller
---
apiVersion: operator.kcp.io/v1alpha1
kind: VirtualWorkspace
metadata:
  name: access
  namespace: example
spec:
  # ...

  kubeconfigSecretRef:
    name: access-vw-server-kubeconfig

  initContainers:
    - name: init
      kubeconfigSecretRef:
        name: access-vw-bootstrap-kubeconfig
      command:
        - /access-vw-init
      args:
        - --kubeconfig=/etc/kcp/init-kubeconfig/kubeconfig
```

`clusterRoles` binds the identity to `ClusterRoles` that already exist in the target workspace, so
the server's permissions are whatever that role grants. If the role is installed by the init
container, the binding stays pending until the virtual workspace has run once.

!!! warning
    A role that grants only the resources the server reads is not enough. kcp gates every request
    on `verb=access` for the non-resource URL `/` in the workspace, *before* any RBAC on the
    resource is consulted, so an identity without it can do nothing at all:

    ```yaml
    rules:
      - verbs: ["access"]
        nonResourceURLs: ["/"]
    ```

    ServiceAccounts declared inside the workspace are exempt from that gate, which is why roles
    written for a ServiceAccount often omit the rule and then appear to grant nothing when bound to
    the certificate identity a `Kubeconfig` mints.

Provisioning the binding through `spec.authorization` also ties the `Kubeconfig`'s lifetime to its
target: the cleanup finalizer has to reach kcp to remove the `ClusterRoleBinding` again, so deleting
the front-proxy or shard first leaves the `Kubeconfig` unfinalizable — and its namespace stuck in
`Terminating`. Where the bootstrapping already has admin rights, having it create its own binding
avoids that.

Two more things follow from how these kubeconfigs are generated. They are self-contained — the
client certificate, key and CA are embedded rather than referenced as paths — so the Secret mounts
anywhere. And their current context is already scoped to `spec.targetWorkspace`, so a server should
not retarget it again; with the access virtual workspace that means dropping `--workspace-path`,
which would otherwise append a second `/clusters/` segment. A front-proxy-targeted kubeconfig also
addresses the front-proxy by its **external** hostname, which therefore has to resolve from inside
the pod — add a `hostAliases` entry via `spec.deploymentTemplate` where it does not.

## Extra volumes

The `VirtualWorkspace` adds to what the operator already mounts, so its fields are called
`extraVolumes` and `extraVolumeMounts`. An init container is written as a container in its own
right — `image`, `command`, `args`, `resources` — so there the fields are plainly `volumes` and
`volumeMounts`.

Volumes are pod-scoped wherever they are declared, so any container can mount any of them; mounts
are not, and they are deliberately never shared. The server container gets
`spec.extraVolumeMounts`, an init container gets its own `volumeMounts`, and neither sees the
other's. Only the certificates and kubeconfigs the operator manages are mounted everywhere.

That split is what makes a handoff possible — the init container writes somewhere the server later
reads, without the server holding a mount it has no business having:

```yaml
spec:
  # ...

  extraVolumes:                 # declared once, mounted by both
    - name: bootstrap-state
      emptyDir: {}

  extraVolumeMounts:            # the server reads it
    - name: bootstrap-state
      mountPath: /var/lib/bootstrap
      readOnly: true

  initContainers:
    - name: init
      volumes:                  # only this container needs a cache
        - name: build-cache
          emptyDir: {}
      volumeMounts:
        - name: bootstrap-state # the init container writes it
          mountPath: /var/lib/bootstrap
        - name: build-cache
          mountPath: /cache
```

Where a volume is declared makes no difference to the Pod, only to where it reads best: put it on
the `VirtualWorkspace` when more than one container mounts it, and on the init container when only
that container does. A name declared in several places is emitted once — the first declaration
wins, in the order `VirtualWorkspace`, then init containers — since a Pod listing the same volume
name twice is rejected.

## Routing traffic

Clients reach a virtual workspace through the [front-proxy](front-proxy.md), which needs a mapping
for the path prefix the server owns:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: FrontProxy
metadata:
  name: my-front-proxy
  namespace: example
spec:
  # ...

  additionalPathMappings:
    - path: /services/access
      backend: https://access-virtual-workspace.example.svc.cluster.local:6443
      backend_server_ca: /etc/kcp-front-proxy/tls/ca/tls.crt
      proxy_client_cert: /etc/kcp-front-proxy/requestheader-client/tls.crt
      proxy_client_key: /etc/kcp-front-proxy/requestheader-client/tls.key
```

The backend is the `Service` created for the `VirtualWorkspace`, named `<name>-virtual-workspace`.
Its serving certificate chains up to the root CA the front-proxy already mounts, so no additional CA
is needed.

The default mappings already route `/services/` to the root shard. This does not shadow the entry
above: the front-proxy matches the longest prefix, so the more specific path wins no matter in which
order the mappings appear.

Requests arriving this way are authenticated by the front-proxy, which forwards the caller's
identity in `X-Remote-*` headers. The operator configures the virtual workspace to trust those
headers only when they come with the front-proxy's own client certificate.
