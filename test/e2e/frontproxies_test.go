//go:build e2e

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

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

func TestCreateFrontProxy(t *testing.T) {
	t.Parallel()
	fmt.Println()

	ctrlruntime.SetLogger(logr.Discard())

	client := utils.GetKubeClient(t)
	ctx := context.Background()

	namespace := utils.CreateSelfDestructingNamespace(t, ctx, client, "create-frontproxy")

	externalHostname := "front-proxy-front-proxy.svc.cluster.local"

	// deploy rootshard
	rootShard := utils.DeployRootShard(ctx, t, client, namespace.Name, externalHostname)

	// deploy front-proxy
	frontProxy := utils.DeployFrontProxy(ctx, t, client, namespace.Name, rootShard.Name, externalHostname)

	// create front-proxy kubeconfig
	configSecretName := "kubeconfig-front-proxy-e2e"

	fpConfig := operatorv1alpha1.Kubeconfig{}
	fpConfig.Name = "front-proxy"
	fpConfig.Namespace = namespace.Name
	fpConfig.Spec = operatorv1alpha1.KubeconfigSpec{
		Target: operatorv1alpha1.KubeconfigTarget{
			FrontProxyRef: &corev1.LocalObjectReference{
				Name: frontProxy.Name,
			},
		},
		Username: "e2e",
		Validity: metav1.Duration{Duration: 2 * time.Hour},
		SecretRef: corev1.LocalObjectReference{
			Name: configSecretName,
		},
		Groups: []string{"system:masters"},
	}

	t.Log("Creating kubeconfig for FrontProxy...")
	if err := client.Create(ctx, &fpConfig); err != nil {
		t.Fatal(err)
	}
	utils.WaitForObject(t, ctx, client, &corev1.Secret{}, types.NamespacedName{Namespace: fpConfig.Namespace, Name: fpConfig.Spec.SecretRef.Name})

	// verify that we can use frontproxy kubeconfig to access rootshard workspaces
	t.Log("Connecting to FrontProxy...")
	kcpClient := utils.ConnectWithKubeconfig(t, ctx, client, namespace.Name, fpConfig.Name)
	// proof of life: list something every logicalcluster in kcp has
	t.Log("Should be able to list Secrets.")
	secrets := &corev1.SecretList{}
	if err := kcpClient.List(ctx, secrets); err != nil {
		t.Fatalf("Failed to list secrets in kcp: %v", err)
	}
}
