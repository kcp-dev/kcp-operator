//go:build e2e

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

package virtualworkspaces

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/kcp-dev/sdk/testing/server"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

const (
	// mockWorkspace is the workspace below root that the init container creates and the server
	// is then scoped to.
	mockWorkspace = "mock"
	// mockClusterRole is installed by the init container and bound to the serving identity.
	mockClusterRole = "mock-vw"
	mockProbeConfig = "mock-probe"
)

// selfCheck mirrors the report the mock virtual workspace serves at /selfcheck.
type selfCheck struct {
	User   string   `json:"user"`
	Groups []string `json:"groups"`

	ServingCertLoaded bool `json:"servingCertLoaded"`
	ClientCALoaded    bool `json:"clientCALoaded"`
	RequestHeaderCA   bool `json:"requestHeaderCALoaded"`

	AllowedReadSucceeded bool `json:"allowedReadSucceeded"`
	ForbiddenWriteDenied bool `json:"forbiddenWriteDenied"`

	Errors []string `json:"errors,omitempty"`
}

func mockImage(t *testing.T) *operatorv1alpha1.ImageSpec {
	t.Helper()

	image := os.Getenv("MOCK_VW_IMG")
	if image == "" {
		image = "ghcr.io/kcp-dev/mock-virtualworkspace:local"
	}

	repository, tag, found := splitImage(image)
	if !found {
		t.Fatalf("MOCK_VW_IMG %q has no tag", image)
	}

	return &operatorv1alpha1.ImageSpec{Repository: repository, Tag: tag}
}

func splitImage(image string) (repository, tag string, found bool) {
	for i := len(image) - 1; i >= 0; i-- {
		switch image[i] {
		case ':':
			return image[:i], image[i+1:], true
		case '/':
			return "", "", false
		}
	}

	return "", "", false
}

// TestCustomVirtualWorkspace deploys a virtual workspace server that is not kcp's own and checks
// the three things that are specific to such a server: that the operator runs the image's own
// entrypoint without passing it kcp-only flags, that an init container bootstraps kcp before the
// server starts, and that the two containers really do run as different identities.
//
// The mock refuses to start on an unknown flag, so a regression in argument generation shows up as
// a pod that never becomes ready rather than as a subtle assertion failure.
func TestCustomVirtualWorkspace(t *testing.T) {
	skipIfNotMainBranch(t)

	ctrlruntime.SetLogger(logr.Discard())

	configClient := utils.GetConfigKubeClient(t)
	workloadClient := utils.GetWorkloadKubeClient(t)
	ctx := context.Background()

	namespace := utils.CreateSelfDestructingNamespace(t, ctx, configClient, "custom-vw")

	externalHostname := fmt.Sprintf("front-proxy-front-proxy.%s.svc.cluster.local", namespace.Name)

	rootShard := utils.DeployRootShard(ctx, t, configClient, workloadClient, namespace.Name, externalHostname, func(rs *operatorv1alpha1.RootShard) {
		rs.Spec.Resources = lowResourceRequirements()
		if rs.Spec.Proxy != nil {
			rs.Spec.Proxy.Resources = lowResourceRequirements()
		}
	})

	frontProxy := utils.DeployFrontProxy(ctx, t, configClient, workloadClient, namespace.Name, rootShard.Name, externalHostname, func(fp *operatorv1alpha1.FrontProxy) {
		fp.Spec.Resources = lowResourceRequirements()
	})

	// The credentials for the two containers. Both go through the front-proxy: bootstrapping
	// walks the workspace tree, and a shard only serves the logical clusters it hosts.
	bootstrapSecret := createKubeconfig(t, ctx, configClient, namespace.Name, "mock-bootstrap", frontProxy.Name, func(kc *operatorv1alpha1.Kubeconfig) {
		kc.Spec.Username = "kcp-admin"
		kc.Spec.Groups = []string{"system:kcp:admin"}
	})

	// The serving identity holds nothing but what mockClusterRole grants. Both the role and the
	// binding are created by the init container, so this Kubeconfig only mints the credential.
	//
	// spec.authorization would do the binding instead, but its cleanup finalizer has to reach
	// kcp: deleting a namespace takes the front-proxy with it, and the Kubeconfig can then never
	// be finalized, wedging the namespace in Terminating.
	serverSecret := createKubeconfig(t, ctx, configClient, namespace.Name, "mock-server", frontProxy.Name, func(kc *operatorv1alpha1.Kubeconfig) {
		kc.Spec.Username = "mock-vw"
		kc.Spec.TargetWorkspace = "root:" + mockWorkspace
	})

	vw := utils.DeployVirtualWorkspace(ctx, t, configClient, workloadClient, namespace.Name, "mock-vw", false, func(vw *operatorv1alpha1.VirtualWorkspace) {
		vw.Spec.Target.RootShardRef = &corev1.LocalObjectReference{Name: rootShard.Name}
		vw.Spec.Resources = lowResourceRequirements()
		vw.Spec.Replicas = ptr.To(int32(1))

		// Not kcp's own server: its binary lives elsewhere and it rejects kcp-only flags.
		vw.Spec.Image = mockImage(t)
		vw.Spec.Command = []string{"/mock-virtualworkspace"}

		vw.Spec.KubeconfigSecretRef = &corev1.LocalObjectReference{Name: serverSecret}
		vw.Spec.ExtraArgs = []string{
			"--mode=serve",
			"--probe-configmap=" + mockProbeConfig,
		}

		vw.Spec.InitContainers = []operatorv1alpha1.VirtualWorkspaceInitContainer{{
			Name:                "bootstrap",
			KubeconfigSecretRef: &corev1.LocalObjectReference{Name: bootstrapSecret},
			Command:             []string{"/mock-virtualworkspace"},
			Args: []string{
				"--mode=init",
				"--kubeconfig=/etc/kcp/init-kubeconfig/kubeconfig",
				"--workspace=" + mockWorkspace,
				"--cluster-role=" + mockClusterRole,
				"--probe-configmap=" + mockProbeConfig,
				"--bind-user=mock-vw",
			},
			Resources: lowResourceRequirements(),
		}}
	})

	// The pod only becomes ready once the init container exited 0, which means bootstrapping
	// succeeded with the credentials it was given, and once the server survived flag parsing.
	t.Log("Waiting for the mock virtual workspace to become ready...")
	waitForMockPod(t, ctx, workloadClient, vw.Namespace, vw.Name)

	assertDeploymentShape(t, ctx, workloadClient, &vw)

	report := fetchSelfCheck(t, ctx, namespace.Name, &vw)

	t.Logf("Self check: %+v", report)

	if len(report.Errors) > 0 {
		t.Fatalf("Mock virtual workspace reported errors: %v", report.Errors)
	}

	// The credentials really are separate: the server holds the scoped identity, and the
	// administrative one it never saw belonged to the init container.
	if report.User != "mock-vw" {
		t.Errorf("Server runs as %q, expected the scoped identity %q.", report.User, "mock-vw")
	}
	for _, group := range []string{"system:masters", "system:kcp:admin"} {
		if slices.Contains(report.Groups, group) {
			t.Errorf("Server identity is in %q, which defeats the point of scoping it: %v", group, report.Groups)
		}
	}

	// And that identity is genuinely limited, not merely different.
	if !report.AllowedReadSucceeded {
		t.Error("Server could not read the ConfigMap its ClusterRole grants.")
	}
	if !report.ForbiddenWriteDenied {
		t.Error("Server was allowed to write a ConfigMap, so it is not scoped.")
	}

	// The PKI the operator mounts arrived intact.
	if !report.ServingCertLoaded || !report.ClientCALoaded || !report.RequestHeaderCA {
		t.Errorf("Server did not find all of its certificates: %+v", report)
	}

	t.Log("Custom virtual workspace served with a scoped identity, bootstrapped by an admin init container.")
}

// waitForMockPod waits for the virtual workspace pod to become ready, reporting what each
// container was actually doing if it does not.
//
// The shared helper only says that the wait timed out, which for this test is the least useful
// thing to know: the pod not starting is exactly how a missing mock image, a rejected command line
// argument or a failed bootstrap all present themselves, and they are told apart by the container
// states.
func waitForMockPod(t *testing.T, ctx context.Context, workloadClient ctrlruntimeclient.Client, namespace, name string) {
	t.Helper()

	listOpts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(namespace),
		ctrlruntimeclient.MatchingLabels{
			"app.kubernetes.io/component": "virtual-workspace",
			"app.kubernetes.io/instance":  name,
		},
	}

	var pods corev1.PodList

	err := wait.PollUntilContextTimeout(ctx, time.Second, 5*time.Minute, false, func(ctx context.Context) (bool, error) {
		pods = corev1.PodList{}
		if err := workloadClient.List(ctx, &pods, listOpts...); err != nil {
			return false, err
		}

		if len(pods.Items) == 0 {
			return false, nil
		}

		for _, pod := range pods.Items {
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					ready = cond.Status == corev1.ConditionTrue
				}
			}
			if !ready {
				return false, nil
			}
		}

		return true, nil
	})
	if err == nil {
		t.Log("Pods are ready.")

		return
	}

	for _, pod := range pods.Items {
		t.Logf("Pod %s is %s.", pod.Name, pod.Status.Phase)

		statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for _, status := range statuses {
			switch {
			case status.State.Waiting != nil:
				t.Logf("  %s: waiting (%s) %s", status.Name, status.State.Waiting.Reason, status.State.Waiting.Message)
			case status.State.Terminated != nil:
				t.Logf("  %s: terminated with exit code %d (%s) %s",
					status.Name, status.State.Terminated.ExitCode, status.State.Terminated.Reason, status.State.Terminated.Message)
			case status.State.Running != nil:
				t.Logf("  %s: running, ready=%t", status.Name, status.Ready)
			}
		}
	}

	t.Fatalf("The mock virtual workspace never became ready: %v", err)
}

// assertDeploymentShape checks the parts of the rendered Deployment that the mock cannot observe
// about itself.
func assertDeploymentShape(t *testing.T, ctx context.Context, workloadClient ctrlruntimeclient.Client, vw *operatorv1alpha1.VirtualWorkspace) {
	t.Helper()

	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{Namespace: vw.Namespace, Name: resources.GetVirtualWorkspaceDeploymentName(vw)}
	if err := workloadClient.Get(ctx, key, deployment); err != nil {
		t.Fatalf("Failed to get Deployment: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec

	if len(podSpec.InitContainers) != 1 {
		t.Fatalf("Expected exactly one init container, got %d.", len(podSpec.InitContainers))
	}

	server := podSpec.Containers[0]
	if got := server.Command; len(got) != 1 || got[0] != "/mock-virtualworkspace" {
		t.Errorf("Server command is %v, expected the image's own entrypoint.", got)
	}

	// The flags only kcp's own server understands must not be generated for this image. The mock
	// would have refused to start, but assert it directly so the failure names the cause.
	for _, arg := range server.Args {
		for _, forbidden := range []string{"--shard-external-url", "--cache-kubeconfig"} {
			if strings.HasPrefix(arg, forbidden) {
				t.Errorf("Server was passed the kcp-only flag %q.", arg)
			}
		}
	}

	// Each container sees its own credential and not the other's, nor the privileged fallback
	// that neither of them asked for.
	assertMounts(t, "server", server, "/etc/kcp/server-kubeconfig",
		"/etc/kcp/init-kubeconfig", "/etc/kcp/logical-cluster-admin-kubeconfig", "/etc/kcp/tls/logical-cluster-admin")
	assertMounts(t, "init", podSpec.InitContainers[0], "/etc/kcp/init-kubeconfig",
		"/etc/kcp/server-kubeconfig", "/etc/kcp/logical-cluster-admin-kubeconfig", "/etc/kcp/tls/logical-cluster-admin")
}

func assertMounts(t *testing.T, what string, container corev1.Container, want string, unwanted ...string) {
	t.Helper()

	paths := map[string]bool{}
	for _, mount := range container.VolumeMounts {
		paths[mount.MountPath] = true
	}

	if !paths[want] {
		t.Errorf("The %s container is missing %q; it has %v.", what, want, container.VolumeMounts)
	}
	for _, path := range unwanted {
		if paths[path] {
			t.Errorf("The %s container should not have %q mounted.", what, path)
		}
	}
}

// fetchSelfCheck port-forwards to the virtual workspace Service and reads the mock's report.
func fetchSelfCheck(t *testing.T, ctx context.Context, namespace string, vw *operatorv1alpha1.VirtualWorkspace) selfCheck {
	t.Helper()

	localPortStr, err := server.GetFreePort(t)
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	localPort, err := strconv.Atoi(localPortStr)
	if err != nil {
		t.Fatalf("Failed to parse port %q: %v", localPortStr, err)
	}

	serviceName := fmt.Sprintf("%s-virtual-workspace", vw.Name)
	utils.SelfDestuctingPortForward(t, ctx, namespace, "svc/"+serviceName, 6443, localPort)

	// The serving certificate is issued for the in-cluster name, not for localhost, so skip
	// verification here; that the certificate loaded at all is asserted through the report.
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
		Timeout:   10 * time.Second,
	}

	var report selfCheck
	url := fmt.Sprintf("https://localhost:%d/selfcheck", localPort)

	// The server answers 503 until its checks against kcp finish, and those wait for the
	// ClusterRoleBinding to take effect, so allow for that on top of port-forward startup.
	var lastStatus string

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false, err
		}

		resp, err := client.Do(req)
		if err != nil {
			// Port-forwarding may not be up yet.
			lastStatus = err.Error()

			return false, nil
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastStatus = err.Error()

			return false, nil
		}

		if resp.StatusCode == http.StatusServiceUnavailable {
			lastStatus = "checks still running"

			return false, nil
		}
		if resp.StatusCode != http.StatusOK {
			lastStatus = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, body)

			return false, nil
		}

		if err := json.Unmarshal(body, &report); err != nil {
			return false, fmt.Errorf("response is not a self check: %q", string(body))
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to read the self check: %v (last: %s)", err, lastStatus)
	}

	return report
}

// createKubeconfig creates a Kubeconfig targeting the given FrontProxy and waits for its Secret,
// returning the Secret's name.
func createKubeconfig(t *testing.T, ctx context.Context, configClient ctrlruntimeclient.Client, namespace, name, frontProxyName string, patches ...func(*operatorv1alpha1.Kubeconfig)) string {
	t.Helper()

	secretName := name + "-kubeconfig"

	kubeconfig := &operatorv1alpha1.Kubeconfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: operatorv1alpha1.KubeconfigSpec{
			Target: operatorv1alpha1.KubeconfigTarget{
				FrontProxyRef: &corev1.LocalObjectReference{Name: frontProxyName},
			},
			Validity:  metav1.Duration{Duration: 2 * time.Hour},
			SecretRef: corev1.LocalObjectReference{Name: secretName},
		},
	}

	for _, patch := range patches {
		patch(kubeconfig)
	}

	t.Logf("Creating Kubeconfig %s (user %q)...", name, kubeconfig.Spec.Username)
	if err := configClient.Create(ctx, kubeconfig); err != nil {
		t.Fatal(err)
	}

	utils.WaitForObject(t, ctx, configClient, &corev1.Secret{}, types.NamespacedName{Namespace: namespace, Name: secretName})

	return secretName
}
