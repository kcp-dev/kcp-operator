/*
Copyright 2026 The kcp Authors.

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
	"slices"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/client-go/tools/clientcmd"

	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
)

var testPeers = []deployv1alpha1.ShardPeer{
	{Name: "r00t", URL: "https://r00t-kcp.ns.svc.cluster.local:6443"},
	{Name: "shard1", URL: "https://shard1-shard-kcp.ns.svc.cluster.local:6443"},
}

func TestPeersKubeconfig(t *testing.T) {
	rec := NewFrontProxy(&deployv1alpha1.CompiledFrontProxy{
		Spec: deployv1alpha1.CompiledFrontProxySpec{ShardPeers: testPeers},
	})

	data, err := rec.peersKubeconfig()
	require.NoError(t, err)

	config, err := clientcmd.Load(data)
	require.NoError(t, err)

	require.Len(t, config.Clusters, len(testPeers))
	for _, peer := range testPeers {
		cluster, ok := config.Clusters[peer.Name]
		require.True(t, ok, "missing cluster for peer %q", peer.Name)
		assert.Equal(t, peer.URL, cluster.Server)
		assert.Equal(t, kubeconfigCAPath, cluster.CertificateAuthority)
	}

	current := config.Contexts[config.CurrentContext]
	require.NotNil(t, current)
	assert.Equal(t, testPeers[0].Name, current.Cluster)

	authInfo := config.AuthInfos[current.AuthInfo]
	require.NotNil(t, authInfo)
	assert.Equal(t, clientCertificatePath, authInfo.ClientCertificate)
	assert.Equal(t, clientKeyPath, authInfo.ClientKey)
}

func TestShardPeerArgs(t *testing.T) {
	const flag = "--shard-peer-kubeconfig=/etc/kcp-front-proxy/peers-kubeconfig/kubeconfig"

	tests := []struct {
		name     string
		peers    []deployv1alpha1.ShardPeer
		version  *semver.Version
		expected bool
	}{
		{
			name:     "unknown version with peers",
			peers:    testPeers,
			expected: true,
		},
		{
			name:     "supported version with peers",
			peers:    testPeers,
			version:  semver.MustParse("0.34.0"),
			expected: true,
		},
		{
			name:     "prerelease of supported version",
			peers:    testPeers,
			version:  semver.MustParse("0.34.0-alpha.1"),
			expected: true,
		},
		{
			name:     "unsupported version",
			peers:    testPeers,
			version:  semver.MustParse("0.33.1"),
			expected: false,
		},
		{
			name:     "no peers",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frontProxy := NewFrontProxy(&deployv1alpha1.CompiledFrontProxy{
				Spec: deployv1alpha1.CompiledFrontProxySpec{ShardPeers: tt.peers},
			})
			assert.Equal(t, tt.expected, slices.Contains(frontProxy.getArgs(tt.version), flag))

			rootShardProxy := NewRootShardProxy(&deployv1alpha1.CompiledRootShard{
				Spec: deployv1alpha1.CompiledRootShardSpec{ShardPeers: tt.peers},
			})
			assert.Equal(t, tt.expected, slices.Contains(rootShardProxy.getArgs(tt.version), flag))
		})
	}
}
