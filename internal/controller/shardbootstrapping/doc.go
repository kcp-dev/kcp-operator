/*
Copyright 2025 The KCP Authors.

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

// Package shardbootstrapping is responsible for creating operator-specific
// resources in all kcp (root)shards.
//
// This is required because the operator will, in other controllers, manage
// RBAC for Kubeconfigs. Since kubeconfigs can target front-proxies instead of
// shards, and then request to provision permissions in a target workspace,
// the operator will have to connect through the front-proxy to reach the
// target workspace (to use the front-proxy's index to resolve the target
// shard). The front-proxy will however by default drop system groups from the
// authentication information, so if the operator tried to use a client cert
// with "system:masters" in it, the front-proxy would drop it and we end up
// authenticated but permissionless on the target shard.
//
// To avoid this scenario, this controller will bootstrap a special
// ClusterRoleBinding on each shard in the system:admin cluster, binding a
// custom group. The kubeconfig controller will then use a certificate with
// that group to truly authenticate at any endpoint (shard or front-proxy) with
// system permissions.
package shardbootstrapping
