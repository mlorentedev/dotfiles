#!/usr/bin/env bash

# Install shell tools: eza, bat, fzf, ripgrep, zoxide, starship, direnv




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


# Helper Functions

command_exists() {
    command -v "$1" &> /dev/null
}

get_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64) echo "x86_64" ;;
        aarch64|arm64) echo "aarch64" ;;
        *) echo "unknown" ;;
    esac
}

get_os() {
    local os
    os=$(uname -s)
    case "$os" in
        Linux) echo "linux" ;;
        Darwin) echo "macos" ;;
        *) echo "unknown" ;;
    esac
}


# Tool Installers


install_eza() {
    if command_exists eza; then
        log_info "eza already installed: $(eza --version | head -1)"
        return 0
    fi

    log_info "Installing eza (modern ls replacement)..."

    local os
    local arch
    os=$(get_os)
    arch=$(get_arch)

    if [ "$os" = "linux" ] && [ "$arch" = "x86_64" ]; then
        local version="0.18.0"
        local url="https://github.com/eza-community/eza/releases/download/v${version}/eza_x86_64-unknown-linux-musl.tar.gz"

        curl -sL "$url" | tar xz -C "$INSTALL_DIR" eza
        chmod +x "$INSTALL_DIR/eza"
        log_success "eza installed"
    else
        log_warning "No binary available for $os-$arch, skipping eza"
    fi
}

install_bat() {
    if command_exists bat || command_exists batcat; then
        log_info "bat already installed"
        return 0
    fi

    log_info "Installing bat (cat with syntax highlighting)..."

    local os
    local arch
    os=$(get_os)
    arch=$(get_arch)

    if [ "$os" = "linux" ] && [ "$arch" = "x86_64" ]; then
        local version="0.24.0"
        local url="https://github.com/sharkdp/bat/releases/download/v${version}/bat-v${version}-x86_64-unknown-linux-musl.tar.gz"

        mkdir -p /tmp/bat
        curl -sL "$url" | tar xz -C /tmp/bat --strip-components=1
        mv /tmp/bat/bat "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/bat"
        rm -rf /tmp/bat
        log_success "bat installed"
    else
        log_warning "No binary available for $os-$arch, skipping bat"
    fi
}

install_fzf() {
    if command_exists fzf; then
        log_info "fzf already installed: $(fzf --version)"
        return 0
    fi

    log_info "Installing fzf (fuzzy finder)..."

    git clone --depth 1 https://github.com/junegunn/fzf.git "$HOME/.fzf"
    "$HOME/.fzf/install" --bin

    # Link to .local/bin
    ln -sf "$HOME/.fzf/bin/fzf" "$INSTALL_DIR/fzf"

    log_success "fzf installed"
}

install_ripgrep() {
    if command_exists rg; then
        log_info "ripgrep already installed: $(rg --version | head -1)"
        return 0
    fi

    log_info "Installing ripgrep (fast grep)..."

    local os
    local arch
    os=$(get_os)
    arch=$(get_arch)

    if [ "$os" = "linux" ] && [ "$arch" = "x86_64" ]; then
        local version="14.1.0"
        local url="https://github.com/BurntSushi/ripgrep/releases/download/${version}/ripgrep-${version}-x86_64-unknown-linux-musl.tar.gz"

        mkdir -p /tmp/rg
        curl -sL "$url" | tar xz -C /tmp/rg --strip-components=1
        mv /tmp/rg/rg "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/rg"
        rm -rf /tmp/rg
        log_success "ripgrep installed"
    else
        log_warning "No binary available for $os-$arch, skipping ripgrep"
    fi
}

install_zoxide() {
    if command_exists zoxide; then
        log_info "zoxide already installed: $(zoxide --version)"
        return 0
    fi

    log_info "Installing zoxide (smart cd)..."

    curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash

    log_success "zoxide installed"
}

install_starship() {
    if command_exists starship; then
        log_info "starship already installed: $(starship --version)"
        return 0
    fi

    log_info "Installing starship (modern prompt)..."

    curl -sS https://starship.rs/install.sh | sh -s -- -y -b "$INSTALL_DIR"

    log_success "starship installed"
}

install_direnv() {
    if command_exists direnv; then
        log_info "direnv already installed: $(direnv version)"
        return 0
    fi

    log_info "Installing direnv (environment switcher)..."

    local os
    local arch
    os=$(get_os)
    arch=$(get_arch)

    if [ "$os" = "linux" ]; then
        local url="https://direnv.net/install.sh"
        curl -sfL "$url" | bash
        mv ./direnv "$INSTALL_DIR/"
        log_success "direnv installed"
    else
        log_warning "Automatic installation not supported for $os, please install manually"
    fi
}

install_age() {
    if command_exists age; then
        log_info "age already installed: $(age --version 2>&1 | head -1)"
        return 0
    fi

    log_info "Installing age (encryption tool)..."

    local os
    local arch
    os=$(get_os)
    arch=$(get_arch)

    if [ "$os" = "linux" ] && [ "$arch" = "x86_64" ]; then
        local version="v1.1.1"
        local url="https://github.com/FiloSottile/age/releases/download/${version}/age-${version}-linux-amd64.tar.gz"

        mkdir -p /tmp/age
        curl -sL "$url" | tar xz -C /tmp/age --strip-components=1
        mv /tmp/age/age "$INSTALL_DIR/"
        mv /tmp/age/age-keygen "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/age" "$INSTALL_DIR/age-keygen"
        rm -rf /tmp/age
        log_success "age installed"
    else
        log_warning "No binary available for $os-$arch, skipping age"
    fi
}


# Main

main() {
    log_info "Installing shell enhancement tools to $INSTALL_DIR..."

    install_eza
    install_bat
    install_fzf
    install_ripgrep
    install_zoxide
    install_starship
    install_direnv
    install_age

    log_success "Shell tools installation complete!"
    log_info "Make sure $INSTALL_DIR is in your PATH"
}

main "$@"
