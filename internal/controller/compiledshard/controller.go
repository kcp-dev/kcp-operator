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

package compiledshard

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/kcp-dev/logicalcluster/v3"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/client"
	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/reconciling/modifier"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/shard"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// requeueAfter is the retry delay when a mounted Secret or ConfigMap does not exist yet.
const requeueAfter = 10 * time.Second

const cleanupFinalizer = "operator.kcp.io/cleanup-shard"

// Reconciler reconciles a CompiledShard object.
type Reconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("compiledshard").
		For(&deployv1alpha1.CompiledShard{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledshards,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledshards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledshards/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var compiled deployv1alpha1.CompiledShard
	if err := r.Get(ctx, req.NamespacedName, &compiled); err != nil {
		return ctrl.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if compiled.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &compiled)
	}

	// Ensure the finalizer before any other work.
	updated, err := r.ensureFinalizer(ctx, &compiled)
	if err != nil {
		return ctrl.Result{}, err
	}
	if updated {
		// Will be requeued by the patch event.
		return ctrl.Result{}, nil
	}

	result, recErr := r.reconcile(ctx, &compiled)
	statusErr := r.reconcileStatus(ctx, &compiled)

	return result, kerrors.NewAggregate([]error{recErr, statusErr})
}

func (r *Reconciler) reconcile(ctx context.Context, compiled *deployv1alpha1.CompiledShard) (ctrl.Result, error) {
	var (
		errs    []error
		requeue bool
	)

	s := &operatorv1alpha1.Shard{
		ObjectMeta: metav1.ObjectMeta{
			Name:        compiled.Name,
			Namespace:   compiled.Namespace,
			Labels:      compiled.Labels,
			Annotations: compiled.Annotations,
		},
		Spec: compiled.Spec.Shard,
	}

	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Spec.RootShard.Name,
			Namespace: compiled.Namespace,
		},
		Spec: compiled.Spec.RootShard.Spec,
	}

	var kcpVW *operatorv1alpha1.VirtualWorkspace
	if compiled.Spec.VirtualWorkspace != nil {
		kcpVW = &operatorv1alpha1.VirtualWorkspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      compiled.Spec.VirtualWorkspace.Name,
				Namespace: compiled.Namespace,
			},
			Spec: compiled.Spec.VirtualWorkspace.Spec,
		}
	}

	ownerRef := *metav1.NewControllerRef(compiled, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledShard"))
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(ownerRef)
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, r.Client)

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		shard.DeploymentReconciler(s, rootShard, kcpVW),
	}, compiled.Namespace, r.Client, ownerRefWrapper, revisionLabels); err != nil {
		if errors.Is(err, modifier.ErrMountNotFound) {
			requeue = true
		} else {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		shard.ServiceReconciler(s),
	}, compiled.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	result := ctrl.Result{}
	if requeue {
		result.RequeueAfter = requeueAfter
	}

	return result, kerrors.NewAggregate(errs)
}

func (r *Reconciler) reconcileStatus(ctx context.Context, oldCompiled *deployv1alpha1.CompiledShard) error {
	compiled := oldCompiled.DeepCopy()

	s := &operatorv1alpha1.Shard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Name,
			Namespace: compiled.Namespace,
		},
	}

	depKey := types.NamespacedName{Namespace: compiled.Namespace, Name: resources.GetShardDeploymentName(s)}
	cond, err := util.GetDeploymentAvailableCondition(ctx, r.Client, depKey)
	if err != nil {
		return err
	}

	cond.ObservedGeneration = compiled.Generation
	compiled.Status.Conditions = util.UpdateCondition(compiled.Status.Conditions, cond)

	compiled.Status.Phase = operatorv1alpha1.ShardPhaseProvisioning
	if availableCond := apimeta.FindStatusCondition(compiled.Status.Conditions, string(operatorv1alpha1.ConditionTypeAvailable)); availableCond != nil && availableCond.Status == metav1.ConditionTrue {
		compiled.Status.Phase = operatorv1alpha1.ShardPhaseRunning
	}

	if !equality.Semantic.DeepEqual(oldCompiled.Status, compiled.Status) {
		return r.Status().Patch(ctx, compiled, ctrlruntimeclient.MergeFrom(oldCompiled))
	}

	return nil
}

// handleDeletion deletes the Deployment and waits until all Pods are gone, then deletes the Shard from the root workspace.
func (r *Reconciler) handleDeletion(ctx context.Context, compiled *deployv1alpha1.CompiledShard) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !slices.Contains(compiled.Finalizers, cleanupFinalizer) {
		return ctrl.Result{}, nil
	}

	s := &operatorv1alpha1.Shard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Name,
			Namespace: compiled.Namespace,
		},
		Spec: compiled.Spec.Shard,
	}

	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      compiled.Spec.RootShard.Name,
			Namespace: compiled.Namespace,
		},
		Spec: compiled.Spec.RootShard.Spec,
	}

	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Namespace: compiled.Namespace, Name: resources.GetShardDeploymentName(s)}, deployment)
	switch {
	case err == nil:
		if deployment.DeletionTimestamp == nil {
			if err := r.Delete(ctx, deployment); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	case !apierrors.IsNotFound(err):
		return ctrl.Result{}, err
	}

	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, ctrlruntimeclient.InNamespace(compiled.Namespace), ctrlruntimeclient.MatchingLabels(resources.GetShardResourceLabels(s))); err != nil {
		return ctrl.Result{}, err
	}
	if len(pods.Items) > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	kcpClient, err := client.NewRootShardClientWithShardCert(ctx, r.Client, s, rootShard, logicalcluster.NewPath("root"), r.Scheme)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Without the certificate secrets we cannot talk to the root shard anymore.
			logger.Info("Certificate secrets are gone, skipping kcp Shard cleanup.")
			return ctrl.Result{}, r.removeFinalizer(ctx, compiled)
		}
		return ctrl.Result{}, err
	}

	if err := kcpClient.Delete(ctx, &kcpcorev1alpha1.Shard{ObjectMeta: metav1.ObjectMeta{Name: compiled.Name}}); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		logger.V(4).Info("kcp Shard object already deleted.")
	}

	return ctrl.Result{}, r.removeFinalizer(ctx, compiled)
}

func (r *Reconciler) ensureFinalizer(ctx context.Context, compiled *deployv1alpha1.CompiledShard) (bool, error) {
	finalizers := sets.New(compiled.GetFinalizers()...)
	if finalizers.Has(cleanupFinalizer) {
		return false, nil
	}

	original := compiled.DeepCopy()
	finalizers.Insert(cleanupFinalizer)
	compiled.SetFinalizers(sets.List(finalizers))

	return true, r.Patch(ctx, compiled, ctrlruntimeclient.MergeFrom(original))
}

func (r *Reconciler) removeFinalizer(ctx context.Context, compiled *deployv1alpha1.CompiledShard) error {
	finalizers := sets.New(compiled.GetFinalizers()...)
	if !finalizers.Has(cleanupFinalizer) {
		return nil
	}

	original := compiled.DeepCopy()
	finalizers.Delete(cleanupFinalizer)
	compiled.SetFinalizers(sets.List(finalizers))

	return r.Patch(ctx, compiled, ctrlruntimeclient.MergeFrom(original))
}
