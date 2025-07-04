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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/utils"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func ServiceReconciler(frontProxy *operatorv1alpha1.FrontProxy) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.GetFrontProxyServiceName(frontProxy), func(svc *corev1.Service) (*corev1.Service, error) {
			labels := resources.GetFrontProxyResourceLabels(frontProxy)
			svc.SetLabels(labels)
			svc.Spec.Type = corev1.ServiceTypeClusterIP

			var port corev1.ServicePort
			if len(svc.Spec.Ports) == 1 {
				port = svc.Spec.Ports[0]
			}

			port.Name = "https"
			port.Protocol = corev1.ProtocolTCP
			port.Port = 6443
			port.TargetPort = intstr.FromInt32(6443)
			port.AppProtocol = ptr.To("https")

			svc.Spec.Ports = []corev1.ServicePort{
				port,
			}
			svc.Spec.Selector = labels

			return utils.ApplyServiceTemplate(svc, frontProxy.Spec.ServiceTemplate), nil
		}
	}
}
