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
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func RootCAIssuerReconciler(server *operatorv1alpha1.CacheServer) reconciling.NamedIssuerReconcilerFactory {
	name := resources.GetCacheServerCAName(server.Name, operatorv1alpha1.RootCA)

	secretName := name
	if server.Spec.Certificates.CASecretRef != nil {
		secretName = server.Spec.Certificates.CASecretRef.Name
	}

	return func() (string, reconciling.IssuerReconciler) {
		return name, func(issuer *certmanagerv1.Issuer) (*certmanagerv1.Issuer, error) {
			issuer.SetLabels(resources.GetCacheServerResourceLabels(server))
			issuer.Spec = certmanagerv1.IssuerSpec{
				IssuerConfig: certmanagerv1.IssuerConfig{
					CA: &certmanagerv1.CAIssuer{
						SecretName: secretName,
					},
				},
			}

			return issuer, nil
		}
	}
}
