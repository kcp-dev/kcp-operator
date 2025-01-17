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

package v1alpha1

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"

	v1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	scheme "github.com/kcp-dev/kcp-operator/sdk/clientset/versioned/scheme"
)

// CacheServersGetter has a method to return a CacheServerInterface.
// A group's client should implement this interface.
type CacheServersGetter interface {
	CacheServers(namespace string) CacheServerInterface
}

// CacheServerInterface has methods to work with CacheServer resources.
type CacheServerInterface interface {
	Create(ctx context.Context, cacheServer *v1alpha1.CacheServer, opts v1.CreateOptions) (*v1alpha1.CacheServer, error)
	Update(ctx context.Context, cacheServer *v1alpha1.CacheServer, opts v1.UpdateOptions) (*v1alpha1.CacheServer, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, cacheServer *v1alpha1.CacheServer, opts v1.UpdateOptions) (*v1alpha1.CacheServer, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.CacheServer, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.CacheServerList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CacheServer, err error)
	CacheServerExpansion
}

// cacheServers implements CacheServerInterface
type cacheServers struct {
	*gentype.ClientWithList[*v1alpha1.CacheServer, *v1alpha1.CacheServerList]
}

// newCacheServers returns a CacheServers
func newCacheServers(c *OperatorV1alpha1Client, namespace string) *cacheServers {
	return &cacheServers{
		gentype.NewClientWithList[*v1alpha1.CacheServer, *v1alpha1.CacheServerList](
			"cacheservers",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1alpha1.CacheServer { return &v1alpha1.CacheServer{} },
			func() *v1alpha1.CacheServerList { return &v1alpha1.CacheServerList{} }),
	}
}
