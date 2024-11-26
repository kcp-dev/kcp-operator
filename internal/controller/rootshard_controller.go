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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/api/v1alpha1"
	operatorkcpiov1alpha1 "github.com/kcp-dev/kcp-operator/api/v1alpha1"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources/rootshard"
)

// RootShardReconciler reconciles a RootShard object
type RootShardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=kcpinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kcpinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kcpinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RootShard object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *RootShardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling RootShard object")
	var rootShard v1alpha1.RootShard
	if err := r.Client.Get(ctx, req.NamespacedName, &rootShard); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to find %s/%s: %w", req.Namespace, req.Name, err)
	}

	// Intermediate CAs that we need to generate a certificate and an issuer for.
	subordinateCAs := []string{
		"requestheader-client",
		"client",
		"service-account",
	}

	caIssuerReconciler, caIssuerName := rootshard.RootCAIssuerReconciler(&rootShard)

	issuerReconcilers := []reconciling.NamedIssuerReconcilerFactory{
		caIssuerReconciler,
	}
	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{}

	for _, ca := range subordinateCAs {
		certReconcilers = append(certReconcilers, rootshard.CaCertificateReconciler(&rootShard, ca, caIssuerName))
		issuerReconcilers = append(issuerReconcilers, rootshard.CAIssuerReconciler(&rootShard, ca))
	}
	if rootShard.Spec.Certificates.IssuerRef != nil {
		certReconcilers = append(certReconcilers, rootshard.RootCaCertificateReconciler(&rootShard))
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if err := reconciling.ReconcileIssuers(ctx, issuerReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootShardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorkcpiov1alpha1.RootShard{}).
		Complete(r)
}
