/*
Copyright 2024 The kcp Authors.

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

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func GetTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(metav1.AddMetaToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(deployv1alpha1.AddToScheme(scheme))
	utilruntime.Must(certmanagerv1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))

	return scheme
}

// MarkCertificatesReady sets all Ready=true certificates in the given namespace.
func MarkCertificatesReady(ctx context.Context, client ctrlruntimeclient.Client, namespace string) error {
	certs := &certmanagerv1.CertificateList{}
	if err := client.List(ctx, certs, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return err
	}

	for i := range certs.Items {
		cert := &certs.Items[i]
		cert.Status.Revision = ptr.To(1)
		cert.Status.Conditions = []certmanagerv1.CertificateCondition{{
			Type:   certmanagerv1.CertificateConditionReady,
			Status: certmanagermetav1.ConditionTrue,
		}}
		if err := client.Update(ctx, cert); err != nil {
			return err
		}
	}

	return nil
}
