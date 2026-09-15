# Virtual workspace samples

Samples for running the three contrib virtual workspace servers through the operator's
`VirtualWorkspace` API, on top of the base samples in the parent directory (RootShard,
FrontProxy, etcd, cert-manager Issuer). The RootShard sample pins
`ghcr.io/kcp-dev/kcp:04fcc9232`, a build carrying the in-flight virtual-workspace changes
(`SelfClusterAccessReview`, ephemeral resources) these servers depend on.

| Sample | Upstream | What it serves |
| --- | --- | --- |
| [access.yaml](access.yaml) | [contrib-access-virtual-workspace](https://github.com/kcp-dev/contrib-access-virtual-workspace) | `SelfClusterAccessReview`: which workspaces can this caller see. |
| [mcp.yaml](mcp.yaml) | [contrib-mcp-virtual-workspace](https://github.com/kcp-dev/contrib-mcp-virtual-workspace) | Model Context Protocol, scoped per caller. Depends on access.yaml. |
| [ephemeral-resources.yaml](ephemeral-resources.yaml) | [contrib-virtual-ephemeral-resources-virtual-workspace](https://github.com/kcp-dev/contrib-virtual-ephemeral-resources-virtual-workspace) | Webhook-backed resources that are answered, never stored. |

Deploy access.yaml before mcp.yaml — the MCP server asks the access virtual workspace for the
caller's workspace list on every request.

Each sample ends with a `FrontProxy` object carrying only its own `additionalPathMappings` entry,
to stay self-contained. When deploying more than one, merge the path mappings into a single
FrontProxy object — applying the samples as-is would have the last one win.

## Caveats

* Since [#292](https://github.com/kcp-dev/kcp-operator/pull/292) the operator passes
  `--shard-external-url` (and `--cache-kubeconfig`, when a cache server is referenced) to custom
  servers as well as to kcp's own. A server that rejects unknown flags will not start until it
  tolerates them; there is no API field to switch this off yet.
* The ephemeral-resources repository publishes no container image yet; build and push your own.
  Its sample also needs provider-side bootstrapping in kcp (endpoint slice CRD, APIExport, RBAC)
  and the repository's `endpointslice-controller` running against the provider workspace — see the
  comments in the sample and `docs/deployment.md` upstream.
