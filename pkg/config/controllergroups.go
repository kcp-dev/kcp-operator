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

package config

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// ControllerGroup is a group of controllers that can be enabled or disabled together.
type ControllerGroup string

const (
	// ControllerGroupConfig groups controllers that are configuring a kcp instance.
	// This includes orchestrating cert-manager resources and compiling the workload CRs.
	ControllerGroupConfig ControllerGroup = "config"

	// ControllerGroupWorkload groups controllers that deploy the compiled workload CRs
	// into compute resources like Deployments and Services.
	ControllerGroupWorkload ControllerGroup = "workload"
)

// AllControllerGroups returns a new set containing all known controller groups.
func AllControllerGroups() sets.Set[ControllerGroup] {
	return sets.New(ControllerGroupConfig, ControllerGroupWorkload)
}

// ParseControllerGroups validates the given controller group names and returns them as a set.
// Unknown names or an empty list result in an error.
func ParseControllerGroups(names []string) (sets.Set[ControllerGroup], error) {
	known := AllControllerGroups()

	parsed := sets.New[ControllerGroup]()
	for _, name := range names {
		group := ControllerGroup(strings.TrimSpace(name))
		if !known.Has(group) {
			return nil, fmt.Errorf("unknown controller group %q, known controller groups are %v", name, sets.List(known))
		}
		parsed.Insert(group)
	}

	if parsed.Len() == 0 {
		return nil, fmt.Errorf("at least one controller group must be enabled, known controller groups are %v", sets.List(known))
	}

	return parsed, nil
}
