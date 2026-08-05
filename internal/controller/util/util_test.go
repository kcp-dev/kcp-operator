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

package util

import (
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func readyCertificate(name string, revision int) *certmanagerv1.Certificate {
	return &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: certmanagerv1.CertificateStatus{
			Revision: ptr.To(revision),
			Conditions: []certmanagerv1.CertificateCondition{{
				Type:   certmanagerv1.CertificateConditionReady,
				Status: certmanagermetav1.ConditionTrue,
			}},
		},
	}
}

func TestCertificateRevisions(t *testing.T) {
	revisions, ready := CertificateRevisions([]*certmanagerv1.Certificate{
		readyCertificate("ca", 1),
		readyCertificate("server", 4),
	})

	if !ready {
		t.Fatal("expected certificates to be ready")
	}

	expected := map[string]string{
		"ca":     "1",
		"server": "4",
	}
	if len(revisions) != len(expected) {
		t.Fatalf("expected %d revisions, got %d", len(expected), len(revisions))
	}
	for k, v := range expected {
		if revisions[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, revisions[k])
		}
	}
}

func TestCertificateRevisionsNotReady(t *testing.T) {
	notReady := &certmanagerv1.Certificate{ObjectMeta: metav1.ObjectMeta{Name: "pending"}}

	if _, ready := CertificateRevisions([]*certmanagerv1.Certificate{readyCertificate("ca", 1), notReady}); ready {
		t.Error("expected ready=false with a pending certificate")
	}

	// Objects captured on the create path are empty and must not count as ready.
	if _, ready := CertificateRevisions([]*certmanagerv1.Certificate{{}}); ready {
		t.Error("expected ready=false with an empty certificate")
	}
}
