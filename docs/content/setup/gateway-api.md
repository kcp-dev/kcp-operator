---
description: >
    Expose a whole kcp installation — front-proxy, shards and virtual workspaces —
    through a single Gateway API gateway with SNI-based host routing.
---

# Exposing kcp via the Gateway API

The [Quickstart](./quickstart.md) exposes kcp with one `LoadBalancer` Service per
component. That works, but a sharded installation quickly accumulates external
IPs: the front-proxy needs one, and every shard whose `shardBaseURL` should be
reachable from outside needs another.

An alternative is to put the whole installation behind a single [Gateway
API](https://gateway-api.sigs.k8s.io/) gateway and route by hostname: one
wildcard DNS entry, one external IP, one `TLSRoute` per component. This is the
setup kcp's own Tilt-based development environment uses
([contrib/tilt](https://github.com/kcp-dev/kcp/tree/main/contrib/tilt) in the
kcp repository), and this page walks through the same pattern end to end,
including the virtual workspace samples.

!!! tip "One-shot local setup"
    Everything below (on `kcp.localhost` hostnames, using a kind cluster) can
    be created in one go with a script from the kcp-operator repository:

    ```sh
    hack/gateway-api-setup.sh        # create kind cluster, gateway, kcp, kubeconfig
    hack/gateway-api-setup.sh down   # tear it down again
    ```

    It finishes with an admin kubeconfig in `./kcp-admin.kubeconfig` and a
    background port-forward to the gateway; the only manual step it may ask for
    is an `/etc/hosts` line, since `.localhost` names do not resolve
    automatically on every platform.

## Why TLS passthrough, not HTTPRoute

kcp authenticates clients with X.509 client certificates (admin kubeconfigs,
logical-cluster-admin connections between components, the front-proxy's
requestheader certificate). A gateway that terminates TLS consumes the client
certificate — everything behind it would see the gateway's identity instead of
the caller's, and mTLS between kcp components would break.

So the gateway must not terminate TLS. Instead it routes on the [SNI server
name](https://en.wikipedia.org/wiki/Server_Name_Indication) in the TLS
handshake and forwards the raw byte stream: a `Gateway` listener in
`Passthrough` mode plus one `TLSRoute` per hostname. `HTTPRoute` cannot be used
for this, since it requires terminating TLS to see the request.

The price of passthrough is that routing is per-hostname, not per-path. That
shapes the naming scheme:

| Hostname | Routed to | Used by |
|----------|-----------|---------|
| `kcp.example.com` | front-proxy Service | clients; also virtual workspace paths under `/services/...` |
| `root.kcp.example.com` | root shard Service | other shards, front-proxy, controllers reaching the shard directly |
| `alpha.kcp.example.com` | shard `alpha` Service | same, for the secondary shard |

A single wildcard DNS record (`*.kcp.example.com`) and a wildcard listener
cover all of them.

## Prerequisites

* kcp-operator and cert-manager, installed as described in [Setup](./index.md),
  and an etcd instance as in the [Quickstart](./quickstart.md).
* A Gateway API implementation that supports `TLSRoute`. `TLSRoute` is part of
  the Gateway API **experimental** channel (`v1alpha3`), so not every
  implementation ships it. This page uses [Envoy
  Gateway](https://gateway.envoyproxy.io/), which does, and whose Helm chart
  installs the required CRDs:

```sh
helm install envoy oci://docker.io/envoyproxy/gateway-helm \
  --version v1.7.0 \
  --namespace envoy-gateway-system --create-namespace
```

## Gateway

Create a `GatewayClass` and a `Gateway` with a single TLS passthrough listener
for the wildcard hostname:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: eg
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: eg
  namespace: envoy-gateway-system
spec:
  gatewayClassName: eg
  # Only needed on clusters without a load balancer implementation (e.g. kind),
  # see "In-cluster resolution" below. On a real cluster, remove this and let
  # the cloud load balancer assign an address.
  addresses:
    - type: IPAddress
      value: 10.96.2.2
  listeners:
    - name: passthrough
      protocol: TLS
      port: 6443
      tls:
        mode: Passthrough
      allowedRoutes:
        namespaces:
          from: All
```

!!! note
    The listener deliberately sets no `hostname`: a wildcard like
    `*.kcp.example.com` only covers one label and would not match the apex
    hostname `kcp.example.com` the front-proxy uses. Which hostname reaches
    which backend is decided by the `TLSRoute` objects below.

Wait for the gateway to be accepted:

```sh
kubectl -n envoy-gateway-system wait gateway/eg --for=condition=Programmed --timeout=5m
```

## Cache server

The Quickstart's single root shard can use kcp's embedded cache, but a
multi-shard installation cannot: the embedded cache is not reachable across
shards, so the shards need a dedicated cache server to see each other's
workspaces. The operator deploys one (with embedded etcd) from a minimal
object, and shards inherit it through the root shard reference:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: CacheServer
metadata:
  name: cache
spec:
  certificates:
    issuerRef:
      group: cert-manager.io
      kind: Issuer
      name: selfsigned
```

It is plain in-cluster infrastructure — no hostname, no `TLSRoute`.

!!! warning "One kcp version everywhere"
    If you pin a kcp image (`spec.image`) on the shards, pin the **same** image
    on the `CacheServer`, the `FrontProxy` and the root shard's internal proxy
    (`RootShard.spec.proxy.image`). Each object defaults to the operator's
    built-in kcp version independently, and a cache server on a different
    build bootstraps a different API set — the shards then never finish
    syncing their informers against it, and the failure surfaces only as
    `Failed to watch ... the server could not find the requested resource`
    in the shard logs.

## Root shard

Compared to the Quickstart, three things change: `shardBaseURL` gives the shard
its own externally routable hostname, the serving certificate must cover that
hostname, and no `LoadBalancer` Service is needed — a `TLSRoute` binds the
hostname to the shard's ClusterIP Service instead. The `cache` section
references the cache server from above instead of enabling the embedded one.

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: RootShard
metadata:
  name: root
spec:
  # The shard's own address, as published to other components (e.g. in
  # APIExportEndpointSlices and for shard-to-shard communication).
  shardBaseURL: https://root.kcp.example.com:6443
  # The client-facing address, i.e. the front-proxy behind the same gateway.
  external:
    hostname: kcp.example.com
    port: 6443
  certificates:
    issuerRef:
      group: cert-manager.io
      kind: Issuer
      name: selfsigned
  certificateTemplates:
    server:
      spec:
        dnsNames:
          # the serving certificate must cover the shard's public hostname
          - root.kcp.example.com
  cache:
    ref:
      name: cache
  etcd:
    endpoints:
      - http://etcd.default.svc.cluster.local:2379
---
apiVersion: gateway.networking.k8s.io/v1alpha3
kind: TLSRoute
metadata:
  name: root
  namespace: default
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - root.kcp.example.com
  rules:
    - backendRefs:
        # the operator names the root shard Service <name>-kcp, serving on 6443
        - name: root-kcp
          port: 6443
          namespace: default
```

## Additional shards

Each further shard follows the same pattern: its own hostname in
`shardBaseURL`, its own certificate DNS name, its own `TLSRoute`. The Service
is named `<name>-shard-kcp`.

Note what a `Shard` does *not* configure: it has no `certificates`/issuer or
`external` block of its own — its PKI, cache and client-facing address all
derive from the referenced `RootShard`. Only the serving certificate's SAN is
shard-local.

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: Shard
metadata:
  name: alpha
spec:
  rootShard:
    ref:
      name: root
  shardBaseURL: https://alpha.kcp.example.com:6443
  certificateTemplates:
    server:
      spec:
        dnsNames:
          - alpha.kcp.example.com
  etcd:
    endpoints:
      - http://etcd.default.svc.cluster.local:2379
---
apiVersion: gateway.networking.k8s.io/v1alpha3
kind: TLSRoute
metadata:
  name: alpha
  namespace: default
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - alpha.kcp.example.com
  rules:
    - backendRefs:
        - name: alpha-shard-kcp
          port: 6443
          namespace: default
```

!!! hint
    If several shards share one etcd cluster, isolate their keyspaces with
    `spec.extraArgs: ["--etcd-prefix=/shard/<name>"]`, as kcp's Tilt setup does.

## Front-proxy

The front-proxy keeps its default `ClusterIP` Service — no `LoadBalancer` — and
gets the client-facing hostname. The operator sets the Service port to the
external port (6443 here), which is what the `TLSRoute` references.

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: FrontProxy
metadata:
  name: frontproxy
spec:
  rootShard:
    ref:
      name: root
  external:
    hostname: kcp.example.com
    port: 6443
---
apiVersion: gateway.networking.k8s.io/v1alpha3
kind: TLSRoute
metadata:
  name: front-proxy
  namespace: default
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - kcp.example.com
  rules:
    - backendRefs:
        # the operator names the Service <name>-front-proxy; its port equals
        # spec.external.port
        - name: frontproxy-front-proxy
          port: 6443
          namespace: default
```

## In-cluster resolution

The hostnames are not only client-facing: shards dial each other and the
front-proxy dials the shards using the very same names (`shardBaseURL`,
`external.hostname`). They must therefore resolve *inside* the cluster too.

**On a real cluster** this is free: the wildcard DNS record points at the
gateway's load balancer, public DNS resolves in pods, done. Nothing below is
needed.

**On kind (or any cluster without a load balancer implementation)** there is no
address to resolve to. kcp's Tilt setup solves this with two tricks, shown in
the manifests above:

1. The `Gateway` pins a free IP from the cluster's Service CIDR
   (`addresses: [{type: IPAddress, value: 10.96.2.2}]`). Envoy Gateway copies
   this into the Envoy Service's `externalIPs`, which kube-proxy routes
   in-cluster even though no cloud load balancer exists — and it lets the
   Gateway report `Programmed`.
2. Every kcp component gets `hostAliases` mapping the hostnames to that IP:

```yaml
# add to the RootShard, every Shard and the FrontProxy
spec:
  deploymentTemplate:
    spec:
      template:
        spec:
          hostAliases:
            - ip: 10.96.2.2
              hostnames:
                - kcp.example.com
                - root.kcp.example.com
                - alpha.kcp.example.com
```

One deployment is easy to miss: the operator also runs an internal proxy for
the root shard (`<name>-proxy`), through which other shards reach it at
`shardBaseURL`. It is configured separately on the `RootShard`, and without
the mapping it crash-loops and the other shards never finish bootstrapping:

```yaml
# add to the RootShard
spec:
  proxy:
    deploymentTemplate:
      spec:
        template:
          spec:
            hostAliases:
              - ip: 10.96.2.2
                hostnames:
                  - kcp.example.com
                  - root.kcp.example.com
                  - alpha.kcp.example.com
```

From the developer machine, port-forward the Envoy Service and map the
hostnames to localhost. The Service name is generated, hence the label
selector:

```sh
kubectl -n envoy-gateway-system port-forward \
  "svc/$(kubectl -n envoy-gateway-system get svc \
    -l gateway.envoyproxy.io/owning-gateway-name=eg \
    -o jsonpath='{.items[0].metadata.name}')" 6443:6443
```

```
# /etc/hosts
127.0.0.1 kcp.example.com root.kcp.example.com alpha.kcp.example.com
```

## Access

From here the [Quickstart](./quickstart.md#initial-access) applies unchanged:
create a `Kubeconfig` targeting the front-proxy, extract the Secret and connect.
The generated kubeconfig points at `https://kcp.example.com:6443`, which now
resolves through the gateway:

```sh
kubectl get secret admin-kubeconfig -o jsonpath="{.data.kubeconfig}" | base64 -d > admin.kubeconfig
KUBECONFIG=admin.kubeconfig kubectl ws .
```

A request for a workspace on shard `alpha` is proxied by the front-proxy to
`alpha.kcp.example.com:6443` — through the gateway again, terminating at the
shard, with all client certificates intact.

## Virtual workspaces

Virtual workspace servers deployed through the operator's
[`VirtualWorkspace`](../reference/crd/operator.kcp.io/virtualworkspaces.md) API
need **no gateway configuration at all**: they are routed *by path*, not by
hostname. A client asks the front-proxy for `/services/<name>/...` on the
ordinary `kcp.example.com` hostname; the front-proxy's `additionalPathMappings`
forward the request to the virtual workspace's in-cluster Service (named
`<name>-virtual-workspace`, serving on 6443). SNI passthrough at the gateway is
what makes this work — the front-proxy still sees the caller's TLS connection
and forwards its identity via requestheader mTLS.

The repository ships three ready-made samples in
[`config/samples/virtual-workspaces/`](https://github.com/kcp-dev/kcp-operator/tree/main/config/samples/virtual-workspaces),
one per contrib virtual workspace server:

| Sample | Serves |
|--------|--------|
| `access.yaml` ([contrib-access-virtual-workspace](https://github.com/kcp-dev/contrib-access-virtual-workspace)) | `SelfClusterAccessReview` — which workspaces can this caller see |
| `mcp.yaml` ([contrib-mcp-virtual-workspace](https://github.com/kcp-dev/contrib-mcp-virtual-workspace)) | Model Context Protocol, scoped per caller (depends on `access.yaml`) |
| `ephemeral-resources.yaml` ([contrib-virtual-ephemeral-resources-virtual-workspace](https://github.com/kcp-dev/contrib-virtual-ephemeral-resources-virtual-workspace)) | webhook-backed resources that are answered, never stored |

To run them behind the gateway from this page, adjust each sample as follows:

* `spec.external` becomes the front-proxy's public address
  (`hostname: kcp.example.com`, `port: 6443`) — it is used to build the URLs
  kcp hands to clients.
* The `hostAliases` blocks (samples pin `example.operator.kcp.io` to a
  hard-coded front-proxy ClusterIP) become the same gateway mapping as above:
  the kubeconfigs minted for the servers address the front-proxy by hostname,
  so it has to resolve inside the virtual workspace pods too.
* The `additionalPathMappings` snippets each sample carries (`/services/access`,
  `/services/mcp`, `/services/ephemeral-buckets`) must be **merged into the one
  `FrontProxy` object** — each sample repeats the `FrontProxy` only to stay
  self-contained, and applying them verbatim would have the last one win.
* Hostname-derived arguments inside the samples follow along, e.g. the access
  sample's `--endpoint-base=https://kcp.example.com:6443/clusters/` and the MCP
  sample's `--access-url=https://kcp.example.com:6443/services/access`.

The shard image matters here: the samples pin `ghcr.io/kcp-dev/kcp:04fcc9232`
on the `RootShard`/`Shard` objects, a build carrying the in-flight
virtual-workspace changes (`SelfClusterAccessReview`, ephemeral resources) the
contrib servers depend on. Check the samples' comments and
[`README`](https://github.com/kcp-dev/kcp-operator/tree/main/config/samples/virtual-workspaces)
for the remaining per-server caveats (bootstrap init containers, impersonation
RBAC, the ephemeral server's provider-side setup).

## Complete working example

kcp's Tilt development environment is a maintained, running instance of this
exact topology — Envoy Gateway, TLS passthrough on `*.kcp.localhost`, a root
shard, a second shard and a front-proxy, all deployed through kcp-operator:
[kcp/contrib/tilt](https://github.com/kcp-dev/kcp/tree/main/contrib/tilt). Its
`gateway.yaml`, `shard-root.yaml`, `shard-theseus.yaml` and `front-proxy.yaml`
map one-to-one onto the sections above and are a good reference when something
does not connect.
