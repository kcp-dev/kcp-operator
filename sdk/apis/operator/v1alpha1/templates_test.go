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

package v1alpha1

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// jsonFields returns the set of JSON field names of a struct type,
// flattening inlined embedded structs.
func jsonFields(t *testing.T, typ reflect.Type) map[string]bool {
	t.Helper()

	fields := map[string]bool{}
	for i := range typ.NumField() {
		f := typ.Field(i)
		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")

		if name == "" && strings.Contains(tag, "inline") {
			for inlined := range jsonFields(t, f.Type) {
				fields[inlined] = true
			}
			continue
		}

		if name == "" || name == "-" {
			continue
		}

		fields[name] = true
	}

	return fields
}

func TestTemplateSpecMirrors(t *testing.T) {
	testcases := []struct {
		name     string
		spec     any
		template any
		omitted  []string
	}{
		{
			name:     "RootShard",
			spec:     RootShardSpec{},
			template: RootShardTemplateSpec{},
		},
		{
			name:     "Shard",
			spec:     ShardSpec{},
			template: ShardTemplateSpec{},
		},
		{
			name:     "FrontProxy",
			spec:     FrontProxySpec{},
			template: FrontProxyTemplateSpec{},
			omitted:  []string{"externalHostname"},
		},
		{
			name:     "CacheServer",
			spec:     CacheServerSpec{},
			template: CacheServerTemplateSpec{},
		},
		{
			name:     "VirtualWorkspace",
			spec:     VirtualWorkspaceSpec{},
			template: VirtualWorkspaceTemplateSpec{},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			specFields := jsonFields(t, reflect.TypeOf(testcase.spec))
			templateFields := jsonFields(t, reflect.TypeOf(testcase.template))

			for f := range specFields {
				if !templateFields[f] && !slices.Contains(testcase.omitted, f) {
					t.Errorf("spec field %q missing from template", f)
				}
			}

			for f := range templateFields {
				if !specFields[f] {
					t.Errorf("template field %q does not exist in spec", f)
				}
			}
		})
	}
}
