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

package v1alpha1

import (
	"testing"

	"github.com/kcp-dev/logicalcluster/v3"
)

func TestGetTargetWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		kc       *Kubeconfig
		expected logicalcluster.Path
	}{
		{
			name: "targetWorkspace set",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{
					TargetWorkspace: "root:org:team",
				},
			},
			expected: logicalcluster.NewPath("root:org:team"),
		},
		{
			name: "deprecated cluster does NOT affect URL",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{
					Authorization: &KubeconfigAuthorization{
						ClusterRoleBindings: KubeconfigClusterRoleBindings{
							Cluster: "root:legacy:workspace",
						},
					},
				},
			},
			expected: logicalcluster.NewPath("root"),
		},
		{
			name: "both empty defaults to root",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{},
			},
			expected: logicalcluster.NewPath("root"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kc.GetTargetWorkspace()
			if !got.Equal(tt.expected) {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetRBACTargetWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		kc       *Kubeconfig
		expected logicalcluster.Path
	}{
		{
			name: "targetWorkspace set",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{
					TargetWorkspace: "root:org:team",
				},
			},
			expected: logicalcluster.NewPath("root:org:team"),
		},
		{
			name: "targetWorkspace empty, deprecated cluster set",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{
					Authorization: &KubeconfigAuthorization{
						ClusterRoleBindings: KubeconfigClusterRoleBindings{
							Cluster: "root:legacy:workspace",
						},
					},
				},
			},
			expected: logicalcluster.NewPath("root:legacy:workspace"),
		},
		{
			name: "both empty defaults to root",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{},
			},
			expected: logicalcluster.NewPath("root"),
		},
		{
			name: "authorization set without cluster defaults to root",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{
					Authorization: &KubeconfigAuthorization{
						ClusterRoleBindings: KubeconfigClusterRoleBindings{
							ClusterRoles: []string{"admin"},
						},
					},
				},
			},
			expected: logicalcluster.NewPath("root"),
		},
		{
			name: "targetWorkspace takes precedence over deprecated cluster",
			kc: &Kubeconfig{
				Spec: KubeconfigSpec{
					TargetWorkspace: "root:new",
					Authorization: &KubeconfigAuthorization{
						ClusterRoleBindings: KubeconfigClusterRoleBindings{
							Cluster: "root:old",
						},
					},
				},
			},
			expected: logicalcluster.NewPath("root:new"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kc.GetRBACTargetWorkspace()
			if !got.Equal(tt.expected) {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
