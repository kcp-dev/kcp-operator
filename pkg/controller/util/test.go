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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

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

// FakeSingleCluster serves the given client for every cluster name, so a reconciler can be tested
// without a manager or a provider.
func FakeSingleCluster(c ctrlruntimeclient.Client) func(context.Context, multicluster.ClusterName) (cluster.Cluster, error) {
	return func(context.Context, multicluster.ClusterName) (cluster.Cluster, error) {
		return &testCluster{client: c}, nil
	}
}

// testCluster implements only the methods the reconcilers use; the rest panic if ever called.
type testCluster struct {
	cluster.Cluster

	client ctrlruntimeclient.Client
}

func (c *testCluster) GetClient() ctrlruntimeclient.Client {
	return c.client
}

func (c *testCluster) GetScheme() *runtime.Scheme {
	return c.client.Scheme()
}
