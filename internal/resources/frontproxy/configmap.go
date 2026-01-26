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
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func (r *reconciler) pathMappingConfigMapName() string {
	if r.frontProxy != nil {
		return resources.GetFrontProxyConfigName(r.frontProxy)
	} else {
		return resources.GetRootShardProxyConfigName(r.rootShard)
	}
}

func (r *reconciler) pathMappingConfigMapReconciler() reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return r.pathMappingConfigMapName(), func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			cm.SetLabels(r.resourceLabels)

			mappings := r.defaultPathMappings()
			if r.frontProxy != nil {
				mappings = append(mappings, r.frontProxy.Spec.AdditionalPathMappings...)
			}

			d, err := yaml.Marshal(mappings)
			if err != nil {
				return nil, err
			}
			cm.Data = map[string]string{
				"path-mapping.yaml": string(d),
			}

			return cm, nil
		}
	}
}

// defaultPathMappings sets up default paths for a front-proxy
func (r *reconciler) defaultPathMappings() []operatorv1alpha1.PathMappingEntry {
	url := resources.GetRootShardBaseURL(r.rootShard)

	// Determine CA path based on CABundleSecretRef
	backendCA := kcpBasepath + "/tls/ca/tls.crt"
	if r.getCABundleSecretRef() != nil {
		backendCA = getCAMountPath(operatorv1alpha1.CABundleCA) + "/tls.crt"
	}

	return []operatorv1alpha1.PathMappingEntry{
		{
			Path:            "/clusters/",
			Backend:         url,
			BackendServerCA: backendCA,
			ProxyClientCert: "/etc/kcp-front-proxy/requestheader-client/tls.crt",
			ProxyClientKey:  "/etc/kcp-front-proxy/requestheader-client/tls.key",
		},
		{
			Path:            "/services/",
			Backend:         url,
			BackendServerCA: backendCA,
			ProxyClientCert: "/etc/kcp-front-proxy/requestheader-client/tls.crt",
			ProxyClientKey:  "/etc/kcp-front-proxy/requestheader-client/tls.key",
		},
	}
}
