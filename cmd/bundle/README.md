# Bundle Applier Tool

The bundle applier is a command-line tool that continuously reads bundle Secrets from a source Kubernetes cluster and applies their contents to a target cluster.

## Overview

This tool is designed to work with the kcp-operator's bundle system, where Kubernetes resources are stored in Secret objects and need to be applied to different clusters. The applier runs in a continuous loop, reconciling bundle contents at regular intervals.

## Building

```bash
go build -o bundle-applier ./cmd/bundle/
```

## Usage

```bash
./bundle-applier \
  --kubeconfig=/path/to/source/kubeconfig \
  --target-kubeconfig-path=/path/to/target/kubeconfig \
  --bundle-name=bundle-rootshard-root \
  --bundle-namespace=kcp-vespucci \
  --reconcile-interval=30s
```

```bash
go run ./cmd/bundle/ \
  --kubeconfig=$HOME/.garden/prod.kubeconfig   \
  --target-kubeconfig-path=$HOME/Downloads/kubeconfig-gardenlogin--kcp--l2e20hggq3.yaml  \
  --bundle-name=bundle-shard-beta \
  --bundle-namespace=kcp-vespucci \
  --reconcile-interval=30s
```

## Required Flags

- `--bundle-name`: Name of the bundle Secret to read and apply (required)
- `--bundle-namespace`: Namespace where the bundle Secret exists (required)

## Optional Flags

- `--kubeconfig`: Path to source cluster kubeconfig. If not specified, uses `KUBECONFIG` environment variable or in-cluster config
- `--target-kubeconfig-path`: Path to the target cluster kubeconfig where resources will be applied. If not specified, uses the same cluster as source (useful for applying bundles from Secrets in the same cluster)
- `--reconcile-interval`: Time between reconciliation loops (default: 30s)
- `--create-namespace`: Create namespace on target cluster if it doesn't exist (default: true)
- `--dry-run`: Run in dry-run mode - logs what would be applied without actually applying (default: false)
- `-o` or `--output`: Output format for dry-run mode. Options: `yaml`, `json` (default: `yaml`)
- `-v`: Log verbosity level (higher = more verbose)

## How It Works

1. **Bundle Reading**: The tool reads the specified Secret from the source cluster. Bundle Secrets contain Kubernetes objects encoded as JSON in the Secret's data fields.

2. **Namespace Creation**: If `--create-namespace` is enabled (default), the tool ensures the bundle's namespace exists on the target cluster.

3. **Object Application**: For each object in the bundle:
   - Parses the JSON data into a Kubernetes object
   - Checks if the object exists on the target cluster
   - Creates the object if it doesn't exist
   - Updates the object if it already exists

4. **Continuous Reconciliation**: The process repeats at the specified `--reconcile-interval`, ensuring the target cluster stays in sync with the bundle contents.

## Example: Applying a RootShard Bundle

```bash
# Apply a rootshard bundle to a target cluster
./bundle-applier \
  --kubeconfig=~/.kube/source-cluster \
  --target-kubeconfig-path=~/.kube/target-cluster \
  --bundle-name=bundle-rootshard-root \
  --bundle-namespace=kcp-vespucci \
  --reconcile-interval=1m \
  -v=2
```

## Example: Dry Run

To see what would be applied without making changes:

```bash
./bundle-applier \
  --kubeconfig=~/.kube/source-cluster \
  --target-kubeconfig-path=~/.kube/target-cluster \
  --bundle-name=bundle-shard-alpha \
  --bundle-namespace=kcp-vespucci \
  --dry-run \
  -v=3
```

## Example: Same-Cluster Bundle Application

When you have a bundle Secret copied to the target cluster (e.g., exported from cluster 1 and applied to cluster 2), you can apply it without specifying a separate target kubeconfig:

```bash
# On cluster 2, after applying the exported bundle secret:
# kubectl apply -f alpha-bundle.yaml

# Apply the bundle to the same cluster where the secret exists
./bundle-applier \
  --kubeconfig=~/.kube/cluster2 \
  --bundle-name=alpha-bundle \
  --bundle-namespace=kcp-zheng \
  -v=2
```

This is useful for workflows where:
1. Cluster 1 generates bundle Secrets (e.g., via kcp-operator)
2. The bundle Secret is exported and copied to cluster 2
3. The bundler runs on cluster 2 to apply the resources from the Secret

```bash
# Export from cluster 1
kubectl --kubeconfig=cluster1.kubeconfig get secret -n kcp-zheng alpha-bundle -o yaml > alpha-bundle.yaml

# Apply to cluster 2
kubectl --kubeconfig=cluster2.kubeconfig apply -f alpha-bundle.yaml

# Run bundler on cluster 2 (same cluster mode)
./bundle-applier \
  --kubeconfig=cluster2.kubeconfig \
  --bundle-name=alpha-bundle \
  --bundle-namespace=kcp-zheng
```

## Environment Variables

- `KUBECONFIG`: Used as source kubeconfig if `--kubeconfig` is not specified

## Error Handling

- If the bundle Secret is not found, a warning is logged and the tool continues
- If individual objects fail to apply, errors are logged but the tool continues processing other objects
- The tool will retry on the next reconciliation interval

## Logging

Use `-v` flag to control log verbosity:
- `-v=0`: Errors and important info only
- `-v=1`: Standard info messages
- `-v=2`: Detailed reconciliation info
- `-v=3`: Verbose debugging including all objects being applied

## Use Cases

1. **Cluster Migration**: Copy resources from one cluster to another
2. **DR/Backup**: Continuously sync critical resources to a backup cluster
3. **Multi-Cluster Deployment**: Deploy the same bundle to multiple target clusters
4. **Development/Testing**: Apply production bundles to dev/test environments

## Bundle Secret Format

Bundle Secrets created by the kcp-operator have this structure:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: bundle-rootshard-root
  namespace: kcp-vespucci
data:
  # Each key is a GVR-style path, value is JSON-encoded Kubernetes object
  apis_apps_v1_namespaces_kcp_vespucci_deployments_root_kcp: <base64-json>
  api_v1_namespaces_kcp_vespucci_services_root_kcp: <base64-json>
  # ... more objects
```

The tool automatically decodes and applies these objects to the target cluster.
