/*
Copyright 2025 The kcp Authors.

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
	"errors"
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/reconciling/modifier"
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

// getClientCABundleSecretRef returns the ClientCABundleRef from the FrontProxy spec.
// This is only used for FrontProxy resources, not for the RootShard internal proxy.
func (r *reconciler) getClientCABundleSecretRef() *corev1.LocalObjectReference {
	if r.frontProxy != nil {
		return r.frontProxy.Spec.ClientCABundleRef
	}
	return nil
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
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, client)

	// Fetch client CA certificates
	clientCACerts, err := r.fetchClientCACerts(ctx, client)
	if err != nil {
		return err
	}

	configMapReconcilers := []k8creconciling.NamedConfigMapReconcilerFactory{
		r.pathMappingConfigMapReconciler(),
	}

	secretReconcilers := []k8creconciling.NamedSecretReconcilerFactory{
		r.dynamicKubeconfigSecretReconciler(),
		r.clientCABundleSecretReconciler(clientCACerts...),
	}

	// Fetch server CA bundle if needed
	if r.getCABundleSecretRef() != nil {
		serverCACert, userCABundle, err := r.fetchBackendCAs(ctx, client)
		if err != nil {
			return err
		}
		secretReconcilers = append(secretReconcilers, r.backendCABundleSecretReconciler(serverCACert, userCABundle))
	}

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		r.serverCertificateReconciler(),
		r.kubeconfigCertificateReconciler(),
		r.requestHeaderCertificateReconciler(),
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

	// must happen after the Secrets and Certificates have been reconciled, since it can fail as long as those do not exist
	if err := k8creconciling.ReconcileDeployments(ctx, deploymentReconcilers, namespace, client, ownerRefWrapper, revisionLabels); err != nil {
		// swallow errors and rely on the caller watching Secrets and re-reconciling whenever they change
		if !errors.Is(err, modifier.ErrMountNotFound) {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileServices(ctx, serviceReconcilers, namespace, client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}

// fetchClientCACerts fetches the ClientCA certificate and optionally the additional
// client CA bundles (from RootShard and/or FrontProxy if configured).
// Returns the certificates in order: ClientCA, RootShard.ClientCABundleRef, FrontProxy.ClientCABundleRef
func (r *reconciler) fetchClientCACerts(ctx context.Context, client ctrlruntimeclient.Client) ([][]byte, error) {
	certs := [][]byte{}

	// fetch the shared, global client CA
	clientCA, err := r.fetchTLSCert(ctx, client, resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.ClientCA))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ClientCA certificate: %w", err)
	}
	certs = append(certs, clientCA)

	// fetch RootShard's optional client CA bundle (inherited by all components)
	if r.rootShard.Spec.ClientCABundleRef != nil {
		rootShardCABundle, err := r.fetchTLSCert(ctx, client, r.rootShard.Spec.ClientCABundleRef.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch RootShard client CA bundle: %w", err)
		}
		certs = append(certs, rootShardCABundle)
	}

	// fetch optional additional client CA bundle if specified on FrontProxy
	// (if this a root proxy, getClientCABundleSecretRef returns nil)
	if ref := r.getClientCABundleSecretRef(); ref != nil {
		additionalCABundle, err := r.fetchTLSCert(ctx, client, ref.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch additional client CA bundle: %w", err)
		}
		certs = append(certs, additionalCABundle)
	}

	return certs, nil
}

// fetchBackendCAs fetches the ServerCA certificate and the user-provided CA bundle.
func (r *reconciler) fetchBackendCAs(ctx context.Context, client ctrlruntimeclient.Client) (serverCA, userCABundle []byte, err error) {
	// fetch ServerCA
	serverCA, err = r.fetchTLSCert(ctx, client, resources.GetRootShardCAName(r.rootShard, operatorv1alpha1.ServerCA))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch ServerCA certificate: %w", err)
	}

	// fetch user-provided CA bundle
	if ref := r.getCABundleSecretRef(); ref != nil {
		userCABundle, err = r.fetchTLSCert(ctx, client, ref.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch user CA bundle: %w", err)
		}
	}

	return serverCA, userCABundle, nil
}

func (r *reconciler) fetchTLSCert(ctx context.Context, client ctrlruntimeclient.Client, secretName string) ([]byte, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: r.rootShard.Namespace}, secret); err != nil {
		return nil, err
	}

	data, exists := secret.Data["tls.crt"]
	if !exists {
		return nil, fmt.Errorf("the Secret %s contains no tls.crt", secretName)
	}

	return data, nil
}
