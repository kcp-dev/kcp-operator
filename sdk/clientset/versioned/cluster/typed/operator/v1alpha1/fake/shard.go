//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by kcp code-generator. DO NOT EDIT.

package fake

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kcp-dev/logicalcluster/v3"

	kcptesting "github.com/kcp-dev/client-go/third_party/k8s.io/client-go/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	applyconfigurationsoperatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/applyconfiguration/operator/v1alpha1"
	kcpoperatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/clientset/versioned/cluster/typed/operator/v1alpha1"
	operatorv1alpha1client "github.com/kcp-dev/kcp-operator/sdk/clientset/versioned/typed/operator/v1alpha1"
)

var shardsResource = schema.GroupVersionResource{Group: "operator.kcp.io", Version: "v1alpha1", Resource: "shards"}
var shardsKind = schema.GroupVersionKind{Group: "operator.kcp.io", Version: "v1alpha1", Kind: "Shard"}

type shardsClusterClient struct {
	*kcptesting.Fake
}

// Cluster scopes the client down to a particular cluster.
func (c *shardsClusterClient) Cluster(clusterPath logicalcluster.Path) kcpoperatorv1alpha1.ShardsNamespacer {
	if clusterPath == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return &shardsNamespacer{Fake: c.Fake, ClusterPath: clusterPath}
}

// List takes label and field selectors, and returns the list of Shards that match those selectors across all clusters.
func (c *shardsClusterClient) List(ctx context.Context, opts metav1.ListOptions) (*operatorv1alpha1.ShardList, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewListAction(shardsResource, shardsKind, logicalcluster.Wildcard, metav1.NamespaceAll, opts), &operatorv1alpha1.ShardList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &operatorv1alpha1.ShardList{ListMeta: obj.(*operatorv1alpha1.ShardList).ListMeta}
	for _, item := range obj.(*operatorv1alpha1.ShardList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested Shards across all clusters.
func (c *shardsClusterClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(kcptesting.NewWatchAction(shardsResource, logicalcluster.Wildcard, metav1.NamespaceAll, opts))
}

type shardsNamespacer struct {
	*kcptesting.Fake
	ClusterPath logicalcluster.Path
}

func (n *shardsNamespacer) Namespace(namespace string) operatorv1alpha1client.ShardInterface {
	return &shardsClient{Fake: n.Fake, ClusterPath: n.ClusterPath, Namespace: namespace}
}

type shardsClient struct {
	*kcptesting.Fake
	ClusterPath logicalcluster.Path
	Namespace   string
}

func (c *shardsClient) Create(ctx context.Context, shard *operatorv1alpha1.Shard, opts metav1.CreateOptions) (*operatorv1alpha1.Shard, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewCreateAction(shardsResource, c.ClusterPath, c.Namespace, shard), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}

func (c *shardsClient) Update(ctx context.Context, shard *operatorv1alpha1.Shard, opts metav1.UpdateOptions) (*operatorv1alpha1.Shard, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewUpdateAction(shardsResource, c.ClusterPath, c.Namespace, shard), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}

func (c *shardsClient) UpdateStatus(ctx context.Context, shard *operatorv1alpha1.Shard, opts metav1.UpdateOptions) (*operatorv1alpha1.Shard, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewUpdateSubresourceAction(shardsResource, c.ClusterPath, "status", c.Namespace, shard), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}

func (c *shardsClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.Invokes(kcptesting.NewDeleteActionWithOptions(shardsResource, c.ClusterPath, c.Namespace, name, opts), &operatorv1alpha1.Shard{})
	return err
}

func (c *shardsClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := kcptesting.NewDeleteCollectionAction(shardsResource, c.ClusterPath, c.Namespace, listOpts)

	_, err := c.Fake.Invokes(action, &operatorv1alpha1.ShardList{})
	return err
}

func (c *shardsClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*operatorv1alpha1.Shard, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewGetAction(shardsResource, c.ClusterPath, c.Namespace, name), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}

// List takes label and field selectors, and returns the list of Shards that match those selectors.
func (c *shardsClient) List(ctx context.Context, opts metav1.ListOptions) (*operatorv1alpha1.ShardList, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewListAction(shardsResource, shardsKind, c.ClusterPath, c.Namespace, opts), &operatorv1alpha1.ShardList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &operatorv1alpha1.ShardList{ListMeta: obj.(*operatorv1alpha1.ShardList).ListMeta}
	for _, item := range obj.(*operatorv1alpha1.ShardList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

func (c *shardsClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(kcptesting.NewWatchAction(shardsResource, c.ClusterPath, c.Namespace, opts))
}

func (c *shardsClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*operatorv1alpha1.Shard, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewPatchSubresourceAction(shardsResource, c.ClusterPath, c.Namespace, name, pt, data, subresources...), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}

func (c *shardsClient) Apply(ctx context.Context, applyConfiguration *applyconfigurationsoperatorv1alpha1.ShardApplyConfiguration, opts metav1.ApplyOptions) (*operatorv1alpha1.Shard, error) {
	if applyConfiguration == nil {
		return nil, fmt.Errorf("applyConfiguration provided to Apply must not be nil")
	}
	data, err := json.Marshal(applyConfiguration)
	if err != nil {
		return nil, err
	}
	name := applyConfiguration.Name
	if name == nil {
		return nil, fmt.Errorf("applyConfiguration.Name must be provided to Apply")
	}
	obj, err := c.Fake.Invokes(kcptesting.NewPatchSubresourceAction(shardsResource, c.ClusterPath, c.Namespace, *name, types.ApplyPatchType, data), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}

func (c *shardsClient) ApplyStatus(ctx context.Context, applyConfiguration *applyconfigurationsoperatorv1alpha1.ShardApplyConfiguration, opts metav1.ApplyOptions) (*operatorv1alpha1.Shard, error) {
	if applyConfiguration == nil {
		return nil, fmt.Errorf("applyConfiguration provided to Apply must not be nil")
	}
	data, err := json.Marshal(applyConfiguration)
	if err != nil {
		return nil, err
	}
	name := applyConfiguration.Name
	if name == nil {
		return nil, fmt.Errorf("applyConfiguration.Name must be provided to Apply")
	}
	obj, err := c.Fake.Invokes(kcptesting.NewPatchSubresourceAction(shardsResource, c.ClusterPath, c.Namespace, *name, types.ApplyPatchType, data, "status"), &operatorv1alpha1.Shard{})
	if obj == nil {
		return nil, err
	}
	return obj.(*operatorv1alpha1.Shard), err
}
