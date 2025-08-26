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

package provisioning

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	presources "github.com/kcp-dev/kcp-operator/internal/resources/provisioning"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// ProvisioningReconciler reconciles Shards and RootShards and ensures that on each
// of them, a dedicated ClusterRoleBinding for the kcp-operator is provisioned in
// the shard local system:admin cluster.
type ProvisioningReconciler struct {
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
func (r *ProvisioningReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("provisioning-controller").
		Watches(&operatorv1alpha1.Shard{}, newWatchHandlerFunc(shardKind)).
		Watches(&operatorv1alpha1.RootShard{}, newWatchHandlerFunc(rootShardKind)).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=shards,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch

func (r *ProvisioningReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, recErr error) {
	kind := req.Namespace
	keyParts := strings.SplitN(req.Name, string(types.Separator), 2)
	key := types.NamespacedName{
		Namespace: keyParts[0],
		Name:      keyParts[1],
	}

	logger := log.FromContext(ctx)
	logger.V(4).Info(fmt.Sprintf("Reconciling %s object", kind))

	var serviceName string
	rootShard := &operatorv1alpha1.RootShard{}

	switch kind {
	case shardKind:
		var s operatorv1alpha1.Shard
		if err := r.Get(ctx, key, &s); err != nil {
			if ctrlruntimeclient.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get Shard: %w", err)
			}

			return ctrl.Result{}, nil
		}

		serviceName = resources.GetShardServiceName(&s)

		ref := s.Spec.RootShard.Reference
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

		serviceName = resources.GetRootShardServiceName(rootShard)

	default:
		panic(fmt.Sprintf("Unexpected object kind %q in reconcile request.", kind))
	}

	// We use the same client cert to connect to all of the shards and root shard.
	secretName := resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.OperatorCertificate)

	recErr = r.provision(ctx, key.Namespace, secretName, serviceName)

	return ctrl.Result{}, recErr
}

func (r *ProvisioningReconciler) provision(ctx context.Context, namespace, secretName, serviceName string) error {
	certSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, certSecret); err != nil {
		return fmt.Errorf("failed to get kubeconfig Secret: %w", err)
	}

	cfg2 := &rest.Config{
		// Host: fmt.Sprintf("https://%s.%s.svc.cluster.local:6443/clusters/system:admin", serviceName, namespace),
		Host: "https://localhost:6443/clusters/system:admin",
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   certSecret.Data["ca.crt"],
			CertData: certSecret.Data["tls.crt"],
			KeyData:  certSecret.Data["tls.key"],
		},
	}

	c, err := client.New(cfg2, ctrlruntimeclient.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := reconciling.ReconcileClusterRoles(ctx, []reconciling.NamedClusterRoleReconcilerFactory{
		presources.ClusterRoleReconciler(),
	}, "", c); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoles: %w", err)
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, []reconciling.NamedClusterRoleBindingReconcilerFactory{
		presources.ClusterRoleBindingReconciler(),
	}, "", c); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	return nil
}
