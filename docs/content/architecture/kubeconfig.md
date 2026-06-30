---
description: >
    Shows how `Kubeconfig` objects can be used to provide credentials to kcp.
---

# Kubeconfigs

Besides provisioning kcp itself, the kcp-operator can also provide [kubeconfigs](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/) to access kcp. Each kubeconfig will internally be backed by a dedicated client certificate.

## Basics

A minimal `Kubeconfig` object typically looks like this:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: susan
  namespace: my-kcp
spec:
  # Required: the username inside Kubernetes;
  # this will be the client certificate's common name.
  username: susan

  # required: groups to attach to the user;
  # this will be the organizations in the client cert.
  groups:
    - system:kcp:admin

  # Required: in what Secret the generated kubeconfig should be stored.
  secretRef:
    name: susan-kubeconfig

  # Required: a Kubeconfig must target either a FrontProxy, Shard or RootShard.
  target:
    frontProxyRef:
      name: my-front-proxy

  # Optional: specifies the workspace path this kubeconfig targets.
  # Used in the generated kubeconfig server URL and as the default RBAC
  # provisioning target. Defaults to "root" if unset.
  targetWorkspace: root:orga:teamb

  # Required: how long the certificate should be valid for;
  # the operator will automatically renew the certificate, after which the
  # Secret will be renewed and have to be re-downloaded.
  validity: 8766h
```

`Kubeconfig` objects must exist in the same namespace as the kcp installation they are meant for.

Once the `Kubeconfig` has been created, you can observe its status to wait for it to be ready. After that, retrieve the Secret mentioned in the `secretRef` to find the finished kubeconfig, ready to use.

!!! warning
    Deleting a `Kubeconfig` will also delete the underlying Secret from the hosting cluster, however this will not invalidate the existing certificate that is embedded in the kubeconfig. This means anyone with a copy of the kubeconfig can keep using it until the certificate expires.

    To disarm an old kubeconfig, make sure to revoke any permissions granted through RBAC for the user and/or their groups.

!!! note
    The `Kubeconfig`'s name is embedded into the certificate in form of a group (organization) named `kubeconfig:<name>`. This is to allow a unique mapping from RBAC rules to `Kubeconfig` objects for the authorization (see further down). Take note that this means the `Kubeconfig`' name is leaked to whoever gets the kubeconfig.

## Target Workspace

By default, the generated kubeconfig's server URL points to the `root` workspace. To target a different workspace, set `spec.targetWorkspace`:

```yaml
spec:
  targetWorkspace: root:orga:teamb
```

This affects two things:

1. **Server URL**: The generated kubeconfig will contain a server URL pointing to the specified workspace (e.g. `https://.../clusters/root:orga:teamb`).
2. **RBAC target**: When authorization is configured (see below), the RBAC rules will be provisioned in this workspace by default.

The field accepts kcp workspace paths like `root`, `root:org`, or `root:org:team`.

## Authorization

Without any further configuration than shown in the basics section above, the created identity (username + groups) will not get any permissions in kcp. So while the kubeconfig is valid and allows proper authentication, pretty much no actions will be permitted yet.

The administrator has to either rely on externally-managed RBAC rules to provide permissions, or use the kcp-operator to provision such RBAC in a workspace.

To make the kcp-operator manage RBAC, use `spec.authorization` inside a `Kubeconfig`:

```yaml
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: susan
  namespace: my-kcp
spec:
  #...snip...

  targetWorkspace: root:orga:teamb

  authorization:
    clusterRoleBindings:
      clusterRoles:
        - cluster-admin
```

This configuration would bind the group `kubeconfig:susan` to the ClusterRole `cluster-admin` inside the workspace specified in `spec.targetWorkspace`. Note that this is specifically not bound to the user (common name), so that two `Kubeconfig` objects that both have the same `spec.name` to not have colliding RBAC.

When deleting a `Kubeconfig` with authorization settings, the kcp-operator will first unprovision (delete) the `ClusterRoleBindings` before the `Kubeconfig` can be deleted.

!!! note "Deprecated: `authorization.clusterRoleBindings.cluster`"
    Previously, the target workspace for RBAC was specified via `spec.authorization.clusterRoleBindings.cluster`. This field is now deprecated in favor of `spec.targetWorkspace`. The two fields cannot be set together. Existing resources using the deprecated field will continue to work for RBAC provisioning, but note that the deprecated field does **not** influence the kubeconfig server URL (which always defaults to `root` unless `spec.targetWorkspace` is set).
