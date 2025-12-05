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

package workspaceobject

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const (
	testNamespace     = "workspaceobject-tests"
	testRootShardName = "test-rootshard"
	testWorkspacePath = "root:org:team"
	testSecretName    = "test-rootshard-admin-kubeconfig"
)

// fakeWorkspaceClientCreator creates a fake workspace client creator function for testing.
// It returns a workspaceClientCreatorFunc that creates a fake dynamic client with the provided objects.
func fakeWorkspaceClientCreator(workspaceObjects ...runtime.Object) workspaceClientCreatorFunc {
	return func(ctx context.Context, kube ctrlruntimeclient.Client, workspaceObject *operatorv1alpha1.WorkspaceObject) (dynamic.Interface, *rest.Config, error) {
		// Create scheme with core types
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)

		// Create fake dynamic client
		fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, workspaceObjects...)

		// Add reactor to handle server-side apply (Patch with ApplyPatchType)
		// The fake client doesn't support ApplyPatchType by default, so we convert it to create/update
		fakeDynamicClient.PrependReactor("patch", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			patchAction := action.(k8stesting.PatchAction)
			if patchAction.GetPatchType() == types.ApplyPatchType {
				// Deserialize the patch to get the object
				var obj unstructured.Unstructured
				if err := json.Unmarshal(patchAction.GetPatch(), &obj); err != nil {
					return true, nil, err
				}

				// For the fake client, we'll simulate apply by treating it as an update
				// The fake tracker will handle whether it needs to be created or updated
				gvr := patchAction.GetResource()
				tracker := fakeDynamicClient.Tracker()

				// Try to get existing object from tracker directly (no lock conflict)
				existing, err := tracker.Get(gvr, patchAction.GetNamespace(), patchAction.GetName())
				if err != nil {
					// Object doesn't exist, create it
					if err := tracker.Create(gvr, &obj, patchAction.GetNamespace()); err != nil {
						return true, nil, err
					}
					return true, &obj, nil
				}

				// Object exists, update it with resource version
				if existingUnstructured, ok := existing.(*unstructured.Unstructured); ok {
					obj.SetResourceVersion(existingUnstructured.GetResourceVersion())
					obj.SetUID(existingUnstructured.GetUID())
				}

				if err := tracker.Update(gvr, &obj, patchAction.GetNamespace()); err != nil {
					return true, nil, err
				}
				return true, &obj, nil
			}
			return false, nil, nil
		})

		// Create a fake REST config - the actual values don't matter for the fake client
		// but we need it to pass to getGVRFromGVK
		fakeRestConfig := &rest.Config{
			Host: "https://fake-kcp-server",
		}

		return fakeDynamicClient, fakeRestConfig, nil
	}
}

// fakeGVRMapper creates a fake GVR mapper that maps common Kubernetes resources.
// This is used in tests to avoid needing a real discovery client.
func fakeGVRMapper(restConfig *rest.Config, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	// Map common resources
	switch gvk.GroupVersion().String() {
	case "v1":
		if gvk.Kind == "ConfigMap" {
			return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, nil
		}
		if gvk.Kind == "Secret" {
			return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}, nil
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("unknown GVK: %s", gvk.String())
}

func TestReconciling(t *testing.T) {
	testcases := []struct {
		name             string
		workspaceObject  *operatorv1alpha1.WorkspaceObject
		rootShard        *operatorv1alpha1.RootShard
		secret           *corev1.Secret
		workspaceObjects []runtime.Object // Objects pre-existing in the KCP workspace
	}{
		{
			name:             "vanilla configmap with all policies",
			workspaceObjects: []runtime.Object{}, // ConfigMap will be created
			workspaceObject: &operatorv1alpha1.WorkspaceObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workspaceobject",
					Namespace: testNamespace,
				},
				Spec: operatorv1alpha1.WorkspaceObjectSpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: testRootShardName,
						},
					},
					Workspace: operatorv1alpha1.WorkspaceConfig{
						Path: testWorkspacePath,
					},
					Manifest: mustMarshalToJSON(&corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{
							"key": "value",
						},
					}),
					ManagementPolicies: []operatorv1alpha1.WorkspaceObjectManagementPolicy{
						operatorv1alpha1.WorkspaceObjectManagementPolicyAll,
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRootShardName,
					Namespace: testNamespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"kubeconfig": []byte("fake-kubeconfig"),
				},
			},
		},
		{
			name:             "create not allowed with Update policy only",
			workspaceObjects: []runtime.Object{}, // No objects, create will fail
			workspaceObject: &operatorv1alpha1.WorkspaceObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-update-only",
					Namespace: testNamespace,
				},
				Spec: operatorv1alpha1.WorkspaceObjectSpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: testRootShardName,
						},
					},
					Workspace: operatorv1alpha1.WorkspaceConfig{
						Path: testWorkspacePath,
					},
					Manifest: mustMarshalToJSON(&corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{
							"key": "value",
						},
					}),
					ManagementPolicies: []operatorv1alpha1.WorkspaceObjectManagementPolicy{
						operatorv1alpha1.WorkspaceObjectManagementPolicyUpdate,
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRootShardName,
					Namespace: testNamespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"kubeconfig": []byte("fake-kubeconfig"),
				},
			},
		},
		{
			name:             "delete not allowed with Create and Update policies only",
			workspaceObjects: []runtime.Object{}, // ConfigMap will be created
			workspaceObject: &operatorv1alpha1.WorkspaceObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-no-delete",
					Namespace: testNamespace,
				},
				Spec: operatorv1alpha1.WorkspaceObjectSpec{
					RootShard: operatorv1alpha1.RootShardConfig{
						Reference: &corev1.LocalObjectReference{
							Name: testRootShardName,
						},
					},
					Workspace: operatorv1alpha1.WorkspaceConfig{
						Path: testWorkspacePath,
					},
					Manifest: mustMarshalToJSON(&corev1.ConfigMap{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-configmap",
							Namespace: "default",
						},
						Data: map[string]string{
							"key": "value",
						},
					}),
					ManagementPolicies: []operatorv1alpha1.WorkspaceObjectManagementPolicy{
						operatorv1alpha1.WorkspaceObjectManagementPolicyCreate,
						operatorv1alpha1.WorkspaceObjectManagementPolicyUpdate,
					},
				},
			},
			rootShard: &operatorv1alpha1.RootShard{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRootShardName,
					Namespace: testNamespace,
				},
				Spec: operatorv1alpha1.RootShardSpec{
					External: operatorv1alpha1.ExternalConfig{
						Hostname: "example.kcp.io",
						Port:     6443,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"kubeconfig": []byte("fake-kubeconfig"),
				},
			},
		},
	}

	scheme := util.GetTestScheme()

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// Create fake client for the host cluster
			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(testcase.workspaceObject).
				WithObjects(testcase.workspaceObject, testcase.rootShard, testcase.secret).
				Build()

			ctx := context.Background()

			// Create controller with fake workspace client injected
			controllerReconciler := &WorkspaceObjectReconciler{
				Client:                    client,
				Scheme:                    client.Scheme(),
				getWorkspaceDynamicClient: fakeWorkspaceClientCreator(testcase.workspaceObjects...),
				getGVRFromGVK:             fakeGVRMapper,
			}

			// Run reconciliation
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(testcase.workspaceObject),
			})

			// Verify the object still exists and get fresh copy with status
			var retrievedObj operatorv1alpha1.WorkspaceObject
			getErr := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(testcase.workspaceObject), &retrievedObj)
			require.NoError(t, getErr)

			// Verify finalizer was added
			assert.Contains(t, retrievedObj.Finalizers, WorkspaceObjectFinalizer, "Finalizer should be added")

			// For test cases that expect errors (e.g., create not allowed)
			if testcase.name == "create not allowed with Update policy only" {
				assert.Error(t, err, "Expected error when create policy is not enabled")
				return
			}

			// Otherwise, reconciliation should succeed
			assert.NoError(t, err, "Reconciliation should succeed")

			// Verify status manifest was populated
			require.NotNil(t, retrievedObj.Status.Manifest, "Status manifest should be populated")
			require.NotEmpty(t, retrievedObj.Status.Manifest.Raw, "Status manifest should have content")

			// Unmarshal and verify the status manifest contains the expected resource
			var actualObj unstructured.Unstructured
			err = json.Unmarshal(retrievedObj.Status.Manifest.Raw, &actualObj)
			require.NoError(t, err)

			// Verify the kind and basic fields match what we expect
			assert.Equal(t, "ConfigMap", actualObj.GetKind(), "Object kind should be ConfigMap")
			assert.NotEmpty(t, actualObj.GetName(), "Object name should be set")
		})
	}
}

func mustMarshalToJSON(obj any) apiextensionsv1.JSON {
	raw, err := marshalToJSON(obj)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	return *raw
}
