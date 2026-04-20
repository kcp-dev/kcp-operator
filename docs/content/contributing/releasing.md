---
description: >
    The release process for cutting new minor and patch releases.
---

# Release Process

This document describes the end-to-end process for publishing a new kcp-operator release.

## Prerequisites

- Push access to the `kcp-dev/kcp-operator` repository.
- All desired changes merged (and cherry-picked for patch releases).

## 1. Bump the Default KCP Image Tag

Update the compiled-in default image tag in `internal/resources/resources.go`:

```go
const (
    ImageRepository = "ghcr.io/kcp-dev/kcp"
    ImageTag        = "v0.XX.Y" // <-- update this
)
```

Commit and merge the change before tagging.

## 2. Tag the Release

### Minor release (v0.X.0)

```bash
git checkout main
git tag -m "version 0.X" v0.X.0
git tag -m "SDK version 0.X" sdk/v0.X.0
git push upstream v0.X.0 sdk/v0.X.0

# Create the release branch
git checkout -B release-0.X
git push -u upstream release-0.X
```

### Patch release (v0.X.Y)

```bash
git checkout release-0.X
git tag -m "version 0.X.Y" v0.X.Y
git tag -m "SDK version 0.X.Y" sdk/v0.X.Y
git push upstream v0.X.Y sdk/v0.X.Y
```

## 3. Publish Documentation

1. Go to the [docs-gen-and-push workflow](https://github.com/kcp-dev/kcp-operator/actions/workflows/docs-gen-and-push.yaml).
2. Run the workflow manually on the release branch.
3. Verify the new version appears on [docs.kcp.io/kcp-operator](https://docs.kcp.io/kcp-operator/).
