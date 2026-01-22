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

package client

import (
	"context"
	"fmt"

	"github.com/kcp-dev/logicalcluster/v3"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// NewRootShardClient returns a new client for talking to the kcp root shard service directly.
func NewRootShardClient(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	baseUrl := fmt.Sprintf("https://%s.%s.svc.cluster.local:6443", resources.GetRootShardServiceName(rootShard), rootShard.Namespace)

	if !cluster.Empty() {
		baseUrl = fmt.Sprintf("%s/clusters/%s", baseUrl, cluster.String())
	}

	return newClient(ctx, c, baseUrl, scheme, rootShard)
}

// NewRootShardClient returns a new client that connects to the operator's internal front-proxy.
func NewRootShardProxyClient(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	baseUrl := fmt.Sprintf("https://%s.%s.svc.cluster.local:6443", resources.GetRootShardProxyServiceName(rootShard), rootShard.Namespace)

	if !cluster.Empty() {
		baseUrl = fmt.Sprintf("%s/clusters/%s", baseUrl, cluster.String())
	}

	return newClient(ctx, c, baseUrl, scheme, rootShard)
}

// NewShardClient returns a new client for talking to a kcp shard service directly.
func NewShardClient(ctx context.Context, c ctrlruntimeclient.Client, shard *operatorv1alpha1.Shard, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	rootShard, err := getRootShardForShard(ctx, c, shard)
	if err != nil {
		return nil, fmt.Errorf("failed to determine effective RootShard: %w", err)
	}

	baseUrl := fmt.Sprintf("https://%s.%s.svc.cluster.local:6443", resources.GetShardServiceName(shard), shard.Namespace)

	if !cluster.Empty() {
		baseUrl = fmt.Sprintf("%s/clusters/%s", baseUrl, cluster.String())
	}

	return newClient(ctx, c, baseUrl, scheme, rootShard)
}

func newClient(
	ctx context.Context,
	c ctrlruntimeclient.Client,
	url string,
	scheme *runtime.Scheme,
	rootShard *operatorv1alpha1.RootShard,
) (ctrlruntimeclient.Client, error) {
	tlsConfig, err := getTLSConfig(ctx, c, rootShard)
	if err != nil {
		return nil, fmt.Errorf("failed to determine TLS settings: %w", err)
	}

	cfg := &rest.Config{
		Host:            url,
		TLSClientConfig: tlsConfig,
	}

	return ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme})
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get

// getTLSConfig returns the CA and serving certificate for a RootShard.
func getTLSConfig(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard) (rest.TLSClientConfig, error) {
	// get the secret for the kcp-operator client cert
	key := types.NamespacedName{
		Namespace: rootShard.Namespace,
		Name:      resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.OperatorCertificate),
	}

	certSecret := &corev1.Secret{}
	if err := c.Get(ctx, key, certSecret); err != nil {
		return rest.TLSClientConfig{}, fmt.Errorf("failed to get root shard proxy Secret: %w", err)
	}

	return rest.TLSClientConfig{
		CAData:   certSecret.Data["ca.crt"],
		CertData: certSecret.Data["tls.crt"],
		KeyData:  certSecret.Data["tls.key"],
	}, nil
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get

func getRootShardForShard(ctx context.Context, c ctrlruntimeclient.Client, shard *operatorv1alpha1.Shard) (*operatorv1alpha1.RootShard, error) {
	rootShard := &operatorv1alpha1.RootShard{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: shard.Namespace, Name: shard.Spec.RootShard.Reference.Name}, rootShard); err != nil {
		return nil, fmt.Errorf("failed to get RootShard: %w", err)
	}

	return rootShard, nil
}
