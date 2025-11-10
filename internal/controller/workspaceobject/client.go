package workspaceobject

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

type workspaceClientCreatorFunc func(ctx context.Context, kube ctrlruntimeclient.Client, workspaceObject *operatorv1alpha1.WorkspaceObject) (dynamic.Interface, *rest.Config, error)

// getWorkspaceDynamicClient retrieves the kubeconfig from the RootShard and creates a dynamic client
// configured for the target workspace
func getWorkspaceDynamicClient(ctx context.Context, kube ctrlruntimeclient.Client, workspaceObject *operatorv1alpha1.WorkspaceObject) (dynamic.Interface, *rest.Config, error) {
	// Get RootShard
	var rootShard operatorv1alpha1.RootShard
	rootShardKey := types.NamespacedName{
		Namespace: workspaceObject.Namespace,
		Name:      workspaceObject.Spec.RootShard.Reference.Name,
	}
	if err := kube.Get(ctx, rootShardKey, &rootShard); err != nil {
		return nil, nil, fmt.Errorf("failed to get RootShard %s: %w", rootShardKey, err)
	}

	// Get workspace path from the WorkspaceConfig
	workspacePath := workspaceObject.Spec.Workspace.Path
	if workspacePath == "" {
		return nil, nil, fmt.Errorf("workspace path is empty in spec")
	}

	// Get kubeconfig secret from RootShard
	// secretName := fmt.Sprintf("%s-logical-cluster-admin-kubeconfig", rootShard.Name)
	secretName := fmt.Sprintf("%s-proxy-dynamic-kubeconfig", rootShard.Name)
	var secret corev1.Secret
	secretKey := types.NamespacedName{
		Namespace: rootShard.Namespace,
		Name:      secretName,
	}
	if err := kube.Get(ctx, secretKey, &secret); err != nil {
		return nil, nil, fmt.Errorf("failed to get kubeconfig secret %s: %w", secretKey, err)
	}

	kubeconfigData, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, nil, fmt.Errorf("kubeconfig not found in secret %s", secretKey)
	}

	// Build REST config (handles TLS embedding & host workspace path)
	config, err := buildWorkspaceRESTConfig(ctx, kube, &rootShard, workspacePath, kubeconfigData)
	if err != nil {
		return nil, nil, err
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return dynamicClient, config, nil
}

// buildWorkspaceRESTConfig constructs a *rest.Config for a workspace.
func buildWorkspaceRESTConfig(ctx context.Context, kube ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard, workspacePath string, kubeconfigData []byte) (*rest.Config, error) {
	certData, keyData, caData, caOK, err := fetchTLSSecret(ctx, kube, rootShard)
	if err != nil {
		return nil, err
	}

	rawConfig, err := clientcmd.Load(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	for name, cluster := range rawConfig.Clusters {
		cluster.CertificateAuthority = "" // remove path reference
		if caOK {
			cluster.CertificateAuthorityData = caData
		}
		rawConfig.Clusters[name] = cluster
	}
	for name, authInfo := range rawConfig.AuthInfos {
		authInfo.ClientCertificate = ""
		authInfo.ClientKey = ""
		authInfo.ClientCertificateData = certData
		authInfo.ClientKeyData = keyData
		rawConfig.AuthInfos[name] = authInfo
	}

	modifiedKubeconfigData, err := clientcmd.Write(*rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize modified kubeconfig: %w", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(modifiedKubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to build REST config: %w", err)
	}

	// Always overwrite TLS fields.
	config.CertData = certData
	config.KeyData = keyData
	if caOK {
		config.CAData = caData
	}

	// Append workspace path to host.
	config.Host = fmt.Sprintf("%s/clusters/%s", config.Host, workspacePath)

	return config, nil
}

// fetchTLSSecret retrieves TLS materials (client cert, key, optional CA) from the "%s-logical-cluster-admin" secret.
// Returns certData, keyData, caData, caOK flag and error.
func fetchTLSSecret(ctx context.Context, kube ctrlruntimeclient.Client, rootShard *operatorv1alpha1.RootShard) ([]byte, []byte, []byte, bool, error) {
	// tlsSecretName := fmt.Sprintf("%s-logical-cluster-admin", rootShard.Name)
	tlsSecretName := fmt.Sprintf("%s-proxy-kubeconfig", rootShard.Name)
	var tlsSecret corev1.Secret
	tlsSecretKey := types.NamespacedName{Namespace: rootShard.Namespace, Name: tlsSecretName}
	if err := kube.Get(ctx, tlsSecretKey, &tlsSecret); err != nil {
		return nil, nil, nil, false, fmt.Errorf("failed to get TLS secret %s: %w", tlsSecretKey, err)
	}

	certData, certOK := tlsSecret.Data["tls.crt"]
	keyData, keyOK := tlsSecret.Data["tls.key"]
	caData, caOK := tlsSecret.Data["ca.crt"] // optional

	if !certOK || !keyOK {
		return nil, nil, nil, false, fmt.Errorf("TLS secret %s missing tls.crt or tls.key", tlsSecretKey)
	}

	return certData, keyData, caData, caOK, nil
}

type mapperGVRFromGVKFunc func(restConfig *rest.Config, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error)

// getGVRFromGVK uses the discovery client to find the proper GroupVersionResource for a given GroupVersionKind
func getGVRFromGVK(restConfig *rest.Config, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Get API resources for the group/version
	apiResourceList, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to get API resources for %s: %w", gvk.GroupVersion().String(), err)
	}

	// Find the resource that matches our kind
	for _, apiResource := range apiResourceList.APIResources {
		if apiResource.Kind == gvk.Kind {
			return gvk.GroupVersion().WithResource(apiResource.Name), nil
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("no resource found for kind %s in %s", gvk.Kind, gvk.GroupVersion().String())
}
