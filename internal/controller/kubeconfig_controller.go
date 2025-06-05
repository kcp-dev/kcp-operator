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
	"errors"
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/kubeconfig"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// KubeconfigReconciler reconciles a Kubeconfig object
type KubeconfigReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Kubeconfig{}).
		Owns(&corev1.Secret{}).
		Owns(&certmanagerv1.Certificate{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KubeconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var kc operatorv1alpha1.Kubeconfig
	if err := r.Get(ctx, req.NamespacedName, &kc); err != nil {
		return ctrl.Result{}, err
	}

	rootShard := &operatorv1alpha1.RootShard{}
	shard := &operatorv1alpha1.Shard{}

	var (
		clientCertIssuer string
		serverCA         string
	)

	switch {
	case kc.Spec.Target.RootShardRef != nil:
		if err := r.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.RootShardRef.Name, Namespace: req.Namespace}, rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get RootShard: %w", err)
		}

		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
		serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	case kc.Spec.Target.ShardRef != nil:
		if err := r.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.ShardRef.Name, Namespace: req.Namespace}, shard); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get Shard: %w", err)
		}

		ref := shard.Spec.RootShard.Reference
		if ref == nil || ref.Name == "" {
			return ctrl.Result{}, errors.New("the Shard does not reference a (valid) RootShard")
		}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: req.Namespace}, rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get RootShard: %w", err)
		}

		// The client CA is shared among all shards and owned by the root shard.
		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
		serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	case kc.Spec.Target.FrontProxyRef != nil:
		var frontProxy operatorv1alpha1.FrontProxy
		if err := r.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.FrontProxyRef.Name, Namespace: req.Namespace}, &frontProxy); err != nil {
			return ctrl.Result{}, fmt.Errorf("referenced FrontProxy '%s' does not exist", kc.Spec.Target.FrontProxyRef.Name)
		}

		ref := frontProxy.Spec.RootShard.Reference
		if ref == nil || ref.Name == "" {
			return ctrl.Result{}, errors.New("the FrontProxy does not reference a (valid) RootShard")
		}
		if err := r.Get(ctx, types.NamespacedName{Name: frontProxy.Spec.RootShard.Reference.Name, Namespace: req.Namespace}, rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get RootShard: %w", err)
		}

		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA)
		serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	default:
		return ctrl.Result{}, fmt.Errorf("no valid target for kubeconfig found")
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		kubeconfig.ClientCertificateReconciler(&kc, clientCertIssuer),
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	clientCertSecret, err := r.getCertificateSecret(ctx, kc.GetCertificateName(), req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	} else if clientCertSecret == nil {
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	serverCASecret, err := r.getCertificateSecret(ctx, serverCA, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	} else if serverCASecret == nil {
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	reconciler, err := kubeconfig.KubeconfigSecretReconciler(&kc, rootShard, shard, serverCASecret, clientCertSecret)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{reconciler}, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KubeconfigReconciler) getCertificateSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx).WithValues("certificate", name)

	certificate := &certmanagerv1.Certificate{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, certificate); err != nil {
		// Because of how the reconciling framework works, this should never happen.
		logger.V(6).Info("Certificate does not exist yet, trying later ...")
		return nil, nil
	}

	ok := false
	for _, cond := range certificate.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady {
			ok = (cond.Status == certmanagermetav1.ConditionTrue)
			break
		}
	}

	if !ok {
		logger.V(4).Info("Certificate is not ready yet, trying later ...")
		return nil, nil
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: certificate.Spec.SecretName, Namespace: certificate.Namespace}, secret); err != nil {
		return nil, err
	}

	return secret, nil
}
