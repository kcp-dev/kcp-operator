/*
Copyright 2025 The kcp Authors.

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

package controller

import (
	"fmt"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	operatorclient "github.com/kcp-dev/kcp-operator/pkg/client"
	"github.com/kcp-dev/kcp-operator/pkg/controller/cacheserver"
	"github.com/kcp-dev/kcp-operator/pkg/controller/compiledcacheserver"
	"github.com/kcp-dev/kcp-operator/pkg/controller/compiledfrontproxy"
	"github.com/kcp-dev/kcp-operator/pkg/controller/compiledrootshard"
	"github.com/kcp-dev/kcp-operator/pkg/controller/compiledshard"
	"github.com/kcp-dev/kcp-operator/pkg/controller/compiledvirtualworkspace"
	"github.com/kcp-dev/kcp-operator/pkg/controller/frontproxy"
	"github.com/kcp-dev/kcp-operator/pkg/controller/kubeconfig"
	kubeconfigrbac "github.com/kcp-dev/kcp-operator/pkg/controller/kubeconfig-rbac"
	"github.com/kcp-dev/kcp-operator/pkg/controller/rootshard"
	"github.com/kcp-dev/kcp-operator/pkg/controller/shard"
	"github.com/kcp-dev/kcp-operator/pkg/controller/virtualworkspace"
)

// Options configures a controller group.
type Options struct {
	// Engage are multicluster options that are passed to each
	// SetupWithManager and applied for For, Owns and Watches.
	Engage  []mcbuilder.EngageOptions
	Address operatorclient.Addresser
}

// AddConfigControllers registers the controllers that configure a kcp instance: they own the
// cert-manager resources and compile the deploy.operator.kcp.io render inputs.
func AddConfigControllers(mgr mcmanager.Manager, options Options) error {
	if options.Address == nil {
		return fmt.Errorf("Options.Address is required")
	}
	if err := (&rootshard.RootShardReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "RootShard", err)
	}
	if err := (&frontproxy.FrontProxyReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "FrontProxy", err)
	}
	if err := (&shard.ShardReconciler{
		GetCluster: mgr.GetCluster,
		Address:    options.Address,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "Shard", err)
	}
	if err := (&cacheserver.CacheServerReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "CacheServer", err)
	}
	if err := (&kubeconfig.KubeconfigReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "Kubeconfig", err)
	}
	if err := (&virtualworkspace.Reconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "VirtualWorkspace", err)
	}
	if err := (&kubeconfigrbac.KubeconfigRBACReconciler{
		GetCluster: mgr.GetCluster,
		Address:    options.Address,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "KubeconfigRBAC", err)
	}
	return nil
}

// AddWorkloadControllers registers the controllers that turn compiled render inputs into
// workloads such as Deployments and Services.
func AddWorkloadControllers(mgr mcmanager.Manager, options Options) error {
	if err := (&compiledrootshard.CompiledRootShardReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "CompiledRootShard", err)
	}
	if err := (&compiledfrontproxy.CompiledFrontProxyReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "CompiledFrontProxy", err)
	}
	if err := (&compiledshard.CompiledShardReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "CompiledShard", err)
	}
	if err := (&compiledcacheserver.CompiledCacheServerReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "CompiledCacheServer", err)
	}
	if err := (&compiledvirtualworkspace.CompiledVirtualWorkspaceReconciler{
		GetCluster: mgr.GetCluster,
	}).SetupWithManager(mgr, options.Engage...); err != nil {
		return fmt.Errorf("unable to create controller %s: %w", "CompiledVirtualWorkspace", err)
	}
	// +kubebuilder:scaffold:builder
	return nil
}
