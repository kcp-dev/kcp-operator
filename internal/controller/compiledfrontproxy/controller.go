/*
Copyright 2026 The kcp Authors.

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

package compiledfrontproxy

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/frontproxy"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// requeueAfter is the retry delay when a mounted Secret or ConfigMap does not exist yet.
const requeueAfter = 10 * time.Second

// Reconciler reconciles a CompiledFrontProxy object.
type Reconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("compiledfrontproxy").
		For(&deployv1alpha1.CompiledFrontProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledfrontproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var compiled deployv1alpha1.CompiledFrontProxy
	if err := r.Get(ctx, req.NamespacedName, &compiled); err != nil {
		return ctrl.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if compiled.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	result, recErr := r.reconcile(ctx, &compiled)
	statusErr := r.reconcileStatus(ctx, &compiled)

	return result, kerrors.NewAggregate([]error{recErr, statusErr})
}

func (r *Reconciler) reconcile(ctx context.Context, compiled *deployv1alpha1.CompiledFrontProxy) (ctrl.Result, error) {
	frontProxy := &operatorv1alpha1.FrontProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        compiled.Name,
			Namespace:   compiled.Namespace,
			Labels:      compiled.Labels,
			Annotations: compiled.Annotations,
		},
		Spec: compiled.Spec.FrontProxy,
	}

	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Spec.RootShard.Name,
			Namespace: compiled.Namespace,
		},
		Spec: compiled.Spec.RootShard.Spec,
	}

	ownerRef := *metav1.NewControllerRef(compiled, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledFrontProxy"))

	requeue, err := frontproxy.NewFrontProxy(frontProxy, rootShard).ReconcileWorkload(ctx, r.Client, compiled.Namespace, ownerRef)

	result := ctrl.Result{}
	if requeue {
		result.RequeueAfter = requeueAfter
	}

	return result, err
}

func (r *Reconciler) reconcileStatus(ctx context.Context, oldCompiled *deployv1alpha1.CompiledFrontProxy) error {
	compiled := oldCompiled.DeepCopy()

	frontProxy := &operatorv1alpha1.FrontProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Name,
			Namespace: compiled.Namespace,
		},
	}

	depKey := types.NamespacedName{Namespace: compiled.Namespace, Name: resources.GetFrontProxyDeploymentName(frontProxy)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		return err
	}

	cond.ObservedGeneration = compiled.Generation
	compiled.Status.Conditions = util.UpdateCondition(compiled.Status.Conditions, cond)

	compiled.Status.Phase = operatorv1alpha1.FrontProxyPhaseProvisioning
	if availableCond := apimeta.FindStatusCondition(compiled.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable)); availableCond != nil && availableCond.Status == metav1.ConditionTrue {
		compiled.Status.Phase = operatorv1alpha1.FrontProxyPhaseRunning
	}

	if !equality.Semantic.DeepEqual(oldCompiled.Status, compiled.Status) {
		return r.Status().Patch(ctx, compiled, ctrlruntimeclient.MergeFrom(oldCompiled))
	}

	return nil
}
