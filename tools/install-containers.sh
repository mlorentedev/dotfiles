#!/usr/bin/env bash
# =============================================================================
# Install Container Tools
# =============================================================================
# Installs: docker, docker-compose, lazydocker
# =============================================================================

set -e

# Source utils if available
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

# =============================================================================
# Docker
# =============================================================================
install_docker() {
    if command_exists docker; then
        log_info "Docker already installed: $(docker --version)"
        return 0
    fi

    log_info "Installing Docker..."
    log_warning "Docker requires root privileges. Please install Docker manually:"
    log_info "  Ubuntu/Debian: https://docs.docker.com/engine/install/ubuntu/"
    log_info "  Or run: curl -fsSL https://get.docker.com | sh"
}

# =============================================================================
# Docker Compose
# =============================================================================
install_docker_compose() {
    if command_exists docker-compose || docker compose version &>/dev/null; then
        log_info "Docker Compose already installed"
        return 0
    fi

    log_info "Installing Docker Compose standalone..."

    local version="v2.24.5"
    local url="https://github.com/docker/compose/releases/download/${version}/docker-compose-linux-x86_64"

    curl -sL "$url" -o "$INSTALL_DIR/docker-compose"
    chmod +x "$INSTALL_DIR/docker-compose"

    log_success "Docker Compose installed"
}

# =============================================================================
# Lazydocker
# =============================================================================
install_lazydocker() {
    if command_exists lazydocker; then
        log_info "lazydocker already installed: $(lazydocker --version)"
        return 0
    fi

    log_info "Installing lazydocker (Docker TUI)..."

    curl -sS https://raw.githubusercontent.com/jesseduffield/lazydocker/master/scripts/install_update_linux.sh | bash

    log_success "lazydocker installed"
}

# =============================================================================
# Main
# =============================================================================
main() {
    log_info "Installing container tools..."

    install_docker
    install_docker_compose
    install_lazydocker

    log_success "Container tools installation complete!"
}

main "$@"
