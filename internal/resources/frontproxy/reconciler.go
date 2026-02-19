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

package frontproxy

import (
	"context"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

type reconciler struct {
	frontProxy     *operatorv1alpha1.FrontProxy
	rootShard      *operatorv1alpha1.RootShard
	resourceLabels map[string]string
}

func NewFrontProxy(frontProxy *operatorv1alpha1.FrontProxy, rootShard *operatorv1alpha1.RootShard) *reconciler {
	if frontProxy == nil {
		panic("Use NewRootShardProxy instead.")
	}

	return &reconciler{
		frontProxy:     frontProxy,
		rootShard:      rootShard,
		resourceLabels: resources.GetFrontProxyResourceLabels(frontProxy),
	}
}

func NewRootShardProxy(rootShard *operatorv1alpha1.RootShard) *reconciler {
	return &reconciler{
		rootShard:      rootShard,
		resourceLabels: resources.GetRootShardProxyResourceLabels(rootShard),
	}
}

// getCABundleSecretRef returns the CABundleSecretRef from either the FrontProxy or RootShard spec.
func (r *reconciler) getCABundleSecretRef() *corev1.LocalObjectReference {
	if r.frontProxy != nil {
		return r.frontProxy.Spec.CABundleSecretRef
	}
	return r.rootShard.Spec.CABundleSecretRef
}

// +kubebuilder:rbac:groups=core,resources=configmaps;secrets;services,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;update;patch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;update;patch

func (r *reconciler) Reconcile(ctx context.Context, client ctrlruntimeclient.Client, namespace string) error {
	var errs []error

	var ref *metav1.OwnerReference
	if r.frontProxy != nil {
		ref = metav1.NewControllerRef(r.frontProxy, operatorv1alpha1.SchemeGroupVersion.WithKind("FrontProxy"))
	} else {
		ref = metav1.NewControllerRef(r.rootShard, operatorv1alpha1.SchemeGroupVersion.WithKind("RootShard"))
	}
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*ref)

	configMapReconcilers := []k8creconciling.NamedConfigMapReconcilerFactory{
		r.pathMappingConfigMapReconciler(),
	}

	secretReconcilers := []k8creconciling.NamedSecretReconcilerFactory{
		r.dynamicKubeconfigSecretReconciler(),
	}

	if r.getCABundleSecretRef() != nil {
		secretReconcilers = append(secretReconcilers, r.mergedCABundleSecretReconciler(ctx, client))
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		r.serverCertificateReconciler(),
		r.kubeconfigCertificateReconciler(),
		r.requestHeaderCertificateReconciler(),
	}

	if r.frontProxy != nil {
		certReconcilers = append(certReconcilers, r.adminKubeconfigCertificateReconciler())
	}

	deploymentReconcilers := []k8creconciling.NamedDeploymentReconcilerFactory{
		r.deploymentReconciler(),
	}

	serviceReconcilers := []k8creconciling.NamedServiceReconcilerFactory{
		r.serviceReconciler(),
	}

	if err := k8creconciling.ReconcileConfigMaps(ctx, configMapReconcilers, namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileSecrets(ctx, secretReconcilers, namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileDeployments(ctx, deploymentReconcilers, namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	if err := k8creconciling.ReconcileServices(ctx, serviceReconcilers, namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}
