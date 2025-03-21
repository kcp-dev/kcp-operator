/*
Copyright 2024 The KCP Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"

	v1alpha1 "github.com/kcp-dev/kcp-operator/sdk/clientset/versioned/typed/operator/v1alpha1"
)

type FakeOperatorV1alpha1 struct {
	*testing.Fake
}

func (c *FakeOperatorV1alpha1) CacheServers(namespace string) v1alpha1.CacheServerInterface {
	return &FakeCacheServers{c, namespace}
}

func (c *FakeOperatorV1alpha1) FrontProxies(namespace string) v1alpha1.FrontProxyInterface {
	return &FakeFrontProxies{c, namespace}
}

func (c *FakeOperatorV1alpha1) Kubeconfigs(namespace string) v1alpha1.KubeconfigInterface {
	return &FakeKubeconfigs{c, namespace}
}

func (c *FakeOperatorV1alpha1) RootShards(namespace string) v1alpha1.RootShardInterface {
	return &FakeRootShards{c, namespace}
}

func (c *FakeOperatorV1alpha1) Shards(namespace string) v1alpha1.ShardInterface {
	return &FakeShards{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeOperatorV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
