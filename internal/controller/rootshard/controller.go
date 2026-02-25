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

package rootshard

import (
	"context"
	"fmt"
	"sort"
	"time"

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

	bundlehelper "github.com/kcp-dev/kcp-operator/internal/controller/bundle"
	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/metrics"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/frontproxy"
	"github.com/kcp-dev/kcp-operator/internal/resources/rootshard"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// RootShardReconciler reconciles a RootShard object
type RootShardReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	shardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		shard := obj.(*operatorv1alpha1.Shard)

		var rootShard operatorv1alpha1.RootShard
		if err := mgr.GetClient().Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: shard.Namespace, Name: shard.Spec.RootShard.Reference.Name}, &rootShard); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&rootShard)})

		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("rootshard").
		For(&operatorv1alpha1.RootShard{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&certmanagerv1.Certificate{}).
		Watches(&operatorv1alpha1.Shard{}, shardHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;secrets,verbs=get;list;watch;create;update;patch;delete

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
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.RootShardResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var rootShard operatorv1alpha1.RootShard
	if err := r.Get(ctx, req.NamespacedName, &rootShard); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.RootShardResourceType, err.Error())
			return ctrl.Result{}, fmt.Errorf("failed to find %s/%s: %w", req.Namespace, req.Name, err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &rootShard)

	if err := r.reconcileStatus(ctx, &rootShard, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

//nolint:unparam // Keep the controller working the same as all the others, even though currently it does always return nil conditions.
func (r *RootShardReconciler) reconcile(ctx context.Context, rootShard *operatorv1alpha1.RootShard) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	if rootShard.DeletionTimestamp != nil {
		return conditions, nil
	}

	// Ensure Bundle object exists if annotation is present
	if _, err := bundlehelper.EnsureBundleForOwner(ctx, r.Client, r.Scheme, rootShard); err != nil {
		errs = append(errs, fmt.Errorf("failed to ensure bundle: %w", err))
	}

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(rootShard, operatorv1alpha1.SchemeGroupVersion.WithKind("RootShard")))

	issuerReconcilers := []reconciling.NamedIssuerReconcilerFactory{
		rootshard.RootCAIssuerReconciler(rootShard),
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		rootshard.ServerCertificateReconciler(rootShard),
		rootshard.ServiceAccountCertificateReconciler(rootShard),
		rootshard.VirtualWorkspacesCertificateReconciler(rootShard),
		rootshard.LogicalClusterAdminCertificateReconciler(rootShard),
		rootshard.ExternalLogicalClusterAdminCertificateReconciler(rootShard),
		rootshard.OperatorClientCertificateReconciler(rootShard),
	}

	// Intermediate CAs that we need to generate a certificate and an issuer for.
	intermediateCAs := []operatorv1alpha1.CA{
		operatorv1alpha1.ServerCA,
		operatorv1alpha1.RequestHeaderClientCA,
		operatorv1alpha1.ClientCA,
		operatorv1alpha1.ServiceAccountCA,
		operatorv1alpha1.FrontProxyClientCA,
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

	if rootShard.Spec.CABundleSecretRef != nil {
		if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
			rootshard.MergedCABundleSecretReconciler(ctx, rootShard, r.Client),
		}, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
		rootshard.LogicalClusterAdminKubeconfigReconciler(rootShard),
		rootshard.ExternalLogicalClusterAdminKubeconfigReconciler(rootShard),
	}, rootShard.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	// Deployment will be scaled to 0 if bundle annotation is present
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

	if err := frontproxy.NewRootShardProxy(rootShard).Reconcile(ctx, r.Client, rootShard.Namespace); err != nil {
		errs = append(errs, fmt.Errorf("failed to reconcile proxy: %w", err))
	}

	return conditions, kerrors.NewAggregate(errs)
}

// reconcileStatus sets both phase and conditions on the reconciled RootShard object.
func (r *RootShardReconciler) reconcileStatus(ctx context.Context, oldRootShard *operatorv1alpha1.RootShard, conditions []metav1.Condition) error {
	rootShard := oldRootShard.DeepCopy()
	var errs []error

	// Add Bundle condition
	bundleCond := bundlehelper.GetBundleReadyCondition(ctx, r.Client, rootShard, rootShard.Generation)
	conditions = append(conditions, bundleCond)

	// Check if rootshard is bundled (has bundle annotation with Ready bundle)
	isBundled := bundleCond.Status == metav1.ConditionTrue && bundleCond.Reason == "BundleReady"

	// Only check deployment status if not bundled
	if !isBundled {
		depKey := types.NamespacedName{Namespace: rootShard.Namespace, Name: resources.GetRootShardDeploymentName(rootShard)}
		cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
		if err != nil {
			errs = append(errs, err)
		} else {
			conditions = append(conditions, cond)
		}
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = rootShard.Generation
		rootShard.Status.Conditions = util.UpdateCondition(rootShard.Status.Conditions, condition)
	}

	if rootShard.DeletionTimestamp != nil {
		rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseDeleting
	} else {
		availableCond := apimeta.FindStatusCondition(rootShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))
		bundleStatusCond := apimeta.FindStatusCondition(rootShard.Status.Conditions, string(operatorv1alpha1.ConditionTypeBundle))

		switch {
		case isBundled:
			// RootShard is bundled, deployment scaled to 0, resources exported via bundle
			rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseBundled

		case bundleStatusCond != nil && bundleStatusCond.Status != metav1.ConditionTrue:
			rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseProvisioning

		case availableCond != nil && availableCond.Status == metav1.ConditionTrue:
			rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseRunning

		default:
			rootShard.Status.Phase = operatorv1alpha1.RootShardPhaseProvisioning
		}
	}

	shards, err := util.GetRootShardChildren(ctx, r.Client, rootShard)
	if err != nil {
		errs = append(errs, err)
	} else {
		rootShard.Status.Shards = make([]operatorv1alpha1.ShardReference, len(shards))
		for i, shard := range shards {
			rootShard.Status.Shards[i] = operatorv1alpha1.ShardReference{Name: shard.Name}
		}
	}
	// sort the shards by name for equality comparison
	sort.Slice(rootShard.Status.Shards, func(i, j int) bool {
		return rootShard.Status.Shards[i].Name < rootShard.Status.Shards[j].Name
	})

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldRootShard.Status, rootShard.Status) {
		if err := r.Status().Patch(ctx, rootShard, ctrlruntimeclient.MergeFrom(oldRootShard)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
