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

package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetShardPeers(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, operatorv1alpha1.AddToScheme(scheme))

	vw := &operatorv1alpha1.VirtualWorkspace{
		ObjectMeta: metav1.ObjectMeta{Name: "shard2-vw", Namespace: "ns"},
		Spec: operatorv1alpha1.VirtualWorkspaceSpec{
			External: operatorv1alpha1.ExternalConfig{Hostname: "vw.example.com", Port: 8443},
		},
	}

	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{Name: "r00t", Namespace: "ns"},
		Spec: operatorv1alpha1.RootShardSpec{
			CommonShardSpec: operatorv1alpha1.CommonShardSpec{ShardBaseURL: "https://root.example.com:6443"},
		},
	}

	shards := []operatorv1alpha1.Shard{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "shard1", Namespace: "ns"},
			Spec: operatorv1alpha1.ShardSpec{
				CommonShardSpec: operatorv1alpha1.CommonShardSpec{ShardBaseURL: "https://shard1.example.com:6443"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "shard2", Namespace: "ns"},
			Spec: operatorv1alpha1.ShardSpec{
				CommonShardSpec: operatorv1alpha1.CommonShardSpec{
					ShardBaseURL:        "https://shard2.example.com:6443",
					KCPVirtualWorkspace: &corev1.LocalObjectReference{Name: vw.Name},
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(vw).Build()

	peers, err := GetShardPeers(context.Background(), client, rootShard, shards)
	require.NoError(t, err)

	assert.Equal(t, []deployv1alpha1.ShardPeer{
		{Name: "r00t", URL: "https://root.example.com:6443"},
		{Name: "shard1", URL: "https://shard1.example.com:6443"},
		// shards with an external VirtualWorkspace serve the Admin workspace there
		{Name: "shard2", URL: "https://vw.example.com:8443"},
	}, peers)
}

func TestGetShardPeersMissingVirtualWorkspace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, operatorv1alpha1.AddToScheme(scheme))

	rootShard := &operatorv1alpha1.RootShard{
		ObjectMeta: metav1.ObjectMeta{Name: "r00t", Namespace: "ns"},
		Spec: operatorv1alpha1.RootShardSpec{
			CommonShardSpec: operatorv1alpha1.CommonShardSpec{
				KCPVirtualWorkspace: &corev1.LocalObjectReference{Name: "missing"},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := GetShardPeers(context.Background(), client, rootShard, nil)
	require.Error(t, err)
}
