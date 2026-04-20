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

package resources

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	corev1 "k8s.io/api/core/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	// FrontProxyCommonName is the CommonName used in the requestheader client certificate for a FrontProxy.
	FrontProxyCommonName = "kcp-front-proxy"

	// RootShardProxyCommonName is the CommonName used in the requestheader client certificate for a RootShard's built-in proxy.
	RootShardProxyCommonName = "kcp-root-shard-proxy"

	ImageRepository = "ghcr.io/kcp-dev/kcp"

	// ImageTag is the default tag to be used for any kcp component.
	//
	// When changing this to a new minor version, you must also update
	// the .prow.yaml accordingly and shift the jobs.
	ImageTag = "v0.31.0"

	// RootShardLabel is placed on Secrets created for Certificates so that
	// the Secrets can be more easily mapped to their RootShards.
	RootShardLabel        = "operator.kcp.io/rootshard"
	ShardLabel            = "operator.kcp.io/shard"
	FrontProxyLabel       = "operator.kcp.io/front-proxy"
	KubeconfigLabel       = "operator.kcp.io/kubeconfig"
	CacheServerLabel      = "operator.kcp.io/cache-server"
	VirtualWorkspaceLabel = "operator.kcp.io/virtual-workspace"

	// KubeconfigUIDLabel is used inside kcp workspaces to link RBAC resources
	// to their owning Kubeconfig object on the kcp operator cluster.
	KubeconfigUIDLabel = "operator.kcp.io/kubeconfig"

	// BundleAnnotation is placed on RootShard, Shard, or FrontProxy objects to trigger automatic Bundle creation
	BundleAnnotation = "operator.kcp.io/bundle"

	// BundleDesiredReplicasAnnotation is placed on deployments to store the desired replica count before scaling to 0 for bundling
	BundleDesiredReplicasAnnotation = "operator.kcp.io/bundle-desired-replicas"

	// OperatorUsername is the common name embedded in the operator's admin certificate
	// that is created for each RootShard. This name alone has no special meaning, as
	// the certificate also has system:masters as an organization, which is what ultimately
	// grants the operator its permissions.
	OperatorUsername = "system:kcp-operator"
)

func GetImageSettings(imageSpec *operatorv1alpha1.ImageSpec) (string, []corev1.LocalObjectReference, *semver.Version) {
	repository := ImageRepository
	if imageSpec != nil && imageSpec.Repository != "" {
		repository = imageSpec.Repository
	}

	tag := ImageTag
	if imageSpec != nil && imageSpec.Tag != "" {
		tag = imageSpec.Tag
	}

	imagePullSecrets := []corev1.LocalObjectReference{}
	if imageSpec != nil && len(imageSpec.ImagePullSecrets) > 0 {
		imagePullSecrets = imageSpec.ImagePullSecrets
	}

	// try to detect the kcp version, but accept that this might not work for custom image tags
	version, _ := semver.NewVersion(tag)

	return fmt.Sprintf("%s:%s", repository, tag), imagePullSecrets, version
}
