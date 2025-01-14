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
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/api/v1alpha1"
	operatorkcpiov1alpha1 "github.com/kcp-dev/kcp-operator/api/v1alpha1"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources/rootshard"
)

// RootShardReconciler reconciles a RootShard object
type RootShardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorkcpiov1alpha1.RootShard{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RootShard object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *RootShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling RootShard object")

	var rootShard v1alpha1.RootShard
	if err := r.Client.Get(ctx, req.NamespacedName, &rootShard); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to find %s/%s: %w", req.Namespace, req.Name, err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	defer func() {
		if err := r.reconcileStatus(ctx, &rootShard); err != nil {
			recErr = kerrors.NewAggregate([]error{recErr, err})
		}
	}()

	return ctrl.Result{}, r.reconcile(ctx, &rootShard)
}

func (r *RootShardReconciler) reconcile(ctx context.Context, rootShard *operatorkcpiov1alpha1.RootShard) error {
	var errs []error

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(rootShard, operatorkcpiov1alpha1.GroupVersion.WithKind("RootShard")))

	// Intermediate CAs that we need to generate a certificate and an issuer for.
	intermediateCAs := []v1alpha1.CA{
		v1alpha1.ServerCA,
		v1alpha1.RequestHeaderClientCA,
		v1alpha1.ClientCA,
		v1alpha1.ServiceAccountCA,
	}

	issuerReconcilers := []reconciling.NamedIssuerReconcilerFactory{
		rootshard.RootCAIssuerReconciler(rootShard),
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		rootshard.ServerCertificateReconciler(rootShard),
		rootshard.ServiceAccountCertificateReconciler(rootShard),
		rootshard.VirtualWorkspacesCertificateReconciler(rootShard),
	}

	for _, ca := range intermediateCAs {
		certReconcilers = append(certReconcilers, rootshard.CACertificateReconciler(rootShard, ca))
		issuerReconcilers = append(issuerReconcilers, rootshard.CAIssuerReconciler(rootShard, ca))
	}
	if rootShard.Spec.Certificates.IssuerRef != nil {
		certReconcilers = append(certReconcilers, rootshard.RootCACertificateReconciler(rootShard))
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := reconciling.ReconcileIssuers(ctx, issuerReconcilers, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		rootshard.DeploymentReconciler(rootShard),
	}, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		rootshard.ServiceReconciler(rootShard),
	}, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}

// reconcileStatus sets both phase and conditions on the reconciled RootShard object.
func (r *RootShardReconciler) reconcileStatus(ctx context.Context, oldRootShard *operatorkcpiov1alpha1.RootShard) error {
	rootShard := oldRootShard.DeepCopy()
	var errs []error

	if rootShard.Status.Phase == "" {
		rootShard.Status.Phase = operatorkcpiov1alpha1.RootShardPhaseProvisioning
	}

	if rootShard.DeletionTimestamp != nil {
		rootShard.Status.Phase = operatorkcpiov1alpha1.RootShardPhaseDeleting
	}

	if err := r.setAvailableCondition(ctx, rootShard); err != nil {
		errs = append(errs, err)
	}

	if cond := apimeta.FindStatusCondition(rootShard.Status.Conditions, string(operatorkcpiov1alpha1.RootShardConditionTypeAvailable)); cond.Status == metav1.ConditionTrue {
		rootShard.Status.Phase = operatorkcpiov1alpha1.RootShardPhaseRunning
	}

	if err := r.Client.Status().Patch(ctx, rootShard, client.MergeFrom(oldRootShard)); err != nil {
		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}

func (r *RootShardReconciler) setAvailableCondition(ctx context.Context, rootShard *operatorkcpiov1alpha1.RootShard) error {

	var dep appsv1.Deployment
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: rootShard.Namespace, Name: rootShard.GetDeploymentName()}, &dep); client.IgnoreNotFound(err) != nil {
		return err
	}

	available := metav1.ConditionFalse
	reason := operatorkcpiov1alpha1.RootShardConditionReasonDeploymentUnavailable
	msg := fmt.Sprintf("Deployment %s/%s does not exist yet", rootShard.Namespace, rootShard.GetDeploymentName())
	if dep.Name != "" {
		switch dep.Status.UpdatedReplicas == dep.Status.ReadyReplicas && dep.Status.ReadyReplicas == ptr.Deref(dep.Spec.Replicas, 0) {
		case true:
			available = metav1.ConditionTrue
			reason = operatorkcpiov1alpha1.RootShardConditionReasonReplicasUp
			msg = fmt.Sprintf("Deployment %s/%s is fully up and running", rootShard.Namespace, rootShard.GetDeploymentName())
		case false:
			available = metav1.ConditionFalse
			reason = operatorkcpiov1alpha1.RootShardConditionReasonReplicasUnavailable
			msg = fmt.Sprintf("Deployment %s/%s is not in desired replica state", rootShard.Namespace, rootShard.GetDeploymentName())
		}
	}

	if rootShard.Status.Conditions == nil {
		rootShard.Status.Conditions = make([]metav1.Condition, 0)
	}

	cond := apimeta.FindStatusCondition(rootShard.Status.Conditions, string(operatorkcpiov1alpha1.RootShardConditionTypeAvailable))

	if cond == nil || cond.ObservedGeneration != rootShard.Generation || cond.Status != available {
		apimeta.SetStatusCondition(&rootShard.Status.Conditions, metav1.Condition{
			Type:               string(operatorkcpiov1alpha1.RootShardConditionTypeAvailable),
			Status:             available,
			ObservedGeneration: rootShard.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(reason),
			Message:            msg,
		})
	}

	return nil
}
