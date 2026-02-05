#!/usr/bin/env bash

# Copyright 2026 The KCP Authors.
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

start_docker_daemon_ci() {
  # DOCKER_REGISTRY_MIRROR_ADDR is injected via Prow preset;
  # MTU improves the docker-in-docker networking;
  # start-docker.sh is part of the build image.
  DOCKER_REGISTRY_MIRROR="${DOCKER_REGISTRY_MIRROR_ADDR:-}" DOCKER_MTU=1400 start-docker.sh
}

# If a Docker mirror is available, we tunnel it into the
# kind cluster, which has its own containerd daemon.
# kind current does not allow accessing ports on the host
# from within the cluster and also does not allow adding
# custom flags to the `docker run` call it does in the
# background.
# To circumvent this, we use socat to make the TCP-based
# mirror available as a local socket and then mount this
# into the kind container.
# Since containerd does not support sockets, we also start
# a second socat process in the kind container that unwraps
# the socket again and listens on 127.0.0.1:5001, which is
# then used for containerd.
# Being a docker registry does not incur a lot of requests,
# just a few big ones. For this socat seems pretty reliable.
create_kind_cluster() {
  local name="$1"
  local image="$2"

  if [[ -z "${DOCKER_REGISTRY_MIRROR_ADDR:-}" ]]; then
    kind create cluster --name "$name" --image "$image"
  else
    mirrorHost="$(echo "$DOCKER_REGISTRY_MIRROR_ADDR" | sed 's#http://##' | sed 's#/+$##g')"

    # make the registry mirror available as a socket,
    # so we can mount it into the kind cluster
    mkdir -p /mirror
    socat UNIX-LISTEN:/mirror/mirror.sock,fork,reuseaddr,unlink-early,mode=777 TCP4:$mirrorHost &

    cat << EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: "$name"
nodes:
  - role: control-plane
    image: "$image"
    # mount the socket
    extraMounts:
    - hostPath: /mirror
      containerPath: /mirror
containerdConfigPatches:
  # point to the local socat process
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["http://127.0.0.1:5001"]
EOF

    kind create cluster --config kind-config.yaml
  fi
}
