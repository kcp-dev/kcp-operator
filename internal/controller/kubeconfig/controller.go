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

package kubeconfig

import (
	"context"
	"errors"
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"k8c.io/reconciler/pkg/equality"
	k8creconciling "k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kcp-dev/kcp-operator/internal/controller/util"
	"github.com/kcp-dev/kcp-operator/internal/reconciling"
	"github.com/kcp-dev/kcp-operator/internal/resources"
	"github.com/kcp-dev/kcp-operator/internal/resources/kubeconfig"
	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// KubeconfigReconciler reconciles a Kubeconfig object
type KubeconfigReconciler struct {
	ctrlruntimeclient.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubeconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Kubeconfig{}).
		Watches(&operatorv1alpha1.RootShard{}, handler.EnqueueRequestsFromMapFunc(r.mapRootShardToKubeconfigs)).
		Watches(&operatorv1alpha1.Shard{}, handler.EnqueueRequestsFromMapFunc(r.mapShardToKubeconfigs)).
		Watches(&operatorv1alpha1.FrontProxy{}, handler.EnqueueRequestsFromMapFunc(r.mapFrontProxyToKubeconfigs)).
		Owns(&corev1.Secret{}).
		Owns(&certmanagerv1.Certificate{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kcp.io,resources=kubeconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.kcp.io,resources=rootshards,verbs=get;list;watch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *KubeconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")

	var kc operatorv1alpha1.Kubeconfig
	if err := r.Get(ctx, req.NamespacedName, &kc); err != nil {
		// object has been deleted.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if kc.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	kcCopy := kc.DeepCopy()
	kcCopy.Status.TargetName = r.getTargetName(&kc)

	conditions, recErr := r.reconcile(ctx, kcCopy, req.NamespacedName)
	if recErr == nil && len(conditions) > 0 {
		for _, cond := range conditions {
			if cond.Reason == "ClientCertificateSecretNotReady" ||
				cond.Reason == "ServerCASecretNotReady" {
				logger.V(4).Info("Reconciling again",
					"kubeconfig", req.NamespacedName,
					"message", cond.Message)

				_ = r.reconcileStatus(ctx, &kc, kcCopy, conditions)

				return ctrl.Result{RequeueAfter: time.Second * 5}, nil
			}
		}
	}

	if err := r.reconcileStatus(ctx, &kc, kcCopy, conditions); err != nil {
		recErr = kerrors.NewAggregate([]error{recErr, err})
	}

	return ctrl.Result{}, recErr
}

func (r *KubeconfigReconciler) reconcile(ctx context.Context, kc *operatorv1alpha1.Kubeconfig, req types.NamespacedName) ([]metav1.Condition, error) {
	var conditions []metav1.Condition

	rootShard := &operatorv1alpha1.RootShard{}
	shard := &operatorv1alpha1.Shard{}
	var frontProxy operatorv1alpha1.FrontProxy
	var caBundle *corev1.Secret

	var (
		clientCertIssuer string
		serverCA         string
	)

	switch {
	case kc.Spec.Target.RootShardRef != nil:
		if err := r.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.RootShardRef.Name, Namespace: req.Namespace}, rootShard); err != nil {
			err = fmt.Errorf("failed to get RootShard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
		serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	case kc.Spec.Target.ShardRef != nil:
		if err := r.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.ShardRef.Name, Namespace: req.Namespace}, shard); err != nil {
			err = fmt.Errorf("failed to get Shard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		ref := shard.Spec.RootShard.Reference
		if ref == nil || ref.Name == "" {
			err := errors.New("the Shard does not reference a (valid) RootShard")
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: req.Namespace}, rootShard); err != nil {
			err = fmt.Errorf("failed to get RootShard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		// The client CA is shared among all shards and owned by the root shard.
		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ClientCA)
		serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

	case kc.Spec.Target.FrontProxyRef != nil:
		if err := r.Get(ctx, types.NamespacedName{Name: kc.Spec.Target.FrontProxyRef.Name, Namespace: req.Namespace}, &frontProxy); err != nil {
			err = fmt.Errorf("referenced FrontProxy '%s' does not exist: %v", kc.Spec.Target.FrontProxyRef.Name, err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		ref := frontProxy.Spec.RootShard.Reference
		if ref == nil || ref.Name == "" {
			err := errors.New("the FrontProxy does not reference a (valid) RootShard")
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: "the FrontProxy does not reference a (valid) RootShard",
			})
			return conditions, err
		}
		if err := r.Get(ctx, types.NamespacedName{Name: frontProxy.Spec.RootShard.Reference.Name, Namespace: req.Namespace}, rootShard); err != nil {
			err = fmt.Errorf("failed to get RootShard: %w", err)
			conditions = append(conditions, metav1.Condition{
				Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
				Status:  metav1.ConditionFalse,
				Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
				Message: err.Error(),
			})
			return conditions, err
		}

		clientCertIssuer = resources.GetRootShardCAName(rootShard, operatorv1alpha1.FrontProxyClientCA)
		serverCA = resources.GetRootShardCAName(rootShard, operatorv1alpha1.ServerCA)

		if frontProxy.Spec.CABundleSecretRef != nil {
			caBundle = &corev1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Name: frontProxy.Spec.CABundleSecretRef.Name, Namespace: req.Namespace}, caBundle); err != nil {
				err = fmt.Errorf("failed to get CA bundle secret %s/%s: %w", req.Namespace, frontProxy.Spec.CABundleSecretRef.Name, err)
				conditions = append(conditions, metav1.Condition{
					Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
					Status:  metav1.ConditionFalse,
					Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
					Message: err.Error(),
				})
				return conditions, err
			}
		}

	default:
		err := errors.New("no valid target for kubeconfig found")
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
			Status:  metav1.ConditionFalse,
			Reason:  string(operatorv1alpha1.ConditionReasonReferenceNotFound),
			Message: err.Error(),
		})
		return conditions, err
	}

	conditions = append(conditions, metav1.Condition{
		Type:    string(operatorv1alpha1.ConditionTypeReferenceValid),
		Status:  metav1.ConditionTrue,
		Reason:  string(operatorv1alpha1.ConditionReasonReferenceValid),
		Message: "Target reference is valid",
	})

	certReconcilers := []reconciling.NamedCertificateReconcilerFactory{
		kubeconfig.ClientCertificateReconciler(kc, clientCertIssuer),
	}

	if err := reconciling.ReconcileCertificates(ctx, certReconcilers, req.Namespace, r.Client); err != nil {
		return conditions, err
	}

	clientCertSecret, err := r.getCertificateSecret(ctx, kc.GetCertificateName(), req.Namespace)
	if err != nil {
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeAvailable),
			Status:  metav1.ConditionFalse,
			Reason:  "ClientCertificateSecretError",
			Message: fmt.Sprintf("Failed to get server CA secret: %v", err),
		})
		return conditions, err
	} else if clientCertSecret == nil {
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeAvailable),
			Status:  metav1.ConditionFalse,
			Reason:  "ClientCertificateSecretNotReady",
			Message: "Server CA certificate is not ready yet",
		})
		return conditions, nil
	}

	serverCASecret, err := r.getCertificateSecret(ctx, serverCA, req.Namespace)
	if err != nil {
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeAvailable),
			Status:  metav1.ConditionFalse,
			Reason:  "ServerCASecretError",
			Message: fmt.Sprintf("Failed to get server CA secret: %v", err),
		})
		return conditions, err
	} else if serverCASecret == nil {
		conditions = append(conditions, metav1.Condition{
			Type:    string(operatorv1alpha1.ConditionTypeAvailable),
			Status:  metav1.ConditionFalse,
			Reason:  "ServerCASecretNotReady",
			Message: "Server CA certificate is not ready yet",
		})
		return conditions, nil
	}

	reconciler, err := kubeconfig.KubeconfigSecretReconciler(kc, rootShard, shard, frontProxy, serverCASecret, clientCertSecret, caBundle)
	if err != nil {
		return conditions, err
	}

	if err := k8creconciling.ReconcileSecrets(ctx, []k8creconciling.NamedSecretReconcilerFactory{reconciler}, req.Namespace, r.Client); err != nil {
		return conditions, err
	}

	conditions = append(conditions, metav1.Condition{
		Type:    string(operatorv1alpha1.ConditionTypeAvailable),
		Status:  metav1.ConditionTrue,
		Reason:  "SecretsReady",
		Message: "Client certificate and server CA secrets are ready",
	})

	return conditions, nil
}

func (r *KubeconfigReconciler) reconcileStatus(ctx context.Context, oldKc *operatorv1alpha1.Kubeconfig, kc *operatorv1alpha1.Kubeconfig, conditions []metav1.Condition) error {
	var errs []error

	for _, condition := range conditions {
		condition.ObservedGeneration = kc.Generation
		kc.Status.Conditions = util.UpdateCondition(kc.Status.Conditions, condition)
	}

	referenceValidCond := apimeta.FindStatusCondition(kc.Status.Conditions, string(operatorv1alpha1.ConditionTypeReferenceValid))
	if referenceValidCond != nil && referenceValidCond.Status == metav1.ConditionTrue {
		kc.Status.Phase = operatorv1alpha1.KubeconfigPhaseReady
	} else if referenceValidCond != nil && referenceValidCond.Status == metav1.ConditionFalse {
		kc.Status.Phase = operatorv1alpha1.KubeconfigPhaseFailed
	} else {
		kc.Status.Phase = operatorv1alpha1.KubeconfigPhaseProvisioning
	}

	if !equality.Semantic.DeepEqual(oldKc.Status, kc.Status) {
		if err := r.Status().Patch(ctx, kc, ctrlruntimeclient.MergeFrom(oldKc)); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}

func (r *KubeconfigReconciler) getCertificateSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx).WithValues("certificate", name)

	certificate := &certmanagerv1.Certificate{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, certificate); err != nil {
		// Because of how the reconciling framework works, this should never happen.
		logger.V(6).Info("Certificate does not exist yet, trying later ...")
		return nil, nil
	}

	ok := false
	for _, cond := range certificate.Status.Conditions {
		if cond.Type == certmanagerv1.CertificateConditionReady {
			ok = (cond.Status == certmanagermetav1.ConditionTrue)
			break
		}
	}

	if !ok {
		logger.V(4).Info("Certificate is not ready yet, trying later ...")
		return nil, nil
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: certificate.Spec.SecretName, Namespace: certificate.Namespace}, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *KubeconfigReconciler) mapRootShardToKubeconfigs(ctx context.Context, obj ctrlruntimeclient.Object) []ctrl.Request {
	logger := log.FromContext(ctx).WithValues("rootShard", obj.GetName())

	logger.V(4).Info("Mapping RootShard to Kubeconfigs")

	return r.mapKubeconfigs(ctx, func(t operatorv1alpha1.KubeconfigTarget) bool {
		return t.RootShardRef != nil && t.RootShardRef.Name == obj.GetName()
	})
}

func (r *KubeconfigReconciler) mapShardToKubeconfigs(ctx context.Context, obj ctrlruntimeclient.Object) []ctrl.Request {
	logger := log.FromContext(ctx).WithValues("shard", obj.GetName())

	logger.V(4).Info("Mapping Shard to Kubeconfigs")

	return r.mapKubeconfigs(ctx, func(t operatorv1alpha1.KubeconfigTarget) bool {
		return t.ShardRef != nil && t.ShardRef.Name == obj.GetName()
	})
}

func (r *KubeconfigReconciler) mapFrontProxyToKubeconfigs(ctx context.Context, obj ctrlruntimeclient.Object) []ctrl.Request {
	logger := log.FromContext(ctx).WithValues("frontProxy", obj.GetName())
	logger.V(4).Info("Mapping FrontProxy to Kubeconfigs")

	return r.mapKubeconfigs(ctx, func(t operatorv1alpha1.KubeconfigTarget) bool {
		return t.FrontProxyRef != nil && t.FrontProxyRef.Name == obj.GetName()
	})
}

func (r *KubeconfigReconciler) mapKubeconfigs(ctx context.Context, matches func(kc operatorv1alpha1.KubeconfigTarget) bool) []ctrl.Request {
	var kubeconfigs operatorv1alpha1.KubeconfigList
	if err := r.List(ctx, &kubeconfigs); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list Kubeconfigs")
		return []ctrl.Request{}
	}

	var requests []ctrl.Request
	for _, kc := range kubeconfigs.Items {
		if matches(kc.Spec.Target) {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      kc.Name,
					Namespace: kc.Namespace,
				},
			})
		}
	}

	return requests
}

func (r *KubeconfigReconciler) getTargetName(kc *operatorv1alpha1.Kubeconfig) string {
	if kc.Spec.Target.RootShardRef != nil {
		return "RootShard/" + kc.Spec.Target.RootShardRef.Name
	}
	if kc.Spec.Target.ShardRef != nil {
		return "Shard/" + kc.Spec.Target.ShardRef.Name
	}
	if kc.Spec.Target.FrontProxyRef != nil {
		return "FrontProxy/" + kc.Spec.Target.FrontProxyRef.Name
	}
	return ""
}
