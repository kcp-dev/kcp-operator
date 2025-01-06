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

package rootshard

import (
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/api/v1alpha1"
)

func ServiceReconciler(rootShard *v1alpha1.RootShard) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return fmt.Sprintf("%s-kcp", rootShard.Name), func(svc *corev1.Service) (*corev1.Service, error) {
			svc.SetLabels(rootShard.GetResourceLabels())
			svc.Spec.Type = corev1.ServiceTypeClusterIP
			svc.Spec.Ports = []corev1.ServicePort{
				{
					Name:        "https",
					Protocol:    corev1.ProtocolTCP,
					Port:        6443,
					TargetPort:  intstr.FromInt32(6443),
					AppProtocol: ptr.To("https"),
				},
				{
					Name:        "https-virtual-workspaces",
					Protocol:    corev1.ProtocolTCP,
					Port:        6444,
					TargetPort:  intstr.FromInt32(6444),
					AppProtocol: ptr.To("https"),
				},
			}
			svc.Spec.Selector = rootShard.GetResourceLabels()

			return svc, nil
		}
	}
}
