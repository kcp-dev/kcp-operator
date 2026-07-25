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

// Command syncer copies Compiled* resources between the config and workload cluster.
//
// Production setups will have their own setup to synchronise resources between
// the config control plane and workload clusters, this is just a really simple
// spec/status copier for the e2e tests.
package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kcp-dev/kcp-operator/internal/resources"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
)

var compiledKinds = map[string]string{
	"CompiledCacheServer":      resources.CacheServerLabel,
	"CompiledFrontProxy":       resources.FrontProxyLabel,
	"CompiledRootShard":        resources.RootShardLabel,
	"CompiledShard":            resources.ShardLabel,
	"CompiledVirtualWorkspace": resources.VirtualWorkspaceLabel,
}

func main() {
	var (
		configKubeconfig   string
		workloadKubeconfig string
		namespacePrefix    string
	)

	flag.StringVar(&configKubeconfig, "config-kubeconfig", "", "Path to the kubeconfig for the config cluster.")
	flag.StringVar(&workloadKubeconfig, "workload-kubeconfig", "", "Path to the kubeconfig for the workload cluster.")
	flag.StringVar(&namespacePrefix, "namespace-prefix", "e2e-", "Only sync namespaces with this prefix.")
	flag.Parse()

	if configKubeconfig == "" || workloadKubeconfig == "" {
		log.Fatal("Both --config-kubeconfig and --workload-kubeconfig must be set.")
	}

	configClient, err := newClient(configKubeconfig)
	if err != nil {
		log.Fatalf("Failed to create config cluster client: %v.", err)
	}

	workloadClient, err := newClient(workloadKubeconfig)
	if err != nil {
		log.Fatalf("Failed to create workload cluster client: %v.", err)
	}

	ctx := ctrlruntime.SetupSignalHandler()
	s := &syncer{
		config:   configClient,
		workload: workloadClient,
		prefix:   namespacePrefix,
	}

	var wg sync.WaitGroup
	for kind := range compiledKinds {
		gvk := deployv1alpha1.SchemeGroupVersion.WithKind(kind)

		wg.Add(2)
		go func() {
			defer wg.Done()
			watchKind(ctx, configClient, gvk, s.syncToWorkload, s.deleteFromWorkload)
		}()
		go func() {
			defer wg.Done()
			watchKind(ctx, workloadClient, gvk, s.syncStatusToConfig, nil)
		}()
	}
	wg.Wait()
}

func newClient(kubeconfig string) (ctrlruntimeclient.WithWatch, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return ctrlruntimeclient.NewWithWatch(config, ctrlruntimeclient.Options{})
}

type eventHandler func(ctx context.Context, gvk schema.GroupVersionKind, obj *unstructured.Unstructured) error

func watchKind(ctx context.Context, client ctrlruntimeclient.WithWatch, gvk schema.GroupVersionKind, upsert, delete eventHandler) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
	if err := client.List(ctx, list); err != nil {
		log.Fatalf("Failed to list %s: %v.", gvk.Kind, err)
	}

	for i := range list.Items {
		if err := upsert(ctx, gvk, &list.Items[i]); err != nil {
			log.Printf("Failed to sync %s %s/%s: %v.", gvk.Kind, list.Items[i].GetNamespace(), list.Items[i].GetName(), err)
		}
	}

	watcher, err := watchtools.NewRetryWatcherWithContext(ctx, list.GetResourceVersion(), &cache.ListWatch{
		WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
			l := &unstructured.UnstructuredList{}
			l.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
			return client.Watch(ctx, l, &ctrlruntimeclient.ListOptions{Raw: &options})
		},
	})
	if err != nil {
		log.Fatalf("Failed to create watcher for %s: %v.", gvk.Kind, err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			obj, isUnstructured := event.Object.(*unstructured.Unstructured)
			if !isUnstructured {
				continue
			}

			var err error
			switch event.Type {
			case watch.Added, watch.Modified:
				err = upsert(ctx, gvk, obj)
			case watch.Deleted:
				if delete != nil {
					err = delete(ctx, gvk, obj)
				}
			}
			if err != nil {
				log.Printf("Failed to sync %s %s/%s: %v.", gvk.Kind, obj.GetNamespace(), obj.GetName(), err)
			}
		}
	}
}

type syncer struct {
	config   ctrlruntimeclient.WithWatch
	workload ctrlruntimeclient.WithWatch
	prefix   string
}

// syncToWorkload copies the object and its related secrets and configmaps to the workload cluster.
func (s *syncer) syncToWorkload(ctx context.Context, gvk schema.GroupVersionKind, obj *unstructured.Unstructured) error {
	if !strings.HasPrefix(obj.GetNamespace(), s.prefix) {
		return nil
	}

	if err := s.ensureNamespace(ctx, obj.GetNamespace()); err != nil {
		return err
	}

	labelKey := compiledKinds[gvk.Kind]
	if value := obj.GetLabels()[labelKey]; value != "" {
		if err := s.syncSecrets(ctx, obj.GetNamespace(), labelKey, value); err != nil {
			return err
		}

		if err := s.syncConfigMaps(ctx, obj.GetNamespace(), labelKey, value); err != nil {
			return err
		}
	}

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(gvk)
	if err := s.workload.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), existing); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		target := &unstructured.Unstructured{}
		target.SetGroupVersionKind(gvk)
		target.SetName(obj.GetName())
		target.SetNamespace(obj.GetNamespace())
		target.SetLabels(obj.GetLabels())
		target.Object["spec"] = obj.Object["spec"]

		log.Printf("Creating %s %s/%s on workload cluster.", gvk.Kind, obj.GetNamespace(), obj.GetName())
		return s.workload.Create(ctx, target)
	}

	existing.SetLabels(obj.GetLabels())
	existing.Object["spec"] = obj.Object["spec"]

	return s.workload.Update(ctx, existing)
}

func (s *syncer) deleteFromWorkload(ctx context.Context, gvk schema.GroupVersionKind, obj *unstructured.Unstructured) error {
	if !strings.HasPrefix(obj.GetNamespace(), s.prefix) {
		return nil
	}

	toDelete := &unstructured.Unstructured{}
	toDelete.SetGroupVersionKind(gvk)
	toDelete.SetName(obj.GetName())
	toDelete.SetNamespace(obj.GetNamespace())

	log.Printf("Deleting %s %s/%s on workload cluster.", gvk.Kind, obj.GetNamespace(), obj.GetName())
	return ctrlruntimeclient.IgnoreNotFound(s.workload.Delete(ctx, toDelete))
}

// syncStatusToConfig copies the status of the object back to the config cluster.
func (s *syncer) syncStatusToConfig(ctx context.Context, gvk schema.GroupVersionKind, obj *unstructured.Unstructured) error {
	if !strings.HasPrefix(obj.GetNamespace(), s.prefix) {
		return nil
	}

	status, ok := obj.Object["status"].(map[string]any)
	if !ok || len(status) == 0 {
		return nil
	}

	configObj := &unstructured.Unstructured{}
	configObj.SetGroupVersionKind(gvk)
	if err := s.config.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), configObj); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	configObj.Object["status"] = status

	return s.config.Status().Update(ctx, configObj)
}

func (s *syncer) ensureNamespace(ctx context.Context, name string) error {
	err := s.workload.Get(ctx, types.NamespacedName{Name: name}, &corev1.Namespace{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	log.Printf("Creating namespace %s on workload cluster.", name)

	return s.workload.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
}

func (s *syncer) syncSecrets(ctx context.Context, namespace, labelKey, labelValue string) error {
	secrets := &corev1.SecretList{}
	if err := s.config.List(ctx, secrets, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabels{labelKey: labelValue}); err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		existing := &corev1.Secret{}
		err := s.workload.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(&secret), existing)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}

			target := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret.Name,
					Namespace: secret.Namespace,
					Labels:    secret.Labels,
				},
				Type: secret.Type,
				Data: secret.Data,
			}
			if err := s.workload.Create(ctx, target); err != nil {
				return err
			}
			continue
		}

		existing.Labels = secret.Labels
		existing.Data = secret.Data
		if err := s.workload.Update(ctx, existing); err != nil {
			return err
		}
	}

	return nil
}

func (s *syncer) syncConfigMaps(ctx context.Context, namespace, labelKey, labelValue string) error {
	configMaps := &corev1.ConfigMapList{}
	if err := s.config.List(ctx, configMaps, ctrlruntimeclient.InNamespace(namespace), ctrlruntimeclient.MatchingLabels{labelKey: labelValue}); err != nil {
		return err
	}

	for _, configMap := range configMaps.Items {
		existing := &corev1.ConfigMap{}
		err := s.workload.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(&configMap), existing)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}

			target := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
					Labels:    configMap.Labels,
				},
				Data:       configMap.Data,
				BinaryData: configMap.BinaryData,
			}
			if err := s.workload.Create(ctx, target); err != nil {
				return err
			}
			continue
		}

		existing.Labels = configMap.Labels
		existing.Data = configMap.Data
		existing.BinaryData = configMap.BinaryData
		if err := s.workload.Update(ctx, existing); err != nil {
			return err
		}
	}

	return nil
}
