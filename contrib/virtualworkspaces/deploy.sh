#!/usr/bin/env bash

# Copyright 2026 The kcp Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib.sh"

VW_DIR="$CONTRIB_DIR/virtualworkspaces"

# vw_function_name <dir name> — access-vw -> vw_access_vw_deploy.
vw_function_name() {
  echo "vw_${1//-/_}_deploy"
}

vw_deploy_one() {
  local name="$1"
  local script="$VW_DIR/$name/deploy.sh"

  [[ -f "$script" ]] || die "no such virtual workspace: $name (expected $script)"

  # shellcheck disable=SC1090
  source "$script"

  local fn
  fn="$(vw_function_name "$name")"
  declare -F "$fn" >/dev/null || die "$script does not define $fn"
  "$fn"
}

vw_deploy_all() {
  local requested=("$@")
  if [[ ${#requested[@]} -eq 0 ]]; then
    read -r -a requested <<< "$VIRTUAL_WORKSPACES"
  fi

  require_tools kubectl docker envsubst python3 curl

  [[ -f "$KUBECONFIG_OUT" ]] || die "no admin kubeconfig at $KUBECONFIG_OUT.
  Run contrib/kcp/deploy.sh first, or point KUBECONFIG_OUT at an existing one."

  local name
  for name in "${requested[@]}"; do
    log "=== virtual workspace: $name"
    vw_deploy_one "$name"
  done
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  vw_deploy_all "$@"
fi
