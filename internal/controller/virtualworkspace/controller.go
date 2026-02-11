/*
Copyright 2026 The KCP Authors.

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

package virtualworkspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"k8c.io/reconciler/pkg/equality"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/metrics"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/reconciling/modifier"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/virtualworkspace"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// Reconciler reconciles a VirtualWorkspace object
type Reconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("virtualworkspace").
		For(&operatorv1alpha1.VirtualWorkspace{}).
		Watches(&operatorv1alpha1.RootShard{}, handler.EnqueueRequestsFromMapFunc(r.mapRootShardToVirtualWorkspaces)).
		Watches(&operatorv1alpha1.Shard{}, handler.EnqueueRequestsFromMapFunc(r.mapShardToVirtualWorkspaces)).
		Watches(&certmanagerv1.Issuer{}, handler.EnqueueRequestsFromMapFunc(r.mapIssuerToVirtualWorkspaces)).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&certmanagerv1.Certificate{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=virtualworkspaces,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=virtualworkspaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.RecordReconciliationMetrics(metrics.VirtualWorkspaceResourceType, duration.Seconds(), nil)
	}()

	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	vw := &operatorv1alpha1.VirtualWorkspace{}
	if err := r.Get(ctx, req.NamespacedName, vw); err != nil {
		// object has been deleted.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		metrics.RecordReconciliationError(metrics.VirtualWorkspaceResourceType, err.Error())
		return ctrl.Result{}, err
	}

	if vw.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	vwCopy := vw.DeepCopy()

	conditions, recErr := r.reconcile(ctx, vwCopy)

	if err := r.reconcileStatus(ctx, vw, vwCopy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

func (r *Reconciler) reconcile(ctx context.Context, vw *operatorv1alpha1.VirtualWorkspace) ([]metav1.Condition, error) {
	var conditions []metav1.Condition

	var (
		rootShard        *operatorv1alpha1.RootShard
		shard            *operatorv1alpha1.Shard
		clientCertIssuer string
	)

	switch {
	case vw.Spec.Target.RootShardRef != nil:
		rootShard = &operatorv1alpha1.RootShard{}

		if err := r.Get(ctx, types.NamespacedName{Name: vw.Spec.Target.RootShardRef.Name, Namespace: vw.Namespace}, rootShard); err != nil {
			err = fmt.Errorf("failed to get RootShard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
		// serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	case vw.Spec.Target.ShardRef != nil:
		shard = &operatorv1alpha1.Shard{}

		if err := r.Get(ctx, types.NamespacedName{Name: vw.Spec.Target.ShardRef.Name, Namespace: vw.Namespace}, shard); err != nil {
			err = fmt.Errorf("failed to get Shard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		ref := shard.Spec.RootShard.Reference
		if ref == nil || ref.Name == "" {
			err := errors.New("the Shard does not reference a (valid) RootShard")
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		rootShard = &operatorv1alpha1.RootShard{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: vw.Namespace}, rootShard); err != nil {
			err = fmt.Errorf("failed to get RootShard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		// The client CA is shared among all shards and owned by the root shard.
		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
		// serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	default:
		err := errors.New("no valid target for VirtualWorkspace found")
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
			Status:  metav1.ConditionFalse,
			Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
			Message: err.Error(),
		})
		return conditions, err
	}

	conditions = append(conditions, metav1.Condition{
		Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
		Status:  metav1.ConditionTrue,
		Reason:  string(operatorv1alpha1.ConditionReasonReferenceValid),
		Message: "Target reference is valid",
	})

	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(vw, operatorv1alpha1.SchemeGroupVersion.WithKind("VirtualWorkspace")))
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, r.Client)

	if err := reconciling.ReconcileCertificates(ctx, []reconciling.NamedCertificateReconcilerFactory{
		virtualworkspace.ClientCertificateReconciler(vw, clientCertIssuer),
		virtualworkspace.ServerCertificateReconciler(vw, rootShard),
	}, vw.Namespace, r.Client, ownerRefWrapper); err != nil {
		return conditions, err
	}

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		virtualworkspace.DeploymentReconciler(vw, rootShard, shard),
	}, vw.Namespace, r.Client, ownerRefWrapper, revisionLabels); err != nil {
		// Swallow these errors and instead rely on us watching Secrets and re-reconciling whenever they change.
		if errors.Is(err, modifier.ErrMountNotFound) {
			err = nil
		}

		return conditions, err
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		virtualworkspace.ServiceReconciler(vw),
	}, vw.Namespace, r.Client, ownerRefWrapper); err != nil {
		return conditions, err
	}

	conditions = append(conditions, metav1.Condition{
		Type:    string(operatorv1alpha1.ConditionTypeAvailable),
		Status:  metav1.ConditionTrue,
		Reason:  "SecretsReady",
		Message: "Client certificate and server CA secrets are ready",
	})

	return conditions, nil
}

func (r *Reconciler) reconcileStatus(ctx context.Context, oldVW *operatorv1alpha1.VirtualWorkspace, vw *operatorv1alpha1.VirtualWorkspace, conditions []metav1.Condition) error {
	for _, condition := range conditions {
		condition.ObservedGeneration = vw.Generation
		vw.Status.Conditions = util.UpdateCondition(vw.Status.Conditions, condition)
	}

	if !equality.Semantic.DeepEqual(oldVW.Status, vw.Status) {
		if err := r.Status().Patch(ctx, vw, ctrlruntimeclient.MergeFrom(oldVW)); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) mapRootShardToVirtualWorkspaces(ctx context.Context, obj ctrlruntimeclient.Object) []ctrl.Request {
	logger := log.FromContext(ctx).WithValues("rootShard", obj.GetName())
	logger.V(4).Info("Mapping RootShard to VirtualWorkspaces")

	return r.mapVirtualWorkspaces(ctx, func(target operatorv1alpha1.VirtualWorkspaceTarget) bool {
		return target.RootShardRef != nil && target.RootShardRef.Name == obj.GetName()
	})
}

func (r *Reconciler) mapShardToVirtualWorkspaces(ctx context.Context, obj ctrlruntimeclient.Object) []ctrl.Request {
	logger := log.FromContext(ctx).WithValues("shard", obj.GetName())
	logger.V(4).Info("Mapping Shard to VirtualWorkspaces")

	return r.mapVirtualWorkspaces(ctx, func(target operatorv1alpha1.VirtualWorkspaceTarget) bool {
		return target.ShardRef != nil && target.ShardRef.Name == obj.GetName()
	})
}

func (r *Reconciler) mapIssuerToVirtualWorkspaces(ctx context.Context, obj ctrlruntimeclient.Object) []ctrl.Request {
	logger := log.FromContext(ctx).WithValues("issuer", obj.GetName())
	logger.V(4).Info("Mapping Issuer to VirtualWorkspaces")

	// Find all VirtualWorkspaces that use this Issuer for their client certificate
	var virtualWorkspaces operatorv1alpha1.VirtualWorkspaceList
	if err := r.List(ctx, &virtualWorkspaces, ctrlruntimeclient.InNamespace(obj.GetNamespace())); err != nil {
		logger.Error(err, "Failed to list VirtualWorkspaces")
		return []ctrl.Request{}
	}

	var requests []ctrl.Request
	for _, vw := range virtualWorkspaces.Items {
		var expectedIssuer string
		switch {
		case vw.Spec.Target.RootShardRef != nil:
			rootShard := &operatorv1alpha1.RootShard{}
			if err := r.Get(ctx, types.NamespacedName{Name: vw.Spec.Target.RootShardRef.Name, Namespace: vw.Namespace}, rootShard); err == nil {
				expectedIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
			}
		case vw.Spec.Target.ShardRef != nil:
			shard := &operatorv1alpha1.Shard{}
			if err := r.Get(ctx, types.NamespacedName{Name: vw.Spec.Target.ShardRef.Name, Namespace: vw.Namespace}, shard); err == nil {
				if ref := shard.Spec.RootShard.Reference; ref != nil {
					rootShard := &operatorv1alpha1.RootShard{}
					if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: vw.Namespace}, rootShard); err == nil {
						expectedIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
					}
				}
			}
		}

		if expectedIssuer == obj.GetName() {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      vw.Name,
					Namespace: vw.Namespace,
				},
			})
		}
	}

	return requests
}

func (r *Reconciler) mapVirtualWorkspaces(ctx context.Context, matches func(t operatorv1alpha1.VirtualWorkspaceTarget) bool) []ctrl.Request {
	var virtualWorkspaces operatorv1alpha1.VirtualWorkspaceList
	if err := r.List(ctx, &virtualWorkspaces); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list VirtualWorkspaces")
		return []ctrl.Request{}
	}

	var requests []ctrl.Request
	for _, vw := range virtualWorkspaces.Items {
		if matches(vw.Spec.Target) {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      vw.Name,
					Namespace: vw.Namespace,
				},
			})
		}
	}

	return requests
}
