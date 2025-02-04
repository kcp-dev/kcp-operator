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

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/frontproxy"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// FrontProxyReconciler reconciles a FrontProxy object
type FrontProxyReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *FrontProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	rootShardHandler := handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		rootShard := obj.(*operatorv1alpha1.RootShard)

		var fpList operatorv1alpha1.FrontProxyList
		if err := mgr.GetClient().List(ctx, &fpList, &ctrlruntimeclient.ListOptions{Namespace: rootShard.Namespace}); err != nil {
			utilruntime.HandleError(err)
			return nil
		}

		var requests []reconcile.Request
		for _, frontProxy := range fpList.Items {
			if ref := frontProxy.Spec.RootShard.Reference; ref != nil && ref.Name == rootShard.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&frontProxy)})
			}
		}

		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.FrontProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&certmanagerv1.Certificate{}).
		Watches(&operatorv1alpha1.RootShard{}, rootShardHandler).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=frontproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kcp.io,resources=frontproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=frontproxies/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete

func (r *FrontProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var frontProxy operatorv1alpha1.FrontProxy
	if err := r.Client.Get(ctx, req.NamespacedName, &frontProxy); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get FrontProxy object: %w", err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &frontProxy)

	if err := r.reconcileStatus(ctx, &frontProxy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

func (r *FrontProxyReconciler) reconcile(ctx context.Context, frontProxy *operatorv1alpha1.FrontProxy) ([]metav1.Condition, error) {
	var (
		errs       []error
		conditions []metav1.Condition
	)

	cond, rootShard := fetchRootShard(ctx, r.Client, frontProxy.Namespace, frontProxy.Spec.RootShard.Reference)
	conditions = append(conditions, cond)

	if rootShard == nil {
		return conditions, nil
	}

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(frontProxy, operatorv1alpha1.SchemeGroupVersion.WithKind("FrontProxy")))

	configMapReconcilers := []k8creconciling.NamedConfigMapReconcilerFactory{
		frontproxy.PathMappingConfigMapReconciler(frontProxy, rootShard),
	}

	secretReconcilers := []k8creconciling.NamedSecretReconcilerFactory{
		frontproxy.DynamicKubeconfigSecretReconciler(frontProxy, rootShard),
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		frontproxy.ServerCertificateReconciler(frontProxy, rootShard),
		frontproxy.KubeconfigCertificateReconciler(frontProxy, rootShard),
		frontproxy.AdminKubeconfigCertificateReconciler(frontProxy, rootShard),
		frontproxy.RequestHeaderCertificateReconciler(frontProxy, rootShard),
	}

	deploymentReconcilers := []k8creconciling.NamedDeploymentReconcilerFactory{
		frontproxy.DeploymentReconciler(frontProxy, rootShard),
	}

	serviceReconcilers := []k8creconciling.NamedServiceReconcilerFactory{
		frontproxy.ServiceReconciler(frontProxy),
	}

	if err := k8creconciling.ReconcileConfigMaps(ctx, configMapReconcilers, frontProxy.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileSecrets(ctx, secretReconcilers, frontProxy.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, frontProxy.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileDeployments(ctx, deploymentReconcilers, frontProxy.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileServices(ctx, serviceReconcilers, frontProxy.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return conditions, kerrors.NewAggregate(errs)
}

func (r *FrontProxyReconciler) reconcileStatus(ctx context.Context, oldFrontProxy *operatorv1alpha1.FrontProxy, conditions []metav1.Condition) error {
	frontProxy := oldFrontProxy.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: frontProxy.Namespace, Name: resources.GetFrontProxyDeploymentName(frontProxy)}
	cond, err := getDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		errs = append(errs, err)
	} else {
		conditions = append(conditions, cond)
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = frontProxy.Generation
		frontProxy.Status.Conditions = updateCondition(frontProxy.Status.Conditions, condition)
	}

	availableCond := apimeta.FindStatusCondition(frontProxy.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))
	switch {
	case availableCond.Status == metav1.ConditionTrue:
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseRunning

	case frontProxy.DeletionTimestamp != nil:
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseDeleting

	case frontProxy.Status.Phase == "":
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseProvisioning
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldFrontProxy.Status, frontProxy.Status) {
		if err := r.Client.Status().Patch(ctx, frontProxy, ctrlruntimeclient.MergeFrom(oldFrontProxy)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
