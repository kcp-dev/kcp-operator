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

package compiledcacheserver

import (
	"context"
	"errors"
	"time"

	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/reconciling/modifier"
	"github.com/kcp-dev/kcp-operator/internal/resources/cacheserver"
	deployv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/deploy/v1alpha1"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// requeueAfter is the retry delay when a mounted Secret or ConfigMap does not exist yet.
const requeueAfter = 10 * time.Second

// Reconciler reconciles a CompiledCacheServer object.
type Reconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("compiledcacheserver").
		For(&deployv1alpha1.CompiledCacheServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=deploy.operator.kcp.io,resources=compiledcacheservers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var compiled deployv1alpha1.CompiledCacheServer
	if err := r.Get(ctx, req.NamespacedName, &compiled); err != nil {
		return ctrl.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if compiled.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, &compiled)
}

func (r *Reconciler) reconcile(ctx context.Context, compiled *deployv1alpha1.CompiledCacheServer) (ctrl.Result, error) {
	var (
		errs    []error
		requeue bool
	)

	server := &operatorv1alpha1.CacheServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        compiled.Name,
			Namespace:   compiled.Namespace,
			Labels:      compiled.Labels,
			Annotations: compiled.Annotations,
		},
		Spec: compiled.Spec.CacheServer,
	}

	ownerRef := *metav1.NewControllerRef(compiled, deployv1alpha1.SchemeGroupVersion.WithKind("CompiledCacheServer"))
	ownerRefWrapper := k8creconciling.OwnerRefWrapper(ownerRef)

	revisionLabels := modifier.RelatedRevisionsLabels(ctx, r.Client)

	if err := k8creconciling.ReconcileDeployments(ctx, []k8creconciling.NamedDeploymentReconcilerFactory{
		cacheserver.DeploymentReconciler(server),
	}, compiled.Namespace, r.Client, ownerRefWrapper, revisionLabels); err != nil {
		if errors.Is(err, modifier.ErrMountNotFound) {
			requeue = true
		} else {
			errs = append(errs, err)
		}
	}

	if err := k8creconciling.ReconcileServices(ctx, []k8creconciling.NamedServiceReconcilerFactory{
		cacheserver.ServiceReconciler(server),
	}, compiled.Namespace, r.Client, ownerRefWrapper); err != nil {
		errs = append(errs, err)
	}

	result := ctrl.Result{}
	if requeue {
		result.RequeueAfter = requeueAfter
	}

	return result, kerrors.NewAggregate(errs)
}
