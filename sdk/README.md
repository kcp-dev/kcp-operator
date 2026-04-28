# kcp Operator SDK

This directory contains the kcp operator's SDK: re-usable Go API types and generated functions for
integrating the kcp operator into 3rd-party applications.

## Usage

To install the SDK, simply `go get` it:

```bash
go get github.com/kcp-dev/kcp-operator/sdk@latest
```

and then in your code import the desired types:

```go
package main

import operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"

func createRootShard() *operatorv1alpha1.RootShard {
   rs := &operatorv1alpha1.RootShard{}
   rs.Name = "my-r00t"
   rs.Namespace = "default"

   return rs
}
```

## SDK Design

The SDK comes as a standalone Go module: `github.com/kcp-dev/kcp-operator/sdk`

The module reduces the transitive dependencies that consumers have to worry about when they want to
integrate the kcp operator. To that end, the SDK is meant to provide the broadest possible
compatibility: dependencies are on the *lowest* version that is usable by the kcp operator. This
drift between the operator's dependencies and those of the SDK is an intended feature of the SDK.

The actual dependency versions used in the kcp operator binaries are controlled exclusively via the
root directory's `go.mod`. Specifically, the SDK is not meant to propagate security fixes to
consumers and force them to upgrade when it might be inconvenient to them.

## Development Guidelines

* Do not update the `go` constraint in the `go.mod` file manually, let `go mod tidy` update it only
  when necessary. The `go` constraint has no influence on what Go version the operator is actually
  built with. It can, however, cause serious annoyances for downstream consumers.
* Likewise, only bump dependencies to keep the SDK compatible with the main module.
