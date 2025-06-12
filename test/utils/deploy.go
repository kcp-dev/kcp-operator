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
	"os"
	"os/exec"
	"testing"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func DeployEtcd(t *testing.T, name, namespace string) string {
	t.Helper()

	t.Logf("Installing etcd %q into %s…", name, namespace)
	args := []string{
		"install",
		name,
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
		"--selector", fmt.Sprintf("app.kubernetes.io/name=etcd,app.kubernetes.io/instance=%s", name),
		"--for", "condition=Ready",
		"--timeout", "3m",
	}

	if err := exec.Command("kubectl", args...).Run(); err != nil {
		t.Fatalf("Failed to wait for etcd to become ready: %v", err)
	}

	return fmt.Sprintf("http://%s.%s.svc.cluster.local:2379", name, namespace)
}

func getKcpTag() string {
	return os.Getenv("KCP_TAG")
}

func applyShardEnv(spec operatorv1alpha1.CommonShardSpec) operatorv1alpha1.CommonShardSpec {
	if tag := getKcpTag(); tag != "" {
		spec.Image = &operatorv1alpha1.ImageSpec{
			Tag: tag,
		}
	}

	return spec
}

func DeployShard(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace, name, rootShard string, patches ...func(*operatorv1alpha1.Shard)) operatorv1alpha1.Shard {
	t.Helper()

	etcd := DeployEtcd(t, "etcd-"+name, namespace)

	shard := operatorv1alpha1.Shard{}
	shard.Name = name
	shard.Namespace = namespace

	shard.Spec = operatorv1alpha1.ShardSpec{
		RootShard: operatorv1alpha1.RootShardConfig{
			Reference: &corev1.LocalObjectReference{
				Name: rootShard,
			},
		},
		CommonShardSpec: applyShardEnv(operatorv1alpha1.CommonShardSpec{
			Etcd: operatorv1alpha1.EtcdConfig{
				Endpoints: []string{etcd},
			},
		}),
	}

	for _, patch := range patches {
		patch(&shard)
	}

	t.Logf("Creating Shard %s…", shard.Name)
	if err := client.Create(ctx, &shard); err != nil {
		t.Fatal(err)
	}

	opts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(shard.Namespace),
		ctrlruntimeclient.MatchingLabels{
			"app.kubernetes.io/component": "shard",
			"app.kubernetes.io/instance":  shard.Name,
		},
	}
	WaitForPods(t, ctx, client, opts...)

	return shard
}

func DeployRootShard(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, externalHostname string, patches ...func(*operatorv1alpha1.RootShard)) operatorv1alpha1.RootShard {
	t.Helper()

	etcd := DeployEtcd(t, "etcd-r00t", namespace)

	rootShard := operatorv1alpha1.RootShard{}
	rootShard.Name = "r00t"
	rootShard.Namespace = namespace

	rootShard.Spec = operatorv1alpha1.RootShardSpec{
		External: operatorv1alpha1.ExternalConfig{
			Hostname: externalHostname,
			Port:     6443,
		},
		Cache: operatorv1alpha1.CacheConfig{
			Embedded: &operatorv1alpha1.EmbeddedCacheConfiguration{
				Enabled: true,
			},
		},
		Certificates: operatorv1alpha1.Certificates{
			IssuerRef: GetSelfSignedIssuerRef(),
		},
		CommonShardSpec: applyShardEnv(operatorv1alpha1.CommonShardSpec{
			Etcd: operatorv1alpha1.EtcdConfig{
				Endpoints: []string{etcd},
			},
			CertificateTemplates: operatorv1alpha1.CertificateTemplateMap{
				string(operatorv1alpha1.ServerCertificate): operatorv1alpha1.CertificateTemplate{
					Spec: &operatorv1alpha1.CertificateSpecTemplate{
						DNSNames: []string{"localhost"},
					},
				},
			},
		}),
	}

	for _, patch := range patches {
		patch(&rootShard)
	}

	t.Logf("Creating RootShard %s…", rootShard.Name)
	if err := client.Create(ctx, &rootShard); err != nil {
		t.Fatal(err)
	}

	opts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(rootShard.Namespace),
		ctrlruntimeclient.MatchingLabels{
			"app.kubernetes.io/component": "rootshard",
			"app.kubernetes.io/instance":  rootShard.Name,
		},
	}
	WaitForPods(t, ctx, client, opts...)

	return rootShard
}

func DeployFrontProxy(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, namespace string, rootShardName string, externalHostname string, patches ...func(*operatorv1alpha1.FrontProxy)) operatorv1alpha1.FrontProxy {
	t.Helper()

	frontProxy := operatorv1alpha1.FrontProxy{}
	frontProxy.Name = "front-proxy"
	frontProxy.Namespace = namespace

	frontProxy.Spec = operatorv1alpha1.FrontProxySpec{
		RootShard: operatorv1alpha1.RootShardConfig{
			Reference: &corev1.LocalObjectReference{
				Name: rootShardName,
			},
		},
		Auth: &operatorv1alpha1.AuthSpec{
			// we need to remove the default system:masters group in order to do our testing
			DropGroups: []string{""},
		},
		CertificateTemplates: operatorv1alpha1.CertificateTemplateMap{
			string(operatorv1alpha1.ServerCertificate): operatorv1alpha1.CertificateTemplate{
				Spec: &operatorv1alpha1.CertificateSpecTemplate{
					DNSNames: []string{"localhost"},
				},
			},
		},
	}

	if tag := getKcpTag(); tag != "" {
		frontProxy.Spec.Image = &operatorv1alpha1.ImageSpec{
			Tag: tag,
		}
	}

	for _, patch := range patches {
		patch(&frontProxy)
	}

	t.Logf("Creating FrontProxy %s…", frontProxy.Name)
	if err := client.Create(ctx, &frontProxy); err != nil {
		t.Fatal(err)
	}

	opts := []ctrlruntimeclient.ListOption{
		ctrlruntimeclient.InNamespace(frontProxy.Namespace),
		ctrlruntimeclient.MatchingLabels{
			"app.kubernetes.io/component": "front-proxy",
			"app.kubernetes.io/instance":  frontProxy.Name,
		},
	}
	WaitForPods(t, ctx, client, opts...)

	return frontProxy
}
