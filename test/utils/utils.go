/*
Copyright 2024 The KCP Authors.

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

package utils

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	kcpoperatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetSelfSignedIssuerRef() *kcpoperatorv1alpha1.ObjectReference {
	return &kcpoperatorv1alpha1.ObjectReference{
		Group: "cert-manager.io",
		Kind:  "ClusterIssuer",
		Name:  "selfsigned",
	}
}

func GetKubeClient(t *testing.T) ctrlruntimeclient.Client {
	t.Helper()

	sc := runtime.NewScheme()
	if err := scheme.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := kcpoperatorv1alpha1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}

	config, err := ctrlruntime.GetConfig()
	if err != nil {
		t.Fatalf("Failed to get kubeconfig: %v", err)
	}

	c, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{
		Scheme: sc,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return c
}

func CreateSelfDestructingNamespace(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, name string) {
	t.Helper()

	ns := corev1.Namespace{}
	ns.Name = name

	t.Logf("Creating namespace %s…", name)
	if err := client.Create(ctx, &ns); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		t.Logf("Deleting namespace %s…", name)
		if err := client.Delete(ctx, &ns); err != nil {
			t.Fatal(err)
		}
	})
}

func SelfDestuctingPortForward(
	t *testing.T,
	ctx context.Context,
	namespace string,
	target string,
	targetPort int,
	localPort int,
) {
	t.Helper()

	args := []string{
		"port-forward",
		"--namespace", namespace,
		target,
		fmt.Sprintf("%d:%d", localPort, targetPort),
	}

	t.Logf("Exposing %s:%d on port %d…", target, targetPort, localPort)

	localCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(localCtx, "kubectl", args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start port-forwarding: %v", err)
	}

	time.Sleep(3 * time.Second)

	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
	})
}

var currentPort = 56029

func getPort() int {
	port := currentPort
	currentPort++

	return port
}

func ConnectWithKubeconfig(
	t *testing.T,
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespace string,
	kubeconfigName string,
) ctrlruntimeclient.Client {
	t.Helper()

	// get kubeconfig
	config := &kcpoperatorv1alpha1.Kubeconfig{}
	key := types.NamespacedName{Namespace: namespace, Name: kubeconfigName}
	if err := client.Get(ctx, key, config); err != nil {
		t.Fatal(err)
	}

	// get the kubeconfig's secret
	secret := &corev1.Secret{}
	key = types.NamespacedName{Namespace: namespace, Name: config.Spec.SecretRef.Name}
	if err := client.Get(ctx, key, secret); err != nil {
		t.Fatal(err)
	}

	// parse kubeconfig
	clientConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["kubeconfig"])
	if err != nil {
		t.Fatalf("Failed to parse kubeconfig: %v", err)
	}

	// deduce service name from the hostname
	parsed, err := url.Parse(clientConfig.Host)
	if err != nil {
		t.Fatalf("Failed to parse kubeconfig's server %q: %v", clientConfig.Host, err)
	}

	hostname, portString, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("Failed to parse kubeconfig's host %q: %v", parsed.Host, err)
	}

	parts := strings.Split(hostname, ".")
	serviceName := parts[0]

	portNum, err := strconv.ParseInt(portString, 10, 32)
	if err != nil {
		t.Fatalf("Failed to parse kubeconfig's port %q: %v", portString, err)
	}

	// start a port forwarding
	localPort := getPort()
	SelfDestuctingPortForward(t, ctx, namespace, "svc/"+serviceName, int(portNum), localPort)

	// patch the target server
	parsed.Host = net.JoinHostPort("localhost", fmt.Sprintf("%d", localPort))
	clientConfig.Host = parsed.String()

	// create a client through the tunnel
	sc := runtime.NewScheme()
	if err := scheme.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}

	kcpClient, err := ctrlruntimeclient.New(clientConfig, ctrlruntimeclient.Options{Scheme: sc})
	if err != nil {
		t.Fatal(err)
	}

	return kcpClient
}

func DeployEtcd(t *testing.T, namespace string) string {
	t.Helper()

	t.Logf("Installing etcd into %s…", namespace)
	args := []string{
		"install",
		"etcd",
		"oci://registry-1.docker.io/bitnamicharts/etcd",
		"--namespace", namespace,
		"--version", "10.7.1", // latest version at the time of writing
		"--set", "auth.rbac.enabled=false",
		"--set", "auth.rbac.create=false",
	}

	if err := exec.Command("helm", args...).Run(); err != nil {
		t.Fatalf("Failed to deploy etcd: %v", err)
	}

	t.Log("Waiting for etcd to get ready…")
	args = []string{
		"wait",
		"pods",
		"--namespace", namespace,
		"--selector", "app.kubernetes.io/name=etcd",
		"--for", "condition=Ready",
		"--timeout", "3m",
	}

	if err := exec.Command("kubectl", args...).Run(); err != nil {
		t.Fatalf("Failed to wait for etcd to become ready: %v", err)
	}

	return fmt.Sprintf("http://etcd.%s.svc.cluster.local:2379", namespace)
}
