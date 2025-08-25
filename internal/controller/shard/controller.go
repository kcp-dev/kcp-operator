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

package shard

import (
	"context"
	"fmt"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/shard"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// ShardReconciler reconciles a Shard object
type ShardReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	rootShardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		rootShard := obj.(*operatorv1alpha1.RootShard)

		var shards operatorv1alpha1.ShardList
		if err := mgr.GetClient().List(ctx, &shards, &ctrlruntimeclient.ListOptions{Namespace: rootShard.Namespace}); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		for _, shard := range shards.Items {
			if ref := shard.Spec.RootShard.Reference; ref != nil && ref.Name == rootShard.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&shard)})
			}
		}

		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Shard{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&certmanagerv1.Certificate{}).
		Watches(&operatorv1alpha1.RootShard{}, rootShardHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;services,verbs=get;list;watch;create;update;patch;delete

func (r *ShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling Shard object")

	var s operatorv1alpha1.Shard
	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get shard: %w", err)
		}

		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &s)

	if err := r.reconcileStatus(ctx, &s, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

func (r *ShardReconciler) reconcile(ctx context.Context, s *operatorv1alpha1.Shard) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	cond, rootShard := util.FetchRootShard(ctx, r.Client, s.Namespace, s.Spec.RootShard.Reference)
	conditions = append(conditions, cond)

	if rootShard == nil {
		return conditions, nil
	}

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(s, operatorv1alpha1.SchemeGroupVersion.WithKind("Shard")))

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		shard.ServerCertificateReconciler(s, rootShard),
		shard.ServiceAccountCertificateReconciler(s, rootShard),
		shard.VirtualWorkspacesCertificateReconciler(s, rootShard),
		shard.RootShardClientCertificateReconciler(s, rootShard),
		shard.LogicalClusterAdminCertificateReconciler(s, rootShard),
		shard.ExternalLogicalClusterAdminCertificateReconciler(s, rootShard),
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
		shard.RootShardClientKubeconfigReconciler(s, rootShard),
		shard.LogicalClusterAdminKubeconfigReconciler(s, rootShard),
		shard.ExternalLogicalClusterAdminKubeconfigReconciler(s, rootShard),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		shard.DeploymentReconciler(s, rootShard),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		shard.ServiceReconciler(s),
	}, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return conditions, kerrors.NewAggregate(errs)
}

// reconcileStatus sets both phase and conditions on the reconciled Shard object.
func (r *ShardReconciler) reconcileStatus(ctx context.Context, oldShard *operatorv1alpha1.Shard, conditions []metav1.Condition) error {
	newShard := oldShard.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: newShard.Namespace, Name: resources.GetShardDeploymentName(newShard)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		errs = append(errs, err)
	} else {
		conditions = append(conditions, cond)
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = newShard.Generation
		newShard.Status.Conditions = util.UpdateCondition(newShard.Status.Conditions, condition)
	}

	availableCond := apimeta.FindStatusCondition(newShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))
	switch {
	case availableCond.Status == metav1.ConditionTrue:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseRunning

	case newShard.DeletionTimestamp != nil:
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseDeleting

	case newShard.Status.Phase == "":
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseProvisioning
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldShard.Status, newShard.Status) {
		if err := r.Client.Status().Patch(ctx, newShard, ctrlruntimeclient.MergeFrom(oldShard)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
