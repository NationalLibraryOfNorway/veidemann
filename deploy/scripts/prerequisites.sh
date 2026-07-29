#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${HOME:-}" ]]; then
  echo "HOME must be set" >&2
  exit 1
fi

INSTALL_DIR="${HOME}/.local/bin"
BASH_COMPLETION_DIR="${XDG_DATA_HOME:-${HOME}/.local/share}/bash-completion/completions"
mkdir -p "$INSTALL_DIR" "$BASH_COMPLETION_DIR"

case ":${PATH}:" in
  *:"${INSTALL_DIR}":*) ;;
  *)
    echo "Installing tools to ${INSTALL_DIR}; add it to PATH for future shells."
    export PATH="${INSTALL_DIR}:${PATH}"
    ;;
esac

OPSYS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

LINKERD_VERSION=edge-26.6.1
KUBECTL_VERSION=v1.36.3
MINIKUBE_VERSION=v1.38.1
VEIDEMANNCTL_VERSION=0.11.0
HELM_VERSION=v4.2.3
SKAFFOLD_VERSION=v2.24.0
STEP_VERSION=0.30.6

ask() {
  local prompt=$1
  read -rp "${prompt} [y/N] " -n 1 reply || true
  echo
  [[ $reply =~ ^[Yy]$ ]]
}

# Decide whether we should install or upgrade a tool.
# Returns 0 if we SHOULD install, 1 if we should NOT.
need_install() {
  local cmd=$1
  local desired=$2
  local current=$3

  if [[ -z "$current" ]]; then
    ask "${cmd} not found. Install ${cmd} ${desired}?" && return 0 || return 1
  fi

  if [[ "$current" != "$desired" ]]; then
    ask "${cmd} version is ${current}, but we want ${desired}. Install ${cmd} ${desired}?" \
      && return 0 || return 1
  fi

  # already at desired version
  return 1
}

# Create temp dir and cd into it (portable mktemp)
tmp=$(mktemp -d "${TMPDIR:-/tmp}/veidemann-prerequisites.XXXXXX")
pushd "$tmp" >/dev/null || exit 1

clean_temp() {
  # popd might fail if directory stack changed; ignore errors
  popd >/dev/null 2>&1 || true
  rm -rf "$tmp"
}
trap clean_temp EXIT

for CMD in "$@"; do
  case "$CMD" in
    kubectl)
      current=""
      if command -v kubectl >/dev/null 2>&1; then
        current="$(kubectl version --client 2>/dev/null \
          | awk -F': ' '/Client Version/ {print $2}')"
      fi

      if need_install kubectl "$KUBECTL_VERSION" "$current"; then
        echo "Installing kubectl ${KUBECTL_VERSION}"
        curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OPSYS}/${ARCH}/kubectl"
        install kubectl "${INSTALL_DIR}/kubectl"
        "${INSTALL_DIR}/kubectl" completion bash > "${BASH_COMPLETION_DIR}/kubectl"
        rm kubectl
      fi
      ;;

    helm)
      current=""
      if command -v helm >/dev/null 2>&1; then
        current="$(helm version --template '{{.Version}}' 2>/dev/null)"
      fi

      if need_install helm "$HELM_VERSION" "$current"; then
        echo "Installing helm ${HELM_VERSION}"
        curl -L "https://get.helm.sh/helm-${HELM_VERSION}-${OPSYS}-${ARCH}.tar.gz" | tar xz
        install "${OPSYS}-${ARCH}/helm" "${INSTALL_DIR}/helm"
        "${INSTALL_DIR}/helm" completion bash > "${BASH_COMPLETION_DIR}/helm"
        rm -r "${OPSYS}-${ARCH}"
      fi
      ;;

    linkerd)
      current=""
      if command -v linkerd >/dev/null 2>&1; then
        current="$(linkerd version --client --short 2>/dev/null)"
      fi

      if need_install linkerd "$LINKERD_VERSION" "$current"; then
        echo "Installing linkerd ${LINKERD_VERSION}"
        curl -Lo linkerd \
          "https://github.com/linkerd/linkerd2/releases/download/${LINKERD_VERSION}/linkerd2-cli-${LINKERD_VERSION}-${OPSYS}-${ARCH}"
        install linkerd "${INSTALL_DIR}/linkerd"
        "${INSTALL_DIR}/linkerd" completion bash > "${BASH_COMPLETION_DIR}/linkerd"
        rm linkerd
      fi
      ;;

    minikube)
      current=""
      if command -v minikube >/dev/null 2>&1; then
        current="$(minikube version 2>/dev/null \
          | awk -F': ' '/version:/ {print $2}')"
      fi

      if need_install minikube "$MINIKUBE_VERSION" "$current"; then
        echo "Installing minikube ${MINIKUBE_VERSION}"
        curl -Lo minikube \
          "https://storage.googleapis.com/minikube/releases/${MINIKUBE_VERSION}/minikube-${OPSYS}-${ARCH}"
        install minikube "${INSTALL_DIR}/minikube"
        "${INSTALL_DIR}/minikube" completion bash > "${BASH_COMPLETION_DIR}/minikube"
        rm minikube
      fi
      ;;

    veidemannctl)
      current=""
      if command -v veidemannctl >/dev/null 2>&1; then
        # output: veidemannctl version x.y.z, ...
        current="$(veidemannctl --version 2>/dev/null \
          | awk '{print $5}' | sed 's/,//')"
      fi

      if need_install veidemannctl "$VEIDEMANNCTL_VERSION" "$current"; then
        echo "Installing veidemannctl ${VEIDEMANNCTL_VERSION}"
        curl -Lo veidemannctl \
          "https://github.com/NationalLibraryOfNorway/veidemann/releases/download/ctl-v${VEIDEMANNCTL_VERSION}/veidemannctl_${OPSYS}_${ARCH}"
        install veidemannctl "${INSTALL_DIR}/veidemannctl"
        "${INSTALL_DIR}/veidemannctl" completion bash > "${BASH_COMPLETION_DIR}/veidemannctl"
        rm veidemannctl
      fi
      ;;

    skaffold)
      current=""
      if command -v skaffold >/dev/null 2>&1; then
        current="$(skaffold version 2>/dev/null)"
      fi

      if need_install skaffold "$SKAFFOLD_VERSION" "$current"; then
        echo "Installing skaffold ${SKAFFOLD_VERSION}"
        curl -Lo skaffold \
          "https://storage.googleapis.com/skaffold/releases/${SKAFFOLD_VERSION}/skaffold-${OPSYS}-${ARCH}"
        install skaffold "${INSTALL_DIR}/skaffold"
        "${INSTALL_DIR}/skaffold" completion bash > "${BASH_COMPLETION_DIR}/skaffold"
        rm skaffold
      fi
      ;;

    step)
      current=""
      if command -v step >/dev/null 2>&1; then
        current="$(step version 2>/dev/null \
          | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
      fi

      if need_install step "$STEP_VERSION" "$current"; then
        echo "Installing step ${STEP_VERSION}"
        curl -L \
          "https://dl.smallstep.com/gh-release/cli/gh-release-header/v${STEP_VERSION}/step_${OPSYS}_${STEP_VERSION}_${ARCH}.tar.gz" \
          | tar xz
        install "step_${STEP_VERSION}/bin/step" "${INSTALL_DIR}/step"
        cp "step_${STEP_VERSION}/autocomplete/bash_autocomplete" "${BASH_COMPLETION_DIR}/step"
        rm -rf "step_${STEP_VERSION}"
      fi
      ;;

    *)
      echo "Unknown command ${CMD}" >&2
      exit 1
      ;;
  esac
done
