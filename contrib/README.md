# contrib

A local kcp installation with virtual workspaces on top of it, for development
and for the `contrib e2e` GitHub workflow. Everything here builds a throwaway
kind cluster; none of it is meant for a real deployment.

The prose version of what it builds is
[docs/content/setup/gateway-api.md](../docs/content/setup/gateway-api.md).

## Running it

```sh
contrib/deploy.sh                 # kcp, then every virtual workspace
contrib/deploy.sh down            # tear everything down
contrib/deploy.sh kcp             # only the kcp layer
contrib/deploy.sh vw              # only the virtual workspaces
contrib/deploy.sh vw access mcp   # only these virtual workspaces
```

Each layer is also runnable on its own, which is the point of the split:
iterating on one virtual workspace does not mean rebuilding the cluster.

```sh
contrib/kcp/deploy.sh
contrib/virtualworkspaces/deploy.sh
contrib/virtualworkspaces/mcp/deploy.sh
```

It finishes with an admin kubeconfig at `./kcp-admin.kubeconfig` and a
background port-forward to the gateway. `.localhost` names do not resolve
automatically everywhere; if they do not, the script prints the `/etc/hosts`
line to add rather than editing it for you.

## Layout

| Path | What it owns |
| --- | --- |
| `lib.sh` | Configuration and helpers, sourced by everything else. |
| `templates/` | Manifests more than one component needs — currently the `Workspace` behind `ensure_workspace`. |
| `deploy.sh` | Entry point; sequences the layers below. |
| `kcp/` | The kind cluster, cert-manager, Envoy Gateway, kcp-operator, etcd, and the kcp objects themselves. |
| `virtualworkspaces/deploy.sh` | Runs each virtual workspace in `VIRTUAL_WORKSPACES` order. |
| `virtualworkspaces/<name>/` | One virtual workspace: its `deploy.sh` and its `templates/`. |

Every `deploy.sh` is both executable and sourceable. Run it directly and it
does its work; source it and you get its functions without side effects, which
is how the orchestrators compose them.

## Adding a virtual workspace

1. Create `virtualworkspaces/<name>/` with a `deploy.sh` and a `templates/`
   directory.
2. Define one function named `vw_<name>_deploy`, with dashes in the directory
   name becoming underscores — `ephemeral-resources` →
   `vw_ephemeral_resources_deploy`. `virtualworkspaces/deploy.sh` resolves it by
   that convention and fails loudly if it is missing.
3. Add the directory name to `VIRTUAL_WORKSPACES` in `lib.sh`.

Nothing else needs editing. Order in `VIRTUAL_WORKSPACES` is meaningful: MCP
calls the access virtual workspace on every request, so access is listed first.

## Templates

Manifests are literal YAML under `templates/`, rendered with `envsubst` and
applied by `apply_template`. Keeping them out of heredocs means an editor can
lint them and `kubectl apply --dry-run` can read them directly.

`envsubst` understands `$VAR` and `${VAR}` and nothing else. Bash parameter
expansion — `${IMAGE%:*}` to split a tag off a reference, say — silently does
not work; compute the pieces in `deploy.sh` and export them separately, as
`ephemeral-resources` does for `EPHEMERAL_IMAGE_REPOSITORY` and
`EPHEMERAL_IMAGE_TAG`.

## Helpers worth knowing

`lib.sh` carries a few that exist because of specific failures:

- **`ensure_workspace <parent> <name>`** — creates a kcp workspace pinned to the
  root shard and waits for Ready. Use it rather than assuming another component
  made the workspace: more than one binds into `root:consumer`, and whichever
  runs first has to create it. A missing workspace surfaces as kubectl's
  unhelpful `failed to download openapi: unknown`.
- **`retry_quiet`** — like `retry`, but shows the command's output only if every
  attempt fails. Use it for applies that are expected to fail for a while;
  plain `retry` prints the same multi-line error on every attempt and buries the
  one that mattered.
- **`apply_template`** — render and apply with retries, for manifests going to
  the management cluster.
- **`kcpctl <workspace> …`** — kubectl against a kcp workspace through the
  front-proxy.

## Configuration

Every value in `lib.sh` is overridable from the environment. The ones worth
knowing:

| Variable | Default | Notes |
| --- | --- | --- |
| `CLUSTER_NAME` | `kcp-gateway` | kind cluster name. |
| `BASE_DOMAIN` | `kcp.localhost` | Hostname kcp is reached on. |
| `PORT` | `6443` | Gateway port, and the local port the forward binds. |
| `GATEWAY_IP` | `10.96.2.2` | Must be free in the cluster's service CIDR. |
| `KCP_IMAGE_REPOSITORY` / `KCP_IMAGE_TAG` | `ghcr.io/kcp-dev/kcp:04fcc9232` | kcp image. |
| `KCP_REPLICAS` | `1` | Replicas per shard/front-proxy/cache server. The operator defaults to 2, which does not fit a 4-vCPU CI runner. |
| `OPERATOR_IMAGE` | built from this checkout | Set to skip the operator build. |
| `VIRTUAL_WORKSPACES` | `access ephemeral-resources mcp` | Which ones to deploy, in order. |
| `EPHEMERAL_VW_DIR` | `../contrib-virtual-ephemeral-resources-virtual-workspace` | Checkout supplying that virtual workspace's `docs/example` manifests; it skips itself if absent. |
| `EPHEMERAL_IMAGE` | published `:latest` | Override to test a locally built image; load it into the kind cluster yourself. |
| `KUBECONFIG_OUT` | `./kcp-admin.kubeconfig` | Where the admin kubeconfig is written. |

## Notes for anyone debugging this

**`$BASE_DOMAIN:$PORT` is a shared local port.** It resolves to `127.0.0.1`, so
whichever process bound the port first answers for it. A stale
`kubectl port-forward` against an unrelated cluster will answer every probe here
and serve a completely different kcp — which surfaces as workspaces that are
never Ready rather than as a connection error. `assert_port_free` in `lib.sh`
catches that at startup; do not remove it.

**The front-proxy is taught to trust the shards' forwarded identity**
(`kcp_trust_shard_identity`). Without it, a shard proxying to a virtual
workspace loses the caller and every request arrives as the shard's certificate
common name. It works, and it makes the client CA an impersonation CA for those
names — read the comment before copying the pattern anywhere real.
