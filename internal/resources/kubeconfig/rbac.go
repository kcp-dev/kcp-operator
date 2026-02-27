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

package kubeconfig

import (
	"fmt"

	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/kcp-dev/kcp-operator/internal/kubernetes"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func OwnerLabels(owner *operatorv1alpha1.Kubeconfig) map[string]string {
	return map[string]string{
		"operator.kcp.io/kubeconfig": string(owner.UID),
	}
}

func KubeconfigGroup(kc *operatorv1alpha1.Kubeconfig) string {
	return fmt.Sprintf("kubeconfig:%s", kc.Name)
}

func ClusterRoleBindingReconciler(owner *operatorv1alpha1.Kubeconfig, clusterRole string, subject rbacv1.Subject) reconciling.NamedClusterRoleBindingReconcilerFactory {
	name := fmt.Sprintf("%s:%s", owner.UID, clusterRole)

	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			kubernetes.EnsureLabels(crb, OwnerLabels(owner))

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRole,
			}

			crb.Subjects = []rbacv1.Subject{subject}

			return crb, nil
		}
	}
}
