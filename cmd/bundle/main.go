/*
Copyright 2026 The KCP Authors.

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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

var (
	sourceKubeconfig  string
	targetKubeconfig  string
	bundleName        string
	bundleNamespace   string
	reconcileInterval time.Duration
	createNamespace   bool
	dryRun            bool
	dryMode           bool
)

func init() {
	flag.StringVar(&sourceKubeconfig, "kubeconfig", "", "Path to source kubeconfig (defaults to KUBECONFIG env var or in-cluster config)")
	flag.StringVar(&targetKubeconfig, "target-kubeconfig-path", "", "Path to target kubeconfig where bundle will be applied (required)")
	flag.StringVar(&bundleName, "bundle-name", "", "Name of the bundle Secret to apply (required)")
	flag.StringVar(&bundleNamespace, "bundle-namespace", "", "Namespace of the bundle Secret (required)")
	flag.DurationVar(&reconcileInterval, "reconcile-interval", 30*time.Second, "Interval between reconciliation loops")
	flag.BoolVar(&createNamespace, "create-namespace", true, "Create namespace on target cluster if it doesn't exist")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run mode - log what would be applied without actually applying")
	flag.BoolVar(&dryMode, "dry-mode", false, "Dry mode - print YAML files to stdout instead of applying")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// Validate required flags
	if targetKubeconfig == "" {
		klog.Fatal("--target-kubeconfig-path is required")
	}
	if bundleName == "" {
		klog.Fatal("--bundle-name is required")
	}
	if bundleNamespace == "" {
		klog.Fatal("--bundle-namespace is required")
	}

	// Use KUBECONFIG env var if --kubeconfig not specified
	if sourceKubeconfig == "" {
		sourceKubeconfig = os.Getenv("KUBECONFIG")
	}

	ctx := context.Background()

	// Setup source cluster client (where bundles are stored)
	sourceConfig, err := buildConfig(sourceKubeconfig)
	if err != nil {
		klog.Fatalf("Failed to build source config: %v", err)
	}
	sourceClientset, err := kubernetes.NewForConfig(sourceConfig)
	if err != nil {
		klog.Fatalf("Failed to create source clientset: %v", err)
	}

	// In dry-mode, we only need the source client to fetch the bundle
	if dryMode {
		if err := printBundle(ctx, sourceClientset); err != nil {
			klog.Fatalf("Failed to print bundle: %v", err)
		}
		return
	}

	// Setup target cluster client (where bundles will be applied)
	targetConfig, err := buildConfig(targetKubeconfig)
	if err != nil {
		klog.Fatalf("Failed to build target config: %v", err)
	}
	targetClientset, err := kubernetes.NewForConfig(targetConfig)
	if err != nil {
		klog.Fatalf("Failed to create target clientset: %v", err)
	}
	targetDynamicClient, err := dynamic.NewForConfig(targetConfig)
	if err != nil {
		klog.Fatalf("Failed to create target dynamic client: %v", err)
	}

	klog.Infof("Starting bundle applier: source=%s, target=%s, bundle=%s/%s",
		formatKubeconfig(sourceKubeconfig), targetKubeconfig, bundleNamespace, bundleName)

	// Main reconciliation loop
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()

	// Run immediately on startup
	if err := reconcileBundle(ctx, sourceClientset, targetClientset, targetDynamicClient); err != nil {
		klog.Errorf("Failed to reconcile bundle: %v", err)
	}

	// Then run on interval
	for {
		select {
		case <-ctx.Done():
			klog.Info("Context cancelled, exiting")
			return
		case <-ticker.C:
			if err := reconcileBundle(ctx, sourceClientset, targetClientset, targetDynamicClient); err != nil {
				klog.Errorf("Failed to reconcile bundle: %v", err)
			}
		}
	}
}

func reconcileBundle(ctx context.Context, sourceClient, targetClient *kubernetes.Clientset, targetDynamic dynamic.Interface) error {
	klog.V(2).Infof("Reconciling bundle %s/%s", bundleNamespace, bundleName)

	// Get the bundle Secret from source cluster
	secret, err := sourceClient.CoreV1().Secrets(bundleNamespace).Get(ctx, bundleName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Warningf("Bundle Secret %s/%s not found in source cluster", bundleNamespace, bundleName)
			return nil
		}
		return fmt.Errorf("failed to get bundle Secret: %w", err)
	}

	klog.V(2).Infof("Found bundle Secret with %d objects", len(secret.Data))

	// Create namespace on target cluster if needed
	if createNamespace {
		if err := ensureNamespace(ctx, targetClient, bundleNamespace); err != nil {
			return fmt.Errorf("failed to ensure namespace: %w", err)
		}
	}

	// Apply each object from the bundle
	applied := 0
	skipped := 0
	failed := 0

	for key, data := range secret.Data {
		if err := applyObject(ctx, targetDynamic, data); err != nil {
			klog.Errorf("Failed to apply object %s: %v", key, err)
			failed++
			continue
		}
		applied++
	}

	klog.Infof("Bundle reconciliation complete: applied=%d, skipped=%d, failed=%d", applied, skipped, failed)
	return nil
}

func applyObject(ctx context.Context, client dynamic.Interface, data []byte) error {
	// Parse the object
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to unmarshal object: %w", err)
	}

	gvk := obj.GroupVersionKind()
	if gvk.Empty() {
		return fmt.Errorf("object has empty GVK")
	}

	// Convert GVK to GVR
	gvr := gvkToGVR(gvk)

	if gvr.Group == "operator.kcp.io" {
		return nil
	}

	namespace := obj.GetNamespace()
	name := obj.GetName()

	if dryRun {
		klog.Infof("[DRY RUN] Would apply %s %s/%s", gvk.Kind, namespace, name)
		return nil
	}

	klog.V(3).Infof("Applying %s %s/%s", gvk.Kind, namespace, name)

	// Get resource client
	var resourceClient dynamic.ResourceInterface
	if namespace != "" {
		resourceClient = client.Resource(gvr).Namespace(namespace)
	} else {
		resourceClient = client.Resource(gvr)
	}

	// Try to get existing object
	existing, err := resourceClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object doesn't exist, create it
			// Clean metadata fields that shouldn't be set on creation
			cleanObjectForCreate(obj)
			_, err := resourceClient.Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create object: %w", err)
			}
			klog.V(2).Infof("Created %s %s/%s", gvk.Kind, namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get existing object: %w", err)
	}

	// Object exists, update it
	// Preserve resourceVersion and UID
	obj.SetResourceVersion(existing.GetResourceVersion())
	obj.SetUID(existing.GetUID())

	_, err = resourceClient.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update object: %w", err)
	}
	klog.V(2).Infof("Updated %s %s/%s", gvk.Kind, namespace, name)

	return nil
}

func ensureNamespace(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	if dryRun {
		klog.Infof("[DRY RUN] Would ensure namespace %s exists", namespace)
		return nil
	}

	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// Create the namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	klog.Infof("Created namespace %s", namespace)
	return nil
}

func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	// If no kubeconfig specified, try in-cluster config
	if kubeconfigPath == "" {
		klog.V(2).Info("No kubeconfig specified, trying in-cluster config")
		config, err := rest.InClusterConfig()
		if err == nil {
			return config, nil
		}
		return nil, fmt.Errorf("no kubeconfig specified and in-cluster config failed: %w", err)
	}

	// Load kubeconfig from file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

func formatKubeconfig(path string) string {
	if path == "" {
		return "in-cluster"
	}
	return path
}

// cleanObjectForCreate removes metadata fields that should not be set when creating a new object
func cleanObjectForCreate(obj *unstructured.Unstructured) {
	// Remove fields that are set by the API server on creation
	obj.SetResourceVersion("")
	obj.SetUID("")
	obj.SetSelfLink("")
	obj.SetGeneration(0)
	obj.SetCreationTimestamp(metav1.Time{})

	// Remove managed fields - these are set by the API server
	obj.SetManagedFields(nil)

	// Remove owner references - these would reference objects that don't exist in target cluster
	obj.SetOwnerReferences(nil)
}

// gvkToGVR converts a GroupVersionKind to a GroupVersionResource
// This is a simple heuristic that works for most standard Kubernetes resources
func gvkToGVR(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	// Handle special cases
	switch gvk.Kind {
	case "Endpoints":
		return schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: "endpoints",
		}
	case "NetworkPolicy":
		return schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: "networkpolicies",
		}
	}

	// Default pluralization: lowercase + 's'
	// For most resources, this is sufficient
	resource := strings.ToLower(gvk.Kind) + "s"

	// Handle resources ending in 'y' -> 'ies'
	if strings.HasSuffix(gvk.Kind, "y") && len(gvk.Kind) > 1 {
		resource = strings.ToLower(gvk.Kind[:len(gvk.Kind)-1]) + "ies"
	}

	// Handle resources ending in 's', 'x', 'z' -> 'es'
	if strings.HasSuffix(gvk.Kind, "s") || strings.HasSuffix(gvk.Kind, "x") || strings.HasSuffix(gvk.Kind, "z") {
		resource = strings.ToLower(gvk.Kind) + "es"
	}

	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}

// printBundle fetches the bundle Secret and prints all YAML files to stdout
func printBundle(ctx context.Context, sourceClient *kubernetes.Clientset) error {
	// Get the bundle Secret from source cluster
	secret, err := sourceClient.CoreV1().Secrets(bundleNamespace).Get(ctx, bundleName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get bundle Secret: %w", err)
	}

	klog.V(2).Infof("Found bundle Secret with %d objects", len(secret.Data))

	// Print each object as YAML
	for _, data := range secret.Data {
		if err := printObject(data); err != nil {
			klog.Errorf("Failed to print object: %v", err)
			continue
		}
	}

	return nil
}

// printObject converts JSON data to YAML and prints it to stdout
func printObject(data []byte) error {
	// Parse the object
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to unmarshal object: %w", err)
	}
	obj.Object["status"] = nil // Remove status for cleaner output
	obj.Object["metadata"].(map[string]interface{})["managedFields"] = nil

	// Convert to YAML
	yamlData, err := yaml.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	// Print separator and YAML
	fmt.Println("---")
	fmt.Print(string(yamlData))

	return nil
}
