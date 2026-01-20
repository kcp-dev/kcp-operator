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

func NewRootShardClient(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	if rootShard == nil {
		panic("No rootShard provided.")
	}

	baseUrl := fmt.Sprintf("https://%s.%s.svc.cluster.local:6443", resources.GetRootShardServiceName(rootShard), rootShard.Namespace)

	if !cluster.Empty() {
		baseUrl = fmt.Sprintf("%s/clusters/%s", baseUrl, cluster.String())
	}

	return newClient(ctx, c, baseUrl, scheme, rootShard, nil, nil)
}

func NewRootShardProxyClient(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	if rootShard == nil {
		panic("No rootShard provided.")
	}

	baseUrl := fmt.Sprintf("https://%s.%s.svc.cluster.local:6443", resources.GetRootShardProxyServiceName(rootShard), rootShard.Namespace)

	if !cluster.Empty() {
		baseUrl = fmt.Sprintf("%s/clusters/%s", baseUrl, cluster.String())
	}

	return newClient(ctx, c, baseUrl, scheme, rootShard, nil, nil)
}

func NewShardClient(ctx context.Context, c ctrlruntimeclient.Client, shard *operatorv1alpha1.Shard, cluster logicalcluster.Name, scheme *runtime.Scheme) (ctrlruntimeclient.Client, error) {
	if shard == nil {
		panic("No shard provided.")
	}

	baseUrl := fmt.Sprintf("https://%s.%s.svc.cluster.local:6443", resources.GetShardServiceName(shard), shard.Namespace)

	if !cluster.Empty() {
		baseUrl = fmt.Sprintf("%s/clusters/%s", baseUrl, cluster.String())
	}

	return newClient(ctx, c, baseUrl, scheme, nil, shard, nil)
}

func newClient(
	ctx context.Context,
	c ctrlruntimeclient.Client,
	url string,
	scheme *runtime.Scheme,
	// only one of these three should be provided, the others nil
	rootShard *operatorv1alpha1.RootShard,
	shard *operatorv1alpha1.Shard,
	frontProxy *operatorv1alpha1.FrontProxy,
) (ctrlruntimeclient.Client, error) {
	tlsConfig, err := getTLSConfig(ctx, c, rootShard, shard, frontProxy)
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

func getTLSConfig(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard, frontProxy *operatorv1alpha1.FrontProxy) (rest.TLSClientConfig, error) {
	rootShard, err := getRootShard(ctx, c, rootShard, shard, frontProxy)
	if err != nil {
		return rest.TLSClientConfig{}, fmt.Errorf("failed to determine effective RootShard: %w", err)
	}

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

func getRootShard(ctx context.Context, c ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, shard *operatorv1alpha1.Shard, frontProxy *operatorv1alpha1.FrontProxy) (*operatorv1alpha1.RootShard, error) {
	if rootShard != nil {
		return rootShard, nil
	}

	var ref *corev1.LocalObjectReference

	switch {
	case shard != nil:
		ref = shard.Spec.RootShard.Reference

	case frontProxy != nil:
		ref = frontProxy.Spec.RootShard.Reference

	default:
		panic("Must be called with either RootShard, Shard or FrontProxy.")
	}

	rootShard = &operatorv1alpha1.RootShard{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: rootShard.Namespace, Name: ref.Name}, rootShard); err != nil {
		return nil, fmt.Errorf("failed to get RootShard: %w", err)
	}

	return rootShard, nil
}
