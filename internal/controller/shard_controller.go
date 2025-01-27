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

package controller

import (
	"context"
	"errors"
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/shard"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// ShardReconciler reconciles a Shard object
type ShardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Shard{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *ShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling Shard object")

	var s operatorv1alpha1.Shard
	if err := r.Client.Get(ctx, req.NamespacedName, &s); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get shard: %w", err)
		}

		return ctrl.Result{}, nil
	}

	defer func() {
		if err := r.reconcileStatus(ctx, &s); err != nil {
			recErr = kerrors.NewAggregate([]error{recErr, err})
		}
	}()

	var rootShard operatorv1alpha1.RootShard
	if ref := s.Spec.RootShard.Reference; ref != nil {
		rootShardRef := types.NamespacedName{
			Namespace: s.Namespace,
			Name:      ref.Name,
		}

		if err := r.Client.Get(ctx, rootShardRef, &rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get root shard: %w", err)
		}
	} else {
		return ctrl.Result{}, errors.New("no RootShard reference specified in Shard spec")
	}

	return ctrl.Result{}, r.reconcile(ctx, &s, &rootShard)
}

func (r *ShardReconciler) reconcile(ctx context.Context, s *operatorv1alpha1.Shard, rootShard *operatorv1alpha1.RootShard) error {
	var errs []error

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(s, operatorv1alpha1.SchemeGroupVersion.WithKind("Shard")))

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		shard.ServerCertificateReconciler(s, rootShard),
		shard.ServiceAccountCertificateReconciler(s, rootShard),
		shard.VirtualWorkspacesCertificateReconciler(s, rootShard),
		shard.RootShardClientCertificateReconciler(s, rootShard),
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, s.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
		shard.RootShardClientKubeconfigReconciler(s, rootShard),
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

	return kerrors.NewAggregate(errs)
}

// reconcileStatus sets both phase and conditions on the reconciled Shard object.
func (r *ShardReconciler) reconcileStatus(ctx context.Context, oldShard *operatorv1alpha1.Shard) error {
	newShard := oldShard.DeepCopy()
	var errs []error

	if newShard.Status.Phase == "" {
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseProvisioning
	}

	if newShard.DeletionTimestamp != nil {
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseDeleting
	}

	if err := r.setAvailableCondition(ctx, newShard); err != nil {
		errs = append(errs, err)
	}

	if cond := apimeta.FindStatusCondition(newShard.Status.Conditions, string(operatorv1alpha1.ShardConditionTypeAvailable)); cond.Status == metav1.ConditionTrue {
		newShard.Status.Phase = operatorv1alpha1.ShardPhaseRunning
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldShard.Status, newShard.Status) {
		if err := r.Client.Status().Patch(ctx, newShard, client.MergeFrom(oldShard)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}

func (r *ShardReconciler) setAvailableCondition(ctx context.Context, s *operatorv1alpha1.Shard) error {
	var dep appsv1.Deployment
	depKey := types.NamespacedName{Namespace: s.Namespace, Name: resources.GetShardDeploymentName(s)}
	if err := r.Client.Get(ctx, depKey, &dep); client.IgnoreNotFound(err) != nil {
		return err
	}

	available := metav1.ConditionFalse
	reason := operatorv1alpha1.ShardConditionReasonDeploymentUnavailable
	msg := deploymentStatusString(dep, depKey)

	if dep.Name != "" {
		if deploymentReady(dep) {
			available = metav1.ConditionTrue
			reason = operatorv1alpha1.ShardConditionReasonReplicasUp
		} else {
			available = metav1.ConditionFalse
			reason = operatorv1alpha1.ShardConditionReasonReplicasUnavailable
		}
	}

	s.Status.Conditions = updateCondition(s.Status.Conditions, metav1.Condition{
		Type:               string(operatorv1alpha1.ShardConditionTypeAvailable),
		Status:             available,
		ObservedGeneration: s.Generation,
		Reason:             string(reason),
		Message:            msg,
	})

	return nil
}
