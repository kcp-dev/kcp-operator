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

package cacheserver

import (
	"context"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources/cacheserver"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// CacheServerReconciler reconciles a CacheServer object
type CacheServerReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

func (r *CacheServerReconciler) SetupWithManager(mgr ctrlruntime.Manager) error {
	return ctrlruntime.NewControllerManagedBy(mgr).
		Named("cache-server").
		For(&operatorv1alpha1.CacheServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&certmanagerv1.Certificate{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=cacheservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kcp.io,resources=cacheservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=cacheservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *CacheServerReconciler) Reconcile(ctx context.Context, req ctrlruntime.Request) (ctrlruntime.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	server := &operatorv1alpha1.CacheServer{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		return ctrlruntime.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if server.DeletionTimestamp != nil {
		return ctrlruntime.Result{}, nil
	}

	if err := r.reconcile(ctx, server); err != nil {
		return ctrlruntime.Result{}, err
	}

	return ctrlruntime.Result{}, nil
}

func (r *CacheServerReconciler) reconcile(ctx context.Context, server *operatorv1alpha1.CacheServer) error {
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*metav1.NewControllerRef(server, operatorv1alpha1.SchemeGroupVersion.WithKind("CacheServer")))

	if err := reconciling.ReconcileCertificates(ctx, []reconciling.NamedCertificateReconcilerFactory{
		cacheserver.RootCACertificateReconciler(server),
		cacheserver.ServerCertificateReconciler(server),
	}, server.Namespace, r.Client, ownerRefWrapper); err != nil {
		return err
	}

	if err := reconciling.ReconcileIssuers(ctx, []reconciling.NamedIssuerReconcilerFactory{
		cacheserver.RootCAIssuerReconciler(server),
	}, server.Namespace, r.Client, ownerRefWrapper); err != nil {
		return err
	}

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		cacheserver.DeploymentReconciler(server),
	}, server.Namespace, r.Client, ownerRefWrapper); err != nil {
		return err
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{
		cacheserver.KubeconfigReconciler(server),
	}, server.Namespace, r.Client, ownerRefWrapper); err != nil {
		return err
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		cacheserver.ServiceReconciler(server),
	}, server.Namespace, r.Client, ownerRefWrapper); err != nil {
		return err
	}

	return nil
}
