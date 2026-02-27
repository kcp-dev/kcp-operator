/*
Copyright 2025 The kcp Authors.

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
	"fmt"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

// GetLoggingArgs returns the command line arguments for logging configuration.
func GetLoggingArgs(spec *operatorv1alpha1.LoggingSpec) []string {
	if spec == nil {
		return nil
	}

	var args []string
	if spec.Level != 0 {
		args = append(args, fmt.Sprintf("-v=%d", spec.Level))
	}

	return args
}
