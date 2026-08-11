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

Their binary is not at `/virtual-workspaces`, so `spec.command` has to name it. Setting
`spec.command` also tells the operator that this is not kcp's server, and it stops generating the
arguments that only kcp's server accepts (`--shard-external-url` and `--cache-kubeconfig`, along
with the cache server's kubeconfig mount). That matters because an apiserver rejects flags it does
not know and exits before it ever serves. Everything an aggregated apiserver does understand — the
serving certificate, the client and requestheader CA configuration, the bind address and port, and
`--kubeconfig` — is still passed, and `spec.extraArgs` covers the rest.

Many of these servers also have to bootstrap themselves in kcp before they can serve, typically by
creating a workspace and installing an `APIExport` in it. `spec.initContainers` runs that first.
Init containers default to the server's own image, which is the common case for servers shipping
their bootstrapping binary alongside the server, and they inherit the server container's volume
mounts so they can reach the same certificates without knowing where the operator mounted them.
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
