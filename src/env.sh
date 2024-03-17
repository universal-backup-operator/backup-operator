#!/usr/bin/env bash

SCRIPT_NAME="$0"
export KUBECONFIG="$(dirname "$(readlink -f "${SCRIPT_NAME}")")/kubeconfig.yaml"
export KIND_NAME="backup-operator"
export KIND_CONFIG='
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
      endpoint = ["http://registry-mirror:5000"]
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry.local"]
      endpoint = ["http://registry-local:5000"]
    [plugins."io.containerd.grpc.v1.cri".registry.configs."registry-mirror:5000".tls]
      insecure_skip_verify = true
    [plugins."io.containerd.grpc.v1.cri".registry.configs."registry-local:5000".tls]
      insecure_skip_verify = true 
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30009
    hostPort: 9000
    listenAddress: "127.0.0.1"
'

kind_create() {
  if ! kind create cluster --name "${KIND_NAME}" --config <(echo "${KIND_CONFIG}"); then
    return
  fi
  kind export kubeconfig --name "${KIND_NAME}" --kubeconfig "${KUBECONFIG}"
  chmod 0600 "${KUBECONFIG}"
  kubectl config set-context --current --namespace=default
  
  docker run -d --name registry-mirror --net=kind \
    -v registry-mirror:/var/lib/registry --restart=always \
    -e REGISTRY_PROXY_REMOTEURL=https://registry-1.docker.io \
    registry:2 || true
  docker run -d --name registry-local --net=kind \
    -v registry-local:/var/lib/registry --restart=always \
    -p 127.0.0.1:5000:5000 registry:2 || true
}

kind_delete() {
  kind delete cluster --name "${KIND_NAME}"
  rm -f "${KUBECONFIG}"
  docker rm -f registry-mirror registry-local || true
}
