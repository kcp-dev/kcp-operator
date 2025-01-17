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
	"context"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/v2/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	scopedclientset "github.com/kcp-dev/kcp-operator/sdk/clientset/versioned"
	clientset "github.com/kcp-dev/kcp-operator/sdk/clientset/versioned/cluster"
	"github.com/kcp-dev/kcp-operator/sdk/informers/externalversions/internalinterfaces"
	operatorv1alpha1listers "github.com/kcp-dev/kcp-operator/sdk/listers/operator/v1alpha1"
)

// RootShardClusterInformer provides access to a shared informer and lister for
// RootShards.
type RootShardClusterInformer interface {
	Cluster(logicalcluster.Name) RootShardInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() operatorv1alpha1listers.RootShardClusterLister
}

type rootShardClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewRootShardClusterInformer constructs a new informer for RootShard type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRootShardClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredRootShardClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredRootShardClusterInformer constructs a new informer for RootShard type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRootShardClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatorV1alpha1().RootShards().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatorV1alpha1().RootShards().Watch(context.TODO(), options)
			},
		},
		&operatorv1alpha1.RootShard{},
		resyncPeriod,
		indexers,
	)
}

func (f *rootShardClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredRootShardClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
		kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f *rootShardClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&operatorv1alpha1.RootShard{}, f.defaultInformer)
}

func (f *rootShardClusterInformer) Lister() operatorv1alpha1listers.RootShardClusterLister {
	return operatorv1alpha1listers.NewRootShardClusterLister(f.Informer().GetIndexer())
}

// RootShardInformer provides access to a shared informer and lister for
// RootShards.
type RootShardInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() operatorv1alpha1listers.RootShardLister
}

func (f *rootShardClusterInformer) Cluster(clusterName logicalcluster.Name) RootShardInformer {
	return &rootShardInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type rootShardInformer struct {
	informer cache.SharedIndexInformer
	lister   operatorv1alpha1listers.RootShardLister
}

func (f *rootShardInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *rootShardInformer) Lister() operatorv1alpha1listers.RootShardLister {
	return f.lister
}

type rootShardScopedInformer struct {
	factory          internalinterfaces.SharedScopedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

func (f *rootShardScopedInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&operatorv1alpha1.RootShard{}, f.defaultInformer)
}

func (f *rootShardScopedInformer) Lister() operatorv1alpha1listers.RootShardLister {
	return operatorv1alpha1listers.NewRootShardLister(f.Informer().GetIndexer())
}

// NewRootShardInformer constructs a new informer for RootShard type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRootShardInformer(client scopedclientset.Interface, resyncPeriod time.Duration, namespace string, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRootShardInformer(client, resyncPeriod, namespace, indexers, nil)
}

// NewFilteredRootShardInformer constructs a new informer for RootShard type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRootShardInformer(client scopedclientset.Interface, resyncPeriod time.Duration, namespace string, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatorV1alpha1().RootShards(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatorV1alpha1().RootShards(namespace).Watch(context.TODO(), options)
			},
		},
		&operatorv1alpha1.RootShard{},
		resyncPeriod,
		indexers,
	)
}

func (f *rootShardScopedInformer) defaultInformer(client scopedclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRootShardInformer(client, resyncPeriod, f.namespace, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	}, f.tweakListOptions)
}
