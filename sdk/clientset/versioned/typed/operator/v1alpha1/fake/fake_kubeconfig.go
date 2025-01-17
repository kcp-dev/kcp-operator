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
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	v1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// FakeKubeconfigs implements KubeconfigInterface
type FakeKubeconfigs struct {
	Fake *FakeOperatorV1alpha1
	ns   string
}

var kubeconfigsResource = v1alpha1.SchemeGroupVersion.WithResource("kubeconfigs")

var kubeconfigsKind = v1alpha1.SchemeGroupVersion.WithKind("Kubeconfig")

// Get takes name of the kubeconfig, and returns the corresponding kubeconfig object, and an error if there is any.
func (c *FakeKubeconfigs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Kubeconfig, err error) {
	emptyResult := &v1alpha1.Kubeconfig{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(kubeconfigsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Kubeconfig), err
}

// List takes label and field selectors, and returns the list of Kubeconfigs that match those selectors.
func (c *FakeKubeconfigs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.KubeconfigList, err error) {
	emptyResult := &v1alpha1.KubeconfigList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(kubeconfigsResource, kubeconfigsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.KubeconfigList{ListMeta: obj.(*v1alpha1.KubeconfigList).ListMeta}
	for _, item := range obj.(*v1alpha1.KubeconfigList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested kubeconfigs.
func (c *FakeKubeconfigs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(kubeconfigsResource, c.ns, opts))

}

// Create takes the representation of a kubeconfig and creates it.  Returns the server's representation of the kubeconfig, and an error, if there is any.
func (c *FakeKubeconfigs) Create(ctx context.Context, kubeconfig *v1alpha1.Kubeconfig, opts v1.CreateOptions) (result *v1alpha1.Kubeconfig, err error) {
	emptyResult := &v1alpha1.Kubeconfig{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(kubeconfigsResource, c.ns, kubeconfig, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Kubeconfig), err
}

// Update takes the representation of a kubeconfig and updates it. Returns the server's representation of the kubeconfig, and an error, if there is any.
func (c *FakeKubeconfigs) Update(ctx context.Context, kubeconfig *v1alpha1.Kubeconfig, opts v1.UpdateOptions) (result *v1alpha1.Kubeconfig, err error) {
	emptyResult := &v1alpha1.Kubeconfig{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(kubeconfigsResource, c.ns, kubeconfig, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Kubeconfig), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeKubeconfigs) UpdateStatus(ctx context.Context, kubeconfig *v1alpha1.Kubeconfig, opts v1.UpdateOptions) (result *v1alpha1.Kubeconfig, err error) {
	emptyResult := &v1alpha1.Kubeconfig{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(kubeconfigsResource, "status", c.ns, kubeconfig, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Kubeconfig), err
}

// Delete takes name of the kubeconfig and deletes it. Returns an error if one occurs.
func (c *FakeKubeconfigs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(kubeconfigsResource, c.ns, name, opts), &v1alpha1.Kubeconfig{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKubeconfigs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(kubeconfigsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.KubeconfigList{})
	return err
}

// Patch applies the patch and returns the patched kubeconfig.
func (c *FakeKubeconfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Kubeconfig, err error) {
	emptyResult := &v1alpha1.Kubeconfig{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(kubeconfigsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Kubeconfig), err
}
