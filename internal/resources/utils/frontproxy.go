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
	"net"
	"strconv"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func GetFrontProxyExternalPort(fp *operatorv1alpha1.FrontProxy, r *operatorv1alpha1.RootShard) int {
	// easy, the user explicitly configured an external port on the FrontProxy itself
	if port := fp.Spec.External.Port; port > 0 {
		return int(port)
	}

	// fallback to deprecated ExternalHostname
	if extName := fp.Spec.ExternalHostname; extName != "" {
		_, port, err := net.SplitHostPort(extName)
		if err == nil {
			parsed, err := strconv.Atoi(port)
			if err == nil {
				return parsed
			}
		}

		// no port was given in the URL; assume the default
		return 6443
	}

	// if nothing valid is configured on the FrontProxy, check the RootShard
	if r != nil {
		if port := r.Spec.External.Port; port > 0 {
			return int(port)
		}
	}

	// last resort, fallback to the default kcp port
	return 6443
}
