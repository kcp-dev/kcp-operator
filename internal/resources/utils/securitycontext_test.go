package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func newSecCtxTestDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "kcp",
						SecurityContext: &corev1.SecurityContext{
							SeccompProfile: &corev1.SeccompProfile{
								Type: corev1.SeccompProfileTypeRuntimeDefault,
							},
							ReadOnlyRootFilesystem:   ptr.To(true),
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									corev1.Capability("ALL"),
								},
							},
						},
					}},
				},
			},
		},
	}
}

func TestApplyPodSecurityContext(t *testing.T) {
	tests := []struct {
		name           string
		podSecurityCtx *corev1.PodSecurityContext
		preTransformFn func(deployment *appsv1.Deployment)
		validateResult func(t *testing.T, deployment *appsv1.Deployment)
	}{
		{
			name:           "defaults to nil if podSecurityContext not configured",
			podSecurityCtx: nil,
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				assert.Nil(t, deployment.Spec.Template.Spec.SecurityContext)
			},
		},
		{
			name: "sets the specified podSecurityContext",
			podSecurityCtx: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				require.NotNil(t, deployment.Spec.Template.Spec.SecurityContext)
				expected := &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(true),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				}
				assert.Equal(t, expected, deployment.Spec.Template.Spec.SecurityContext)
			},
		},
		{
			name:           "persists pre-set podSecurityContext if not specified",
			podSecurityCtx: nil,
			preTransformFn: func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(false),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeUnconfined,
					},
				}
			},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				require.NotNil(t, deployment.Spec.Template.Spec.SecurityContext)
				expected := &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(false),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeUnconfined,
					},
				}
				assert.Equal(t, expected, deployment.Spec.Template.Spec.SecurityContext)
			},
		},
		{
			name: "overrides pre-set podSecurityContext if specified",
			podSecurityCtx: &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			preTransformFn: func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(false),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeUnconfined,
					},
				}
			},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				require.NotNil(t, deployment.Spec.Template.Spec.SecurityContext)
				expected := &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(true),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				}
				assert.Equal(t, expected, deployment.Spec.Template.Spec.SecurityContext)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deployment := newSecCtxTestDeployment()

			if tc.preTransformFn != nil {
				tc.preTransformFn(deployment)
			}
			result := ApplyPodSecurityContext(deployment, tc.podSecurityCtx)
			tc.validateResult(t, result)
		})
	}
}

func TestApplyContainerSecurityContext(t *testing.T) {
	tests := []struct {
		name           string
		securityCtx    *corev1.SecurityContext
		preTransformFn func(deployment *appsv1.Deployment)
		validateResult func(t *testing.T, deployment *appsv1.Deployment)
	}{
		{
			name:        "securityContext not configured preserves existing contexts",
			securityCtx: nil,
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				expected := newSecCtxTestDeployment().Spec.Template.Spec.Containers[0].SecurityContext
				assert.Equal(t, expected, deployment.Spec.Template.Spec.Containers[0].SecurityContext)
			},
		},
		{
			name: "securityContext configured overrides existing context on all containers",
			securityCtx: &corev1.SecurityContext{
				RunAsNonRoot: ptr.To(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			},
			preTransformFn: func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers = append(
					deployment.Spec.Template.Spec.Containers,
					corev1.Container{Name: "sidecar"},
				)
			},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				expected := &corev1.SecurityContext{
					RunAsNonRoot: ptr.To(true),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				}
				containers := deployment.Spec.Template.Spec.Containers
				require.Len(t, containers, 2)
				for _, container := range containers {
					assert.Equal(t, expected, container.SecurityContext)
				}
				assert.NotSame(t, containers[0].SecurityContext, containers[1].SecurityContext)
			},
		},
		{
			name:        "no containers is a no-op",
			securityCtx: &corev1.SecurityContext{RunAsNonRoot: ptr.To(true)},
			preTransformFn: func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers = nil
			},
			validateResult: func(t *testing.T, deployment *appsv1.Deployment) {
				assert.Empty(t, deployment.Spec.Template.Spec.Containers)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deployment := newSecCtxTestDeployment()

			if tc.preTransformFn != nil {
				tc.preTransformFn(deployment)
			}
			result := ApplyContainerSecurityContext(deployment, tc.securityCtx)
			tc.validateResult(t, result)
		})
	}
}
