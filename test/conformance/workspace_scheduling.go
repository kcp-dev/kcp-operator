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

package conformance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kcp-dev/logicalcluster/v3"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	kcptenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
	"github.com/kcp-dev/kcp-operator/test/utils"
)

type workspaceSchedulingTest struct {
	frontProxyName string
	hostClient     ctrlruntimeclient.Client
	hostNamespace  string
}

func NewWorkspaceSchedulingTest(frontProxyName string, hostClient ctrlruntimeclient.Client, hostNamespace string) *workspaceSchedulingTest {
	return &workspaceSchedulingTest{
		frontProxyName: frontProxyName,
		hostClient:     hostClient,
		hostNamespace:  hostNamespace,
	}
}

type logger interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func prefixLogger(log logger, prefix string) *prefixedLogger {
	return &prefixedLogger{
		log:    log,
		prefix: prefix,
	}
}

type prefixedLogger struct {
	log    logger
	prefix string
}

func (l *prefixedLogger) AppendPrefix(s string) *prefixedLogger {
	return &prefixedLogger{
		log:    l.log,
		prefix: l.prefix + " " + s,
	}
}

func (l *prefixedLogger) Logf(format string, args ...interface{}) {
	l.log.Logf(l.prefix+" "+format, args...)
}

func (l *prefixedLogger) Errorf(format string, args ...interface{}) {
	l.log.Errorf(l.prefix+" "+format, args...)
}

func (test *workspaceSchedulingTest) Run(t *testing.T, ctx context.Context, baseWorkspace logicalcluster.Path) error {
	testLog := prefixLogger(t, "[WorkspaceScheduling]")

	// create kubeconfig through the front-proxy
	const kubeconfigName = "workspace-scheduling"
	if err := test.createKubeconfig(testLog, ctx, kubeconfigName); err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	testLog.Logf("Setting up tunnel through front-proxy %s...", test.frontProxyName)
	frontProxyClusterClient := utils.KubeconfigClusterClient(t, ctx, test.hostClient, test.hostNamespace, kubeconfigName)

	// list all shards on the root shard
	rootWsClient := frontProxyClusterClient.Cluster(baseWorkspace)

	testLog.Logf("Listing available shards...")
	shardList := &kcpcorev1alpha1.ShardList{}
	if err := rootWsClient.List(ctx, shardList); err != nil {
		return fmt.Errorf("failed to list shards through front-proxy: %w", err)
	}

	// create a new workspace on every shard (includes the root shard)
	var hasErrors bool

	for _, shard := range shardList.Items {
		workspace := newWorkspace(shard.Name)
		shardLog := testLog.AppendPrefix(fmt.Sprintf("[wshard=root,cshard=%s]", shard.Name))

		shardLog.Logf("Creating workspace %s...", workspace.Name)
		if err := rootWsClient.Create(ctx, workspace); err != nil {
			// continue on errors, so we always get a complete picture of the errors
			shardLog.Errorf("Failed to create workspace: %v", err)
			hasErrors = true
			continue
		}

		// wait for the workspace to become ready
		shardLog.Logf("Waiting for workspace to become ready...")
		err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 60*time.Second, false, func(ctx context.Context) (done bool, err error) {
			err = rootWsClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(workspace), workspace)
			if err != nil {
				return false, err
			}

			return workspace.Status.Phase == kcpcorev1alpha1.LogicalClusterPhaseReady, nil
		})
		if err != nil {
			shardLog.Errorf("[FAIL] Failed to wait for workspace to become ready: %v", err)
			hasErrors = true
			continue
		}

		// create a client into the workspace (through the FP) and see if it responds correctly
		wsClient := frontProxyClusterClient.Cluster(baseWorkspace.Join(workspace.Name))
		lc := &kcpcorev1alpha1.LogicalCluster{}
		if err := wsClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: "cluster"}, lc); err != nil {
			shardLog.Errorf("[FAIL] Failed to get logicalcluster: %v", err)
			hasErrors = true
			continue
		}

		shardLog.Logf("[OK] Successfully scheduled workspace %s on shard %s!", workspace.Name, shard.Name)
	}

	if hasErrors {
		return fmt.Errorf("not all workspaces have become ready, setup is not working correctly")
	}

	return nil
}

func newWorkspace(targetShard string) *kcptenancyv1alpha1.Workspace {
	return &kcptenancyv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("test-workspace-%s", targetShard),
		},
		Spec: kcptenancyv1alpha1.WorkspaceSpec{
			Type: &kcptenancyv1alpha1.WorkspaceTypeReference{
				Name: "universal",
			},
			Location: &kcptenancyv1alpha1.WorkspaceLocation{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"name": targetShard,
					},
				},
			},
		},
	}
}

// createKubeconfig creates a Kubeconfig resource and waits for the secret to be generated.
func (test *workspaceSchedulingTest) createKubeconfig(log logger, ctx context.Context, name string) error {
	secretName := name + "-secret"
	kubeconfig := &operatorv1alpha1.Kubeconfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.hostNamespace,
		},
		Spec: operatorv1alpha1.KubeconfigSpec{
			Username: "e2e",
			Validity: metav1.Duration{Duration: 2 * time.Hour},
			SecretRef: corev1.LocalObjectReference{
				Name: secretName,
			},
			Groups: []string{"system:kcp:admin"},
			Target: operatorv1alpha1.KubeconfigTarget{
				FrontProxyRef: &corev1.LocalObjectReference{
					Name: test.frontProxyName,
				},
			},
		},
	}

	log.Logf("Creating kubeconfig %s...", name)
	if err := test.hostClient.Create(ctx, kubeconfig); err != nil {
		return fmt.Errorf("failed to create kubeconfig: %w", err)
	}

	err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, 3*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: test.hostNamespace, Name: secretName}

		err = test.hostClient.Get(ctx, key, secret)

		return err == nil, ctrlruntimeclient.IgnoreNotFound(err)
	})
	if err != nil {
		return fmt.Errorf("failed to wait for kubeconfig to become ready: %v", err)
	}

	return nil
}
