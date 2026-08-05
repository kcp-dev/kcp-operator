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

package modifier

import (
	"fmt"
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCapture(t *testing.T) {
	reconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, nil
	}

	existing := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: "cert"},
		Status:     certmanagerv1.CertificateStatus{Revision: ptr.To(3)},
	}

	var certs []*certmanagerv1.Certificate
	if _, err := Capture(&certs)(reconciler)(existing); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 1 {
		t.Fatalf("expected 1 captured object, got %d", len(certs))
	}
	if certs[0].Name != "cert" || ptr.Deref(certs[0].Status.Revision, 0) != 3 {
		t.Errorf("captured object does not match cluster state: %+v", certs[0])
	}
}

func TestCaptureSkipsOtherTypes(t *testing.T) {
	reconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return existing, nil
	}

	var certs []*certmanagerv1.Certificate
	if _, err := Capture(&certs)(reconciler)(&corev1.Secret{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 0 {
		t.Errorf("expected no captured objects, got %d", len(certs))
	}
}

func TestCaptureSkipsOnError(t *testing.T) {
	reconciler := func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		return nil, fmt.Errorf("boom")
	}

	var certs []*certmanagerv1.Certificate
	if _, err := Capture(&certs)(reconciler)(&certmanagerv1.Certificate{}); err == nil {
		t.Fatal("expected error")
	}

	if len(certs) != 0 {
		t.Errorf("expected no captured objects, got %d", len(certs))
	}
}
