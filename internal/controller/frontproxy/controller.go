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

package frontproxy

import (
	"context"
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

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
	"github.com/kcp-dev/kcp-operator/internal/metrics"
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
		Named("frontproxy").
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
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.FrontProxyResourceType, duration.Seconds(), recErr)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var frontProxy operatorv1alpha1.FrontProxy
	if err := r.Get(ctx, req.NamespacedName, &frontProxy); err != nil {
		if ctrlruntimeclient.IgnoreNotFound(err) != nil {
			metrics.RecordReconciliationError(metrics.FrontProxyResourceType, err.Error())
			return ctrl.Result{}, fmt.Errorf("failed to get FrontProxy object: %w", err)
		}

		// Object has apparently been deleted already.
		return ctrl.Result{}, nil
	}

	conditions, recErr := r.reconcile(ctx, &frontProxy)

	if err := r.reconcileStatus(ctx, &frontProxy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	metrics.RecordObjectMetrics(
		metrics.FrontProxyResourceType,
		frontProxy.Name,
		req.Namespace,
		string(frontProxy.Status.Phase),
		frontProxy.Status.Conditions,
	)

	return ctrl.Result{}, recErr
}

func (r *FrontProxyReconciler) reconcile(ctx context.Context, frontProxy *operatorv1alpha1.FrontProxy) ([]metav1.Condition, error) {
	var conditions []metav1.Condition

	if frontProxy.DeletionTimestamp != nil {
		return conditions, nil
	}

	cond, rootShard := util.FetchRootShard(ctx, r.Client, frontProxy.Namespace, frontProxy.Spec.RootShard.Reference)
	conditions = append(conditions, cond)

	if rootShard == nil {
		return conditions, nil
	}

	fpReconciler := frontproxy.NewFrontProxy(frontProxy, rootShard)

	if err := fpReconciler.Reconcile(ctx, r.Client, frontProxy.Namespace); err != nil {
		return conditions, fmt.Errorf("failed to reconcile: %w", err)
	}

	return conditions, nil
}

func (r *FrontProxyReconciler) reconcileStatus(ctx context.Context, oldFrontProxy *operatorv1alpha1.FrontProxy, conditions []metav1.Condition) error {
	frontProxy := oldFrontProxy.DeepCopy()
	var errs []error

	depKey := types.NamespacedName{Namespace: frontProxy.Namespace, Name: resources.GetFrontProxyDeploymentName(frontProxy)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		errs = append(errs, err)
	} else {
		conditions = append(conditions, cond)
	}

	for _, condition := range conditions {
		condition.ObservedGeneration = frontProxy.Generation
		frontProxy.Status.Conditions = util.UpdateCondition(frontProxy.Status.Conditions, condition)
	}

	if frontProxy.DeletionTimestamp != nil {
		frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseDeleting
	} else {
		availableCond := apimeta.FindStatusCondition(frontProxy.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable))
		if availableCond != nil && availableCond.Status == metav1.ConditionTrue {
			frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseRunning
		} else {
			frontProxy.Status.Phase = operatorv1alpha1.FrontProxyPhaseProvisioning
		}
	}

	// only patch the status if there are actual changes.
	if !equality.Semantic.DeepEqual(oldFrontProxy.Status, frontProxy.Status) {
		if err := r.Status().Patch(ctx, frontProxy, ctrlruntimeclient.MergeFrom(oldFrontProxy)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
