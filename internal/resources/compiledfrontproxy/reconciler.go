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

package compiledfrontproxy

import (
	"context"
	"errors"
	"fmt"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/pkg/reconciling/modifier"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

type reconciler struct {
	frontProxy     *deployv1alpha1.CompiledFrontProxy
	rootShard      *deployv1alpha1.CompiledRootShard
	resourceLabels map[string]string
}

func NewFrontProxy(frontProxy *deployv1alpha1.CompiledFrontProxy) *reconciler {
	if frontProxy == nil {
		panic("Use NewRootShardProxy instead.")
	}

	return &reconciler{
		frontProxy:     frontProxy,
		resourceLabels: resources.GetCompiledFrontProxyResourceLabels(frontProxy),
	}
}

func NewRootShardProxy(rootShard *deployv1alpha1.CompiledRootShard) *reconciler {
	return &reconciler{
		rootShard:      rootShard,
		resourceLabels: resources.GetCompiledRootShardProxyResourceLabels(rootShard),
	}
}

// rootShardName returns the name of the RootShard this proxy fronts, in either mode.
func (r *reconciler) rootShardName() string {
	if r.frontProxy != nil {
		return r.frontProxy.Spec.RootShard.Name
	}
	return r.rootShard.Name
}

// rootShardSpec returns the resolved RootShard spec, in either mode.
func (r *reconciler) rootShardSpec() operatorv1alpha1.RootShardSpec {
	if r.frontProxy != nil {
		return r.frontProxy.Spec.RootShard.Spec
	}
	return r.rootShard.Spec.RootShard
}

// rootShardBaseURL returns the in-cluster URL of the RootShard this proxy fronts.
func (r *reconciler) rootShardBaseURL() string {
	if r.frontProxy != nil {
		return resources.GetNamedRootShardBaseURL(r.frontProxy.Spec.RootShard, r.frontProxy.Namespace)
	}
	return resources.GetCompiledRootShardBaseURL(r.rootShard)
}

// rootShardCAName returns the name of one of the RootShard's CAs, in either mode.
func (r *reconciler) rootShardCAName(caName operatorv1alpha1.CA) string {
	if r.frontProxy != nil {
		return resources.GetNamedRootShardCAName(r.frontProxy.Spec.RootShard, caName)
	}
	return resources.GetCompiledRootShardCAName(r.rootShard, caName)
}

// certName returns the name of one of the proxy's own certificates.
func (r *reconciler) certName(certKind operatorv1alpha1.Certificate) string {
	if r.frontProxy != nil {
		return resources.GetCompiledFrontProxyCertificateName(r.frontProxy, certKind)
	}
	return resources.GetCompiledRootShardProxyCertificateName(r.rootShard, certKind)
}

// certCommonName returns the CommonName the proxy presents to the shards it fronts.
func (r *reconciler) certCommonName() string {
	if r.frontProxy != nil {
		return resources.FrontProxyCommonName
	}
	return resources.RootShardProxyCommonName
}

// clientCABundleSecretName is the Secret holding the client CAs the proxy accepts.
func (r *reconciler) clientCABundleSecretName() string {
	if r.frontProxy != nil {
		return fmt.Sprintf("%s-merged-client-ca", r.frontProxy.Name)
	}
	return fmt.Sprintf("%s-proxy-merged-client-ca", r.rootShard.Name)
}

// backendCABundleSecretName is the Secret holding the CAs the proxy trusts for its backends.
func (r *reconciler) backendCABundleSecretName() string {
	if r.frontProxy != nil {
		return fmt.Sprintf("%s-merged-ca-bundle", r.frontProxy.Name)
	}
	return fmt.Sprintf("%s-proxy-merged-ca-bundle", r.rootShard.Name)
}

// getCABundleSecretRef returns the CABundleSecretRef from either the FrontProxy or RootShard spec.
func (r *reconciler) getCABundleSecretRef() *corev1.LocalObjectReference {
	if r.frontProxy != nil {
		return r.frontProxy.Spec.FrontProxy.CABundleSecretRef
	}
	return r.rootShardSpec().CABundleSecretRef
}

// +kubebuilder:rbac:groups=core,resources=configmaps;secrets;services,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;update;patch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;update;patch

func (r *reconciler) Reconcile(ctx context.Context, client ctrlruntimeclient.Client, namespace string) error {
	var errs []error

	var ref *metav1.OwnerReference
	if r.frontProxy != nil {
		ref = metav1.NewControllerRef(r.frontProxy, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledFrontProxy"))
	} else {
		ref = metav1.NewControllerRef(r.rootShard, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledRootShard"))
	}
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(*ref)
	revisionLabels := modifier.RelatedRevisionsLabels(ctx, client)

	configMapReconcilers := []k8creconciling.NamedConfigMapReconcilerFactory{
		r.pathMappingConfigMapReconciler(),
	}

	secretReconcilers := []k8creconciling.NamedSecretReconcilerFactory{
		r.dynamicKubeconfigSecretReconciler(),
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
