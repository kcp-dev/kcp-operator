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

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources/frontproxy"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// FrontProxyReconciler reconciles a FrontProxy object
type FrontProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=frontproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kcp.io,resources=frontproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=frontproxies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FrontProxy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *FrontProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling FrontProxy object")
	var frontProxy operatorv1alpha1.FrontProxy
	if err := r.Client.Get(ctx, req.NamespacedName, &frontProxy); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to find %s/%s: %w", req.Namespace, req.Name, err)
	}

	ref := frontProxy.Spec.RootShard.Reference
	rootShard := &operatorv1alpha1.RootShard{}
	switch {
	case ref != nil:
		if err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: req.Namespace}, rootShard); err != nil {
			return ctrl.Result{}, fmt.Errorf("referenced RootShard '%s' could not be fetched", ref.Name)
		}
	default:
		return ctrl.Result{}, fmt.Errorf("no valid RootShard in FrontProxy spec defined")
	}

	configMapReconcilers := []k8creconciling.NamedConfigMapReconcilerFactory{
		frontproxy.ConfigmapReconciler(&frontProxy),
	}

	secretReconcilers := []k8creconciling.NamedSecretReconcilerFactory{
		frontproxy.DynamicKubeconfigSecretReconciler(&frontProxy, rootShard),
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		frontproxy.ServerCertificateReconciler(&frontProxy, rootShard),
		frontproxy.KubeconfigReconciler(&frontProxy, rootShard),
		frontproxy.AdminKubeconfigReconciler(&frontProxy, rootShard),
		frontproxy.RequestHeaderReconciler(&frontProxy, rootShard),
	}

	deploymentReconcilers := []k8creconciling.NamedDeploymentReconcilerFactory{
		frontproxy.DeploymentReconciler(&frontProxy, rootShard),
	}

	if err := k8creconciling.ReconcileConfigMaps(ctx, configMapReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if err := k8creconciling.ReconcileSecrets(ctx, secretReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if err := k8creconciling.ReconcileDeployments(ctx, deploymentReconcilers, req.Namespace, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FrontProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.FrontProxy{}).
		Complete(r)
}
