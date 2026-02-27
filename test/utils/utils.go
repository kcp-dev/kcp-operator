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

package utils

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kcp-dev/logicalcluster/v3"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	kcptenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/sdk/testing/server"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func NewScheme(t *testing.T) *runtime.Scheme {
	sc := runtime.NewScheme()
	if err := scheme.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := kcpcorev1alpha1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}
	if err := kcptenancyv1alpha1.AddToScheme(sc); err != nil {
		t.Fatal(err)
	}

	return sc
}

func GetSelfSignedIssuerRef() *operatorv1alpha1.ObjectReference {
	return &operatorv1alpha1.ObjectReference{
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
	if err := operatorv1alpha1.AddToScheme(sc); err != nil {
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

func CreateSelfDestructingNamespace(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, name string) *corev1.Namespace {
	t.Helper()

	ns := corev1.Namespace{}
	ns.Name = fmt.Sprintf("e2e-%s", name)

	t.Logf("Creating namespace %s...", name)
	if err := client.Create(ctx, &ns); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		t.Logf("Deleting namespace %s...", name)
		if err := client.Delete(ctx, &ns); err != nil {
			t.Fatal(err)
		}
	})

	return &ns
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

	t.Logf("Exposing %s:%d on port %d...", target, targetPort, localPort)

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

func getPort(t *testing.T) int {
	port, err := server.GetFreePort(t)
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("Failed to parse port %q as number: %v", port, err)
	}

	return portNum
}

func ConnectWithKubeconfig(
	t *testing.T,
	ctx context.Context,
	client ctrlruntimeclient.Client,
	namespace string,
	kubeconfigName string,
	cluster logicalcluster.Path,
) ctrlruntimeclient.Client {
	t.Helper()

	// get kubeconfig
	config := &operatorv1alpha1.Kubeconfig{}
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
	localPort := getPort(t)
	SelfDestuctingPortForward(t, ctx, namespace, "svc/"+serviceName, int(portNum), localPort)

	// patch the target server
	parsed.Host = net.JoinHostPort("localhost", fmt.Sprintf("%d", localPort))
	clientConfig.Host = parsed.String()

	// switch to another workspace is desired
	if !cluster.Empty() {
		clientConfig.Host = changeClusterInURL(clientConfig.Host, cluster)
	}

	// create a client through the tunnel
	kcpClient, err := ctrlruntimeclient.New(clientConfig, ctrlruntimeclient.Options{Scheme: NewScheme(t)})
	if err != nil {
		t.Fatal(err)
	}

	return kcpClient
}

func ConnectWithRootShardProxy(
	t *testing.T,
	ctx context.Context,
	client ctrlruntimeclient.Client,
	rootShard *operatorv1alpha1.RootShard,
	cluster logicalcluster.Path,
) ctrlruntimeclient.Client {
	t.Helper()

	// get the secret for the kcp-operator client cert
	key := types.NamespacedName{
		Namespace: rootShard.Namespace,
		Name:      resources.GetRootShardCertificateName(rootShard, operatorv1alpha1.OperatorCertificate),
	}

	certSecret := &corev1.Secret{}
	if err := client.Get(ctx, key, certSecret); err != nil {
		t.Fatalf("Failed to get root shard proxy Secret: %v", err)
	}

	// start a port forwarding
	localPort := getPort(t)
	servicePort := 6443
	serviceName := resources.GetRootShardProxyServiceName(rootShard)

	SelfDestuctingPortForward(t, ctx, rootShard.Namespace, "svc/"+serviceName, servicePort, localPort)

	// create rest config
	proxyUrl := fmt.Sprintf("https://%s", net.JoinHostPort("localhost", fmt.Sprintf("%d", localPort)))

	if !cluster.Empty() {
		proxyUrl = changeClusterInURL(proxyUrl, cluster)
	}

	cfg := &rest.Config{
		Host: proxyUrl,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   certSecret.Data["ca.crt"],
			CertData: certSecret.Data["tls.crt"],
			KeyData:  certSecret.Data["tls.key"],
		},
	}

	// create a client through the tunnel
	kcpClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: NewScheme(t)})
	if err != nil {
		t.Fatal(err)
	}

	return kcpClient
}

var clusterRegexp = regexp.MustCompile(`/clusters/([^/]+)`)

func changeClusterInURL(u string, newCluster logicalcluster.Path) string {
	newPath := fmt.Sprintf("/clusters/%s", newCluster)

	matches := clusterRegexp.FindAllString(u, 1)
	if len(matches) == 0 {
		return u + newPath
	}

	// make sure that if a URL is "/clusters/root/apis/example.com/v1/namespaces/bla/clusters/mycluster",
	// we only replace the first match, especially important if the URL was "/clusters/X/apis/example.com/v1/clusters/X"
	// (i.e. accessing the cluster resource X in the kcp cluster also called X)
	return strings.Replace(u, matches[0], newPath, 1)
}
