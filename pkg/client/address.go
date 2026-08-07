/*
Copyright 2026 The kcp Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"net"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// Endpoint is how kcp components are reached.
type Endpoint struct {
	// URL is the base URL of the component.
	URL string

	// ServerName is the name verified during the TLS handshake.
	// Empty verifies the URL's host.
	ServerName string

	// DialContext connects to the address on the named network using
	// the provided context.
	// it has the same interfaces as [net.Dialer] DialContext.
	Dial func(ctx context.Context, network, address string) (net.Conn, error)
}

// Addresser resolves the components an operator has to talk to.
//
// By default kcp-operator addresses components through their in-cluster
// service addresses. In deployments across multiple clusters the user
// must supply how kcp-operator can reach the components.
type Addresser interface {
	// RootShard addresses a root shard.
	RootShard(rootShard *operatorv1alpha1.RootShard) Endpoint

	// RootShardProxy addresses the internal proxy deployed alongside the root shard.
	RootShardProxy(rootShard *operatorv1alpha1.RootShard) Endpoint

	// Shard addresses a shard.
	Shard(shard *operatorv1alpha1.Shard) Endpoint
}

// InCluster is the default implementation, targeting each component
// with the in-cluster service address.
type InCluster struct{}

var _ Addresser = InCluster{}

func (InCluster) RootShard(rootShard *operatorv1alpha1.RootShard) Endpoint {
	return Endpoint{
		URL: resources.GetRootShardBaseURL(rootShard),
	}
}

func (InCluster) RootShardProxy(rootShard *operatorv1alpha1.RootShard) Endpoint {
	return Endpoint{
		URL: resources.GetRootShardProxyBaseURL(rootShard),
	}
}

func (InCluster) Shard(shard *operatorv1alpha1.Shard) Endpoint {
	return Endpoint{
		URL: resources.GetShardBaseURL(shard),
	}
}
