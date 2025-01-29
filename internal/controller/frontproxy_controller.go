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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/frontproxy"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// FrontProxyReconciler reconciles a FrontProxy object
type FrontProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *FrontProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.FrontProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&certmanagerv1.Certificate{}).
		Watches(&corev1.Secret{}, newSecretGrandchildWatcher(resources.FrontProxyLabel)).
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
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get FrontProxy object: %w", err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	defer func() {
		if err := r.reconcileStatus(ctx, &frontProxy); err != nil {
			recErr = kerrors.NewAggregate([]error{recErr, err})
		}
	}()

	return ctrl.Result{}, r.reconcile(ctx, &frontProxy)
}

func (r *FrontProxyReconciler) reconcile(ctx context.Context, frontProxy *operatorv1alpha1.FrontProxy) error {
	var errs []error

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(frontProxy, operatorv1alpha1.SchemeGroupVersion.WithKind("FrontProxy")))

	ref := frontProxy.Spec.RootShard.Reference
	if ref == nil {
		return fmt.Errorf("no valid RootShard in FrontProxy spec defined")
	}

	rootShard := &operatorv1alpha1.RootShard{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: frontProxy.Namespace}, rootShard); err != nil {
		return fmt.Errorf("referenced RootShard '%s' could not be fetched", ref.Name)
	}

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

	return kerrors.NewAggregate(errs)
}

func (r *FrontProxyReconciler) reconcileStatus(ctx context.Context, oldFrontProxy *operatorv1alpha1.FrontProxy) error {
	frontProxy := oldFrontProxy.DeepCopy()
	var errs []error

	if frontProxy.Status.Phase == "" {
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseProvisioning
	}

	if frontProxy.DeletionTimestamp != nil {
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseDeleting
	}

	if err := r.setAvailableCondition(ctx, frontProxy); err != nil {
		errs = append(errs, err)
	}

	if cond := apimeta.FindStatusCondition(frontProxy.Status.Conditions, string(operatorv1alpha1.RootShardConditionTypeAvailable)); cond.Status == metav1.ConditionTrue {
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseRunning
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldFrontProxy.Status, frontProxy.Status) {
		if err := r.Client.Status().Patch(ctx, frontProxy, client.MergeFrom(oldFrontProxy)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}

func (r *FrontProxyReconciler) setAvailableCondition(ctx context.Context, frontProxy *operatorv1alpha1.FrontProxy) error {
	var dep appsv1.Deployment
	depKey := types.NamespacedName{Namespace: frontProxy.Namespace, Name: resources.GetFrontProxyDeploymentName(frontProxy)}
	if err := r.Client.Get(ctx, depKey, &dep); client.IgnoreNotFound(err) != nil {
		return err
	}

	available := metav1.ConditionFalse
	reason := operatorv1alpha1.FrontProxyConditionReasonDeploymentUnavailable
	msg := deploymentStatusString(dep, depKey)

	if dep.Name != "" {
		if deploymentReady(dep) {
			available = metav1.ConditionTrue
			reason = operatorv1alpha1.FrontProxyConditionReasonReplicasUp
		} else {
			available = metav1.ConditionFalse
			reason = operatorv1alpha1.FrontProxyConditionReasonReplicasUnavailable
		}
	}

	frontProxy.Status.Conditions = updateCondition(frontProxy.Status.Conditions, metav1.Condition{
		Type:               string(operatorv1alpha1.FrontProxyConditionTypeAvailable),
		Status:             available,
		ObservedGeneration: frontProxy.Generation,
		Reason:             string(reason),
		Message:            msg,
	})

	return nil
}
