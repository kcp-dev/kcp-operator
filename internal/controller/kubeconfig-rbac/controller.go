/*
Copyright 2025 The KCP Authors.

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

package kubeconfigrbac

import (
	"context"
	"fmt"
	"slices"

	"github.com/kcp-dev/logicalcluster/v3"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/client"
	"github.com/kcp-dev/kcp-operator/internal/resources/kubeconfig"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

const cleanupFinalizer = "operator.kcp.io/cleanup-rbac"

// KubeconfigRBACReconciler reconciles a Kubeconfig object
type KubeconfigRBACReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeconfigRBACReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Kubeconfig{}).
		Named("kubeconfig-rbac").
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KubeconfigRBACReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	config := &operatorv1alpha1.Kubeconfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		return ctrl.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, config)

	return ctrl.Result{}, err
}

func (r *KubeconfigRBACReconciler) reconcile(ctx context.Context, config *operatorv1alpha1.Kubeconfig) error {
	if config.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, config)
	}

	// NB: Reconciling a Kubeconfig assumes that the authz settings are immutable, i.e. it is not
	// possible to first configure RBAC for workspace A and then update the Kubeconfig to mean workspace B.

	// No auth configured right now and since there is no finalizer, we also have nothing to
	// potentially clean up, hence we're done here.
	if config.Spec.Authorization == nil && !slices.Contains(config.Finalizers, cleanupFinalizer) {
		return nil
	}

	// If there is any kind of authorization configured, first we ensure our own finalizer.
	if config.Spec.Authorization != nil {
		updated, err := r.ensureFinalizer(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to ensure cleanup finalizer: %w", err)
		}

		if updated {
			return nil // will requeue because we changed the object
		}
	}

	// Make sure whatever is in the workspace matches what is configured in the Kubeconfig
	if err := r.reconcileBindings(ctx, config); err != nil {
		return fmt.Errorf("failed to ensure ClusterRoleBindings: %w", err)
	}

	// If nothing is configured, now it the perfect time to remove our finalizer again
	// so that for future reconciliations, we quickly know that we can ignore this Kubeconfig.
	if config.Spec.Authorization == nil {
		if err := r.removeFinalizer(ctx, config); err != nil {
			return fmt.Errorf("failed to remove cleanup finalizer: %w", err)
		}
	}

	return nil
}

func (r *KubeconfigRBACReconciler) reconcileBindings(ctx context.Context, kc *operatorv1alpha1.Kubeconfig) error {
	targetClient, err := client.NewInternalKubeconfigClient(ctx, r.Client, kc, logicalcluster.Name(kc.Spec.Authorization.ClusterRoleBindings.Cluster), nil)
	if err != nil {
		return fmt.Errorf("failed to create client to kubeconfig target: %w", err)
	}

	// find all existing bindings
	ownerLabels := kubeconfig.OwnerLabels(kc)
	crbList := &rbacv1.ClusterRoleBindingList{}
	if err := targetClient.List(ctx, crbList, ctrlruntimeclient.MatchingLabels(ownerLabels)); err != nil {
		return fmt.Errorf("failed to list existing ClusterRoleBindings: %w", err)
	}

	// delete those not configured in the kubeconfig anymore
	var desiredBindings sets.Set[string]
	if a := kc.Spec.Authorization; a != nil {
		desiredBindings = sets.New(a.ClusterRoleBindings.ClusterRoles...)
	}

	logger := log.FromContext(ctx)

	for _, crb := range crbList.Items {
		roleName := crb.RoleRef.Name

		if !desiredBindings.Has(roleName) {
			logger.V(2).WithValues("name", crb.Name, "clusterrole", roleName).Info("Deleting overhanging ClusterRoleBinding")

			if err := targetClient.Delete(ctx, &crb); err != nil {
				return fmt.Errorf("failed to delete overhanging ClusterRoleBinding %s: %w", crb.Name, err)
			}
		}
	}

	// create reconcilers for each intended binding
	subject := rbacv1.Subject{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "Group",
		Name:     kubeconfig.KubeconfigGroup(kc),
	}

	reconcilers := make([]reconciling.NamedClusterRoleBindingReconcilerFactory, 0, desiredBindings.Len())
	for _, roleName := range sets.List(desiredBindings) {
		reconcilers = append(reconcilers, kubeconfig.ClusterRoleBindingReconciler(kc, roleName, subject))
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, reconcilers, "", targetClient); err != nil {
		return fmt.Errorf("failed to ensure ClusterRoleBindings: %w", err)
	}

	return nil
}

func (r *KubeconfigRBACReconciler) handleDeletion(ctx context.Context, kc *operatorv1alpha1.Kubeconfig) error {
	// Did we already perform our cleanup or did this kubeconfig never have any bindings?
	if !slices.Contains(kc.Finalizers, cleanupFinalizer) {
		return nil
	}

	// This should always be true, unless cleanup succeeded but removing the finalizer failed in a
	// previous reconcile cycle.
	if cluster := kc.Status.Authorization.ProvisionedCluster; cluster != "" {
		targetClient, err := client.NewInternalKubeconfigClient(ctx, r.Client, kc, logicalcluster.Name(cluster), nil)
		if err != nil {
			return fmt.Errorf("failed to create client to kubeconfig target: %w", err)
		}

		// find all existing bindings
		ownerLabels := kubeconfig.OwnerLabels(kc)
		crbList := &rbacv1.ClusterRoleBindingList{}
		if err := targetClient.List(ctx, crbList, ctrlruntimeclient.MatchingLabels(ownerLabels)); err != nil {
			return fmt.Errorf("failed to list existing ClusterRoleBindings: %w", err)
		}

		// delete all of them
		logger := log.FromContext(ctx)

		for _, crb := range crbList.Items {
			logger.V(2).WithValues("name", crb.Name).Info("Deleting ClusterRoleBinding")

			if err := targetClient.Delete(ctx, &crb); err != nil {
				return fmt.Errorf("failed to delete ClusterRoleBinding %s: %w", crb.Name, err)
			}
		}

		// clean status
		oldKubeconfig := kc.DeepCopy()
		kc.Status.Authorization.ProvisionedCluster = ""
		if err := r.Status().Patch(ctx, kc, ctrlruntimeclient.MergeFrom(oldKubeconfig)); err != nil {
			return fmt.Errorf("failed to finish cleanup by updating status: %w", err)
		}
	}

	// when all are gone, remove the finalizer
	if err := r.removeFinalizer(ctx, kc); err != nil {
		return fmt.Errorf("failed to remove cleanup finalizer: %w", err)
	}

	return nil
}

func (r *KubeconfigRBACReconciler) ensureFinalizer(ctx context.Context, config *operatorv1alpha1.Kubeconfig) (updated bool, err error) {
	finalizers := sets.New(config.GetFinalizers()...)
	if finalizers.Has(cleanupFinalizer) {
		return false, nil
	}

	original := config.DeepCopy()

	finalizers.Insert(cleanupFinalizer)
	config.SetFinalizers(sets.List(finalizers))

	if err := r.Patch(ctx, config, ctrlruntimeclient.MergeFrom(original)); err != nil {
		return false, err
	}

	return true, nil
}

func (r *KubeconfigRBACReconciler) removeFinalizer(ctx context.Context, config *operatorv1alpha1.Kubeconfig) error {
	finalizers := sets.New(config.GetFinalizers()...)
	if !finalizers.Has(cleanupFinalizer) {
		return nil
	}

	original := config.DeepCopy()

	finalizers.Delete(cleanupFinalizer)
	config.SetFinalizers(sets.List(finalizers))

	return r.Patch(ctx, config, ctrlruntimeclient.MergeFrom(original))
}
