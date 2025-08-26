/*
Copyright 2025 The KCP Authors.

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

package shardbootstrapping

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8c.io/reconciler/pkg/equality"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	ctrlresources "github.com/kcp-dev/kcp-operator/internal/resources/shardbootstrapping"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// ShardBootstrappingReconciler reconciles Shards and RootShards and ensures that on each
// of them, a dedicated ClusterRoleBinding for the kcp-operator is bootstrapped in
// the shard local system:admin cluster.
type ShardBootstrappingReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

const (
	rootShardKind = "RootShard"
	shardKind     = "Shard"
)

func newWatchHandlerFunc(kind string) handler.TypedEventHandler[ctrlruntimeclient.Object, reconcile.Request] {
	return handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		key := ctrlruntimeclient.ObjectKeyFromObject(obj)

		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: kind,
				Name:      key.String(),
			},
		}}
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShardBootstrappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("shard-bootstrapping-controller").
		Watches(&operatorv1alpha1.Shard{}, newWatchHandlerFunc(shardKind)).
		Watches(&operatorv1alpha1.RootShard{}, newWatchHandlerFunc(rootShardKind)).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch

func (r *ShardBootstrappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	kind := req.Namespace
	keyParts := strings.SplitN(req.Name, string(types.Separator), 2)
	key := types.NamespacedName{
		Namespace: keyParts[0],
		Name:      keyParts[1],
	}

	logger := log.FromContext(ctx)
	logger.V(4).Info(fmt.Sprintf("Reconciling %s object", kind))

	var serviceName string

	shard := &operatorv1alpha1.Shard{}
	rootShard := &operatorv1alpha1.RootShard{}

	switch kind {
	case shardKind:
		if err := r.Get(ctx, key, shard); err != nil {
			if ctrlruntimeclient.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get Shard: %w", err)
			}

			return ctrl.Result{}, nil
		}

		if shard.DeletionTimestamp != nil || shard.Status.Phase != operatorv1alpha1.ShardPhaseRunning {
			return ctrl.Result{}, nil
		}

		serviceName = resources.GetShardServiceName(shard)

		ref := shard.Spec.RootShard.Reference
		if ref == nil || ref.Name == "" {
			return ctrl.Result{}, errors.New("the Shard does not reference a (valid) RootShard")
		}

		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: key.Namespace}, rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get RootShard: %w", err)
		}

	case rootShardKind:
		if err := r.Get(ctx, key, rootShard); err != nil {
			if ctrlruntimeclient.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get RootShard: %w", err)
			}

			return ctrl.Result{}, nil
		}

		if rootShard.DeletionTimestamp != nil || rootShard.Status.Phase != operatorv1alpha1.RootShardPhaseRunning {
			return ctrl.Result{}, nil
		}

		serviceName = resources.GetRootShardServiceName(rootShard)

	default:
		panic(fmt.Sprintf("Unexpected object kind %q in reconcile request.", kind))
	}

	// We use the same client cert to connect to all of the shards and root shard.
	secretName := resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.OperatorCertificate)

	conditions, recErr := r.reconcile(ctx, key.Namespace, secretName, serviceName)

	switch kind {
	case shardKind:
		oldShard := shard.DeepCopy()
		upsertConditions(shard, conditions)

		if !equality.Semantic.DeepEqual(oldShard.Status, shard.Status) {
			if err := r.Status().Patch(ctx, shard, ctrlruntimeclient.MergeFrom(oldShard)); err != nil {
				recErr = kerrors.NewAggregate([]error{recErr, err})
			}
		}

	case rootShardKind:
		oldRootShard := rootShard.DeepCopy()
		upsertConditions(rootShard, conditions)

		if !equality.Semantic.DeepEqual(oldRootShard.Status, rootShard.Status) {
			if err := r.Status().Patch(ctx, rootShard, ctrlruntimeclient.MergeFrom(oldRootShard)); err != nil {
				recErr = kerrors.NewAggregate([]error{recErr, err})
			}
		}
	}

	return ctrl.Result{}, recErr
}

func errorCondition(err error) []metav1.Condition {
	return []metav1.Condition{{
		Type:    string(operatorv1alpha1.ConditionTypeBootstrapping),
		Status:  metav1.ConditionFalse,
		Reason:  string(operatorv1alpha1.ConditionReasonBootstrappingPending),
		Message: fmt.Sprintf("Bootstrapping failed: %v.", err),
	}}
}

func (r *ShardBootstrappingReconciler) reconcile(ctx context.Context, namespace, secretName, serviceName string) ([]metav1.Condition, error) {
	certSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, certSecret); err != nil {
		err = fmt.Errorf("failed to get kubeconfig Secret: %w", err)
		c := errorCondition(err)
		return c, err
	}

	cfg := &rest.Config{
		Host: fmt.Sprintf("https://%s.%s.svc.cluster.local:6443/clusters/system:admin", serviceName, namespace),
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   certSecret.Data["ca.crt"],
			CertData: certSecret.Data["tls.crt"],
			KeyData:  certSecret.Data["tls.key"],
		},
	}

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
	if err != nil {
		err = fmt.Errorf("failed to create shard client: %w", err)
		c := errorCondition(err)
		return c, err
	}

	if err := reconciling.ReconcileClusterRoles(ctx, []reconciling.NamedClusterRoleReconcilerFactory{
		ctrlresources.ClusterRoleReconciler(),
	}, "", client); err != nil {
		err = fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
		c := errorCondition(err)
		return c, err
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingReconcilerFactory{
		ctrlresources.ClusterRoleBindingReconciler(),
	}, "", client); err != nil {
		err = fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
		c := errorCondition(err)
		return c, err
	}

	return []metav1.Condition{{
		Type:    string(operatorv1alpha1.ConditionTypeBootstrapping),
		Status:  metav1.ConditionTrue,
		Reason:  string(operatorv1alpha1.ConditionReasonBootstrappingCompleted),
		Message: "All resources have been created.",
	}}, nil
}

type conditionedObject interface {
	ctrlruntimeclient.Object
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
}

func upsertConditions(obj conditionedObject, newConditions []metav1.Condition) {
	objConditions := obj.GetConditions()
	for _, condition := range newConditions {
		condition.ObservedGeneration = obj.GetGeneration()
		objConditions = util.UpdateCondition(objConditions, condition)
	}

	obj.SetConditions(objConditions)
}
