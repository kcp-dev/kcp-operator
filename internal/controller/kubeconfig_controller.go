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
	"net/url"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorkcpiov1alpha1 "github.com/kcp-dev/kcp-operator/api/v1alpha1"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources/kubeconfig"
)

// KubeconfigReconciler reconciles a Kubeconfig object
type KubeconfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
	logger.Info("Reconciling Kubeconfig object")

	var kc operatorkcpiov1alpha1.Kubeconfig
	if err := r.Client.Get(ctx, req.NamespacedName, &kc); err != nil {
		return ctrl.Result{}, err
	}

	var (
		issuer, serverURL, serverName string
	)

	switch {
	case kc.Spec.Target.RootShardRef != nil:
		var rootShard operatorkcpiov1alpha1.RootShard
		if err := r.Client.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.RootShardRef.Name, Namespace: req.Namespace}, &rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("referenced RootShard '%s' does not exist", kc.Spec.Target.RootShardRef.Name)
		}
		issuer = rootShard.GetCAName(operatorkcpiov1alpha1.ClientCA)
		serverURL = rootShard.GetShardBaseURL()
		serverName = rootShard.Name
	default:
		return ctrl.Result{}, fmt.Errorf("no valid target for kubeconfig found")
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		kubeconfig.ClientCertificateReconciler(&kc, issuer),
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	var certificate certmanagerv1.Certificate
	if err := r.Client.Get(ctx, types.NamespacedName{Name: kc.GetCertificateName(), Namespace: req.Namespace}, &certificate); err != nil {
		logger.V(6).Info("Certificate does not exist yet, trying later ...", "certificate", kc.GetCertificateName())
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	ok := false
	for _, cond := range certificate.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady {
			ok = (cond.Status == certmanagermetav1.ConditionTrue)
			break
		}
	}

	if !ok {
		logger.Info("Certificate is not ready yet, trying later ...", "certificate", certificate.Name)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	var secret corev1.Secret
	if err := r.Client.Get(ctx, types.NamespacedName{Name: certificate.Spec.SecretName, Namespace: certificate.Namespace}, &secret); err != nil {
		return ctrl.Result{}, err
	}

	rootWSURL, err := url.JoinPath(serverURL, "clusters", "root")
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
		kubeconfig.KubeconfigSecretReconciler(&kc, &secret, serverName, rootWSURL),
	}, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorkcpiov1alpha1.Kubeconfig{}).
		Complete(r)
}
