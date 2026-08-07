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

package utils

import (
	"os"
	"testing"
)

const (
	// TopologySingle is a single cluster running both controller groups.
	TopologySingle = "single"
	// TopologyConfigWorkload is a config and a workload cluster, each running only its controller group.
	TopologyConfigWorkload = "config-workload"
)

func Topology() string {
	if topology := os.Getenv("E2E_TOPOLOGY"); topology != "" {
		return topology
	}

	return ""
}

func SkipUnlessTopology(t *testing.T, topology string) {
	if current := Topology(); current != topology {
		t.Skipf("Test requires the %s topology, but the harness is running %s.", topology, current)
	}
}
