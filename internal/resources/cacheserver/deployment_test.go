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

package cacheserver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetArgsEtcdPrefix(t *testing.T) {
	tests := []struct {
		name   string
		etcd   *operatorv1alpha1.EtcdConfig
		expect func(*testing.T, []string)
	}{
		{
			name: "dedicated etcd wires the prefix when set",
			etcd: &operatorv1alpha1.EtcdConfig{
				Endpoints: []string{"https://etcd:2379"},
				Prefix:    "/custom-prefix",
			},
			expect: func(t *testing.T, args []string) {
				assert.Contains(t, args, "--etcd-prefix=/custom-prefix")
			},
		},
		{
			name: "dedicated etcd omits the prefix when unset",
			etcd: &operatorv1alpha1.EtcdConfig{
				Endpoints: []string{"https://etcd:2379"},
			},
			expect: func(t *testing.T, args []string) {
				for _, arg := range args {
					assert.NotContains(t, arg, "--etcd-prefix")
				}
			},
		},
		{
			name: "embedded etcd never wires a prefix",
			etcd: nil,
			expect: func(t *testing.T, args []string) {
				for _, arg := range args {
					assert.NotContains(t, arg, "--etcd-prefix")
					assert.NotContains(t, arg, "--etcd-servers")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &operatorv1alpha1.CacheServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cachey"},
				Spec:       operatorv1alpha1.CacheServerSpec{Etcd: tt.etcd},
			}

			tt.expect(t, getArgs(server, nil))
		})
	}
}
