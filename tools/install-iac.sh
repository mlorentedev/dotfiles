#!/usr/bin/env bash

# Install IaC tools: terraform, ansible, ansible-lint




set -e

# Source utils if available
# shellcheck disable=SC1091
if [ -f "$(dirname "$0")/../scripts/.local/bin/utils" ]; then
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


# Terraform

install_terraform() {
    if command_exists terraform; then
        log_info "terraform already installed: $(terraform version | head -1)"
        return 0
    fi

    log_info "Installing terraform..."

    local arch
    arch=$(get_arch)
    local version="1.7.0"
    local url="https://releases.hashicorp.com/terraform/${version}/terraform_${version}_linux_${arch}.zip"

    mkdir -p /tmp/terraform
    curl -sL "$url" -o /tmp/terraform/terraform.zip
    unzip -q /tmp/terraform/terraform.zip -d /tmp/terraform
    mv /tmp/terraform/terraform "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/terraform"
    rm -rf /tmp/terraform

    log_success "terraform installed: $version"
}


# Ansible

install_ansible() {
    if command_exists ansible; then
        log_info "ansible already installed: $(ansible --version | head -1)"
        return 0
    fi

    log_info "Installing ansible..."

    # Install using pip to ~/.local/bin
    if command_exists pip3; then
        pip3 install --user ansible
        log_success "ansible installed"
    elif command_exists pip; then
        pip install --user ansible
        log_success "ansible installed"
    else
        log_warning "pip not found. Please install Python and pip first"
        log_info "  sudo apt-get install python3-pip"
        return 1
    fi
}


# Ansible Lint

install_ansible_lint() {
    if command_exists ansible-lint; then
        log_info "ansible-lint already installed: $(ansible-lint --version)"
        return 0
    fi

    log_info "Installing ansible-lint..."

    if command_exists pip3; then
        pip3 install --user ansible-lint
        log_success "ansible-lint installed"
    elif command_exists pip; then
        pip install --user ansible-lint
        log_success "ansible-lint installed"
    else
        log_warning "pip not found, skipping ansible-lint"
        return 1
    fi
}


# Main

main() {
    log_info "Installing Infrastructure as Code tools..."

    install_terraform
    install_ansible
    install_ansible_lint

    log_success "IaC tools installation complete!"
}

main "$@"
