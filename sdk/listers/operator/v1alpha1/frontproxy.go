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

package v1alpha1

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// FrontProxyClusterLister can list FrontProxies across all workspaces, or scope down to a FrontProxyLister for one workspace.
// All objects returned here must be treated as read-only.
type FrontProxyClusterLister interface {
	// List lists all FrontProxies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error)
	// Cluster returns a lister that can list and get FrontProxies in one workspace.
	Cluster(clusterName logicalcluster.Name) FrontProxyLister
	FrontProxyClusterListerExpansion
}

type frontProxyClusterLister struct {
	indexer cache.Indexer
}

// NewFrontProxyClusterLister returns a new FrontProxyClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewFrontProxyClusterLister(indexer cache.Indexer) *frontProxyClusterLister {
	return &frontProxyClusterLister{indexer: indexer}
}

// List lists all FrontProxies in the indexer across all workspaces.
func (s *frontProxyClusterLister) List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*operatorv1alpha1.FrontProxy))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get FrontProxies.
func (s *frontProxyClusterLister) Cluster(clusterName logicalcluster.Name) FrontProxyLister {
	return &frontProxyLister{indexer: s.indexer, clusterName: clusterName}
}

// FrontProxyLister can list FrontProxies across all namespaces, or scope down to a FrontProxyNamespaceLister for one namespace.
// All objects returned here must be treated as read-only.
type FrontProxyLister interface {
	// List lists all FrontProxies in the workspace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error)
	// FrontProxies returns a lister that can list and get FrontProxies in one workspace and namespace.
	FrontProxies(namespace string) FrontProxyNamespaceLister
	FrontProxyListerExpansion
}

// frontProxyLister can list all FrontProxies inside a workspace or scope down to a FrontProxyLister for one namespace.
type frontProxyLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all FrontProxies in the indexer for a workspace.
func (s *frontProxyLister) List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*operatorv1alpha1.FrontProxy))
	})
	return ret, err
}

// FrontProxies returns an object that can list and get FrontProxies in one namespace.
func (s *frontProxyLister) FrontProxies(namespace string) FrontProxyNamespaceLister {
	return &frontProxyNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// frontProxyNamespaceLister helps list and get FrontProxies.
// All objects returned here must be treated as read-only.
type FrontProxyNamespaceLister interface {
	// List lists all FrontProxies in the workspace and namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error)
	// Get retrieves the FrontProxy from the indexer for a given workspace, namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*operatorv1alpha1.FrontProxy, error)
	FrontProxyNamespaceListerExpansion
}

// frontProxyNamespaceLister helps list and get FrontProxies.
// All objects returned here must be treated as read-only.
type frontProxyNamespaceLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
	namespace   string
}

// List lists all FrontProxies in the indexer for a given workspace and namespace.
func (s *frontProxyNamespaceLister) List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*operatorv1alpha1.FrontProxy))
	})
	return ret, err
}

// Get retrieves the FrontProxy from the indexer for a given workspace, namespace and name.
func (s *frontProxyNamespaceLister) Get(name string) (*operatorv1alpha1.FrontProxy, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(operatorv1alpha1.Resource("frontproxies"), name)
	}
	return obj.(*operatorv1alpha1.FrontProxy), nil
}

// NewFrontProxyLister returns a new FrontProxyLister.
// We assume that the indexer:
// - is fed by a workspace-scoped LIST+WATCH
// - uses cache.MetaNamespaceKeyFunc as the key function
// - has the cache.NamespaceIndex as an index
func NewFrontProxyLister(indexer cache.Indexer) *frontProxyScopedLister {
	return &frontProxyScopedLister{indexer: indexer}
}

// frontProxyScopedLister can list all FrontProxies inside a workspace or scope down to a FrontProxyLister for one namespace.
type frontProxyScopedLister struct {
	indexer cache.Indexer
}

// List lists all FrontProxies in the indexer for a workspace.
func (s *frontProxyScopedLister) List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error) {
	err = cache.ListAll(s.indexer, selector, func(i interface{}) {
		ret = append(ret, i.(*operatorv1alpha1.FrontProxy))
	})
	return ret, err
}

// FrontProxies returns an object that can list and get FrontProxies in one namespace.
func (s *frontProxyScopedLister) FrontProxies(namespace string) FrontProxyNamespaceLister {
	return &frontProxyScopedNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// frontProxyScopedNamespaceLister helps list and get FrontProxies.
type frontProxyScopedNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all FrontProxies in the indexer for a given workspace and namespace.
func (s *frontProxyScopedNamespaceLister) List(selector labels.Selector) (ret []*operatorv1alpha1.FrontProxy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*operatorv1alpha1.FrontProxy))
	})
	return ret, err
}

// Get retrieves the FrontProxy from the indexer for a given workspace, namespace and name.
func (s *frontProxyScopedNamespaceLister) Get(name string) (*operatorv1alpha1.FrontProxy, error) {
	key := s.namespace + "/" + name
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(operatorv1alpha1.Resource("frontproxies"), name)
	}
	return obj.(*operatorv1alpha1.FrontProxy), nil
}
