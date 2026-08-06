/*
Copyright 2024 The kcp Authors.

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

package compiledcacheserver

import (
	"context"
	"errors"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/resources/compiledcacheserver"
	"github.com/kcp-dev/kcp-operator/pkg/controller/util"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling/modifier"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
)

// CompiledCacheServerReconciler reconciles a CompiledCacheServer object
type CompiledCacheServerReconciler struct {
	Client ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

func (r *CompiledCacheServerReconciler) SetupWithManager(mgr ctrlruntime.Manager) error {
	// The rendered Deployment mounts Secrets owned by the source object, not by this one,
	// so an ownership watch would never retry a Deployment blocked on a missing mount.
	mountHandler := util.EnqueueAllInNamespace(mgr.GetClient(), func() ctrlruntimeclient.ObjectList {
		return &deployv1alpha1.CompiledCacheServerList{}
	})

	return ctrlruntime.NewControllerManagedBy(mgr).
		Named("compiled-cache-server").
		For(&deployv1alpha1.CompiledCacheServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Watches(&corev1.Secret{}, mountHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledcacheservers,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledcacheservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets,verbs=get;list;watch

func (r *CompiledCacheServerReconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (ctrlruntime.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	server := &deployv1alpha1.CompiledCacheServer{}
	if err := r.Client.Get(ctx, req.NamespacedName, server); err != nil {
		return ctrlruntime.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if server.DeletionTimestamp != nil {
		return ctrlruntime.Result{}, nil
	}

	if err := r.reconcile(ctx, r.Client, server); err != nil {
		return ctrlruntime.Result{}, err
	}

	return ctrlruntime.Result{}, nil
}

func (r *CompiledCacheServerReconciler) reconcile(ctx context.Context, client ctrlruntimeclient.Client, server *deployv1alpha1.CompiledCacheServer) error {
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(server, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledCacheServer")))
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, client)

	// This will fail as long as some of the referenced Secrets/ConfigMaps do not exist yet. We rely on
	// requeueing to eventually get there in the end. Importantly, reconciling Deployments has to happen
	// after all Secrets have been reconciled.
	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		compiledcacheserver.DeploymentReconciler(server),
	}, server.Namespace, client, ownerRefWrapper, revisionLabels); err != nil {
		// Swallow these errors and instead rely on us watching Secrets and re-reconciling whenever they change.
		if errors.Is(err, modifier.ErrMountNotFound) {
			return nil
		}
		return err
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		compiledcacheserver.ServiceReconciler(server),
	}, server.Namespace, client, ownerRefWrapper); err != nil {
		return err
	}

	return nil
}
