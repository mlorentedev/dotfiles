#!/usr/bin/env bash

# Install Kubernetes tools: kubectl, k9s, helm, kubectx, stern




set -e

# Source utils if available
if [ -f "$(dirname "$0")/../scripts/.local/bin/utils" ]; then
    # shellcheck source=scripts/.local/bin/utils
    source "$(dirname "$0")/../scripts/.local/bin/utils"
else
    log_info() { echo "[INFO] $1"; }
    log_success() { echo "[SUCCESS] $1"; }
    log_warning() { echo "[WARNING] $1"; }
    log_error() { echo "[ERROR] $1"; }
fi

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

command_exists() {
    command -v "$1" &> /dev/null
}

get_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}


# kubectl

install_kubectl() {
    if command_exists kubectl; then
        log_info "kubectl already installed: $(kubectl version --client --short 2>/dev/null || kubectl version --client)"
        return 0
    fi

    log_info "Installing kubectl..."

    local arch
    local version
    arch=$(get_arch)
    version=$(curl -sL https://dl.k8s.io/release/stable.txt)
    local url="https://dl.k8s.io/release/${version}/bin/linux/${arch}/kubectl"

    curl -sL "$url" -o "$INSTALL_DIR/kubectl"
    chmod +x "$INSTALL_DIR/kubectl"

    log_success "kubectl installed: $version"
}


# k9s

install_k9s() {
    if command_exists k9s; then
        log_info "k9s already installed: $(k9s version --short 2>/dev/null || k9s version)"
        return 0
    fi

    log_info "Installing k9s (Kubernetes TUI)..."

    local arch
    arch=$(get_arch)
    local version="v0.31.7"
    local url="https://github.com/derailed/k9s/releases/download/${version}/k9s_Linux_${arch}.tar.gz"

    mkdir -p /tmp/k9s
    curl -sL "$url" | tar xz -C /tmp/k9s
    mv /tmp/k9s/k9s "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/k9s"
    rm -rf /tmp/k9s

    log_success "k9s installed"
}


# Helm

install_helm() {
    if command_exists helm; then
        log_info "helm already installed: $(helm version --short)"
        return 0
    fi

    log_info "Installing helm..."

    curl -sS https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | HELM_INSTALL_DIR="$INSTALL_DIR" bash

    log_success "helm installed"
}


# kubectx and kubens

install_kubectx() {
    if command_exists kubectx && command_exists kubens; then
        log_info "kubectx and kubens already installed"
        return 0
    fi

    log_info "Installing kubectx and kubens..."

    local version="v0.9.5"

    curl -sL "https://raw.githubusercontent.com/ahmetb/kubectx/${version}/kubectx" -o "$INSTALL_DIR/kubectx"
    curl -sL "https://raw.githubusercontent.com/ahmetb/kubectx/${version}/kubens" -o "$INSTALL_DIR/kubens"

    chmod +x "$INSTALL_DIR/kubectx" "$INSTALL_DIR/kubens"

    log_success "kubectx and kubens installed"
}


# stern (log tailing)

install_stern() {
    if command_exists stern; then
        log_info "stern already installed: $(stern --version)"
        return 0
    fi

    log_info "Installing stern (log tailing)..."

    local arch
    arch=$(get_arch)
    local version="v1.28.0"
    local url="https://github.com/stern/stern/releases/download/${version}/stern_${version#v}_linux_${arch}.tar.gz"

    mkdir -p /tmp/stern
    curl -sL "$url" | tar xz -C /tmp/stern
    mv /tmp/stern/stern "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/stern"
    rm -rf /tmp/stern

    log_success "stern installed"
}


# Main

main() {
    log_info "Installing Kubernetes tools to $INSTALL_DIR..."

    install_kubectl
    install_k9s
    install_helm
    install_kubectx
    install_stern

    log_success "Kubernetes tools installation complete!"
}

main "$@"
