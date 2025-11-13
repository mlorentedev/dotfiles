#!/usr/bin/env bash
# Dotfiles bootstrap - install configs using GNU Stow
# Usage: ./bootstrap.sh [--minimal|--tools|--all]

set -e

# Configuration
DOTFILES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="$HOME/.dotfiles-backup-$(date +%Y%m%d-%H%M%S)"
LOG_FILE="$DOTFILES_DIR/bootstrap.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'
# Logging Functions

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_step() {
    echo -e "\n${PURPLE}==>${NC} ${CYAN}$1${NC}\n" | tee -a "$LOG_FILE"
}
# System Detection

detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if [ -f /etc/os-release ]; then
            # shellcheck source=/dev/null
            . /etc/os-release
            OS=$ID
            OS_VERSION=$VERSION_ID
        else
            OS="unknown"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
    else
        OS="unknown"
    fi
    log_info "Detected OS: $OS $OS_VERSION"
}
# Dependency Checks

command_exists() {
    command -v "$1" &> /dev/null
}

check_required_commands() {
    local missing=()

    for cmd in git curl bash; do
        if ! command_exists "$cmd"; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required commands: ${missing[*]}"
        log_error "Please install them and try again"
        exit 1
    fi
}
# Backup Existing Files

backup_existing_files() {
    log_step "Backing up existing dotfiles"

    local files_to_backup=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.zshrc"
        "$HOME/.gitconfig"
        "$HOME/.profile"
    )

    local backed_up=0
    for file in "${files_to_backup[@]}"; do
        if [ -f "$file" ] && [ ! -L "$file" ]; then
            mkdir -p "$BACKUP_DIR"
            cp "$file" "$BACKUP_DIR/"
            log_info "Backed up: $file"
            ((backed_up++))
        fi
    done

    if [ $backed_up -gt 0 ]; then
        log_success "Backed up $backed_up files to $BACKUP_DIR"
    else
        log_info "No existing files to backup"
    fi
}
# Install GNU Stow

install_stow() {
    if command_exists stow; then
        log_info "GNU Stow already installed: $(stow --version | head -1)"
        return 0
    fi

    log_step "Installing GNU Stow"

    case "$OS" in
        ubuntu|debian)
            if command_exists sudo; then
                sudo apt-get update
                sudo apt-get install -y stow
            else
                log_error "sudo not available and stow not installed"
                log_error "Please install stow manually: apt-get install stow"
                exit 1
            fi
            ;;
        fedora|rhel|centos)
            if command_exists sudo; then
                sudo dnf install -y stow || sudo yum install -y stow
            else
                log_error "sudo not available and stow not installed"
                exit 1
            fi
            ;;
        macos)
            if command_exists brew; then
                brew install stow
            else
                log_error "Homebrew not found. Please install: https://brew.sh"
                exit 1
            fi
            ;;
        *)
            log_error "Unsupported OS for automatic stow installation: $OS"
            log_error "Please install stow manually"
            exit 1
            ;;
    esac

    log_success "GNU Stow installed"
}
# Stow Modules

stow_module() {
    local module=$1
    local module_dir="$DOTFILES_DIR/$module"

    if [ ! -d "$module_dir" ]; then
        log_warning "Module not found: $module"
        return 1
    fi

    log_info "Stowing module: $module"

    cd "$DOTFILES_DIR"
    if stow -v "$module" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Module stowed: $module"
        return 0
    else
        log_error "Failed to stow module: $module"
        return 1
    fi
}

stow_minimal_modules() {
    log_step "Installing minimal configuration"

    local modules=(
        "bash"
        "zsh"
        "git"
        "shell-common"
        "scripts"
    )

    for module in "${modules[@]}"; do
        stow_module "$module"
    done
}

stow_tools_modules() {
    log_step "Installing tools configuration"

    local modules=(
        "starship"
        "direnv"
    )

    for module in "${modules[@]}"; do
        stow_module "$module"
    done
}
# Tool Installation

install_essential_tools() {
    log_step "Installing essential tools"

    # Create ~/.local/bin if it doesn't exist
    mkdir -p "$HOME/.local/bin"

    # Install tools using install scripts
    if [ -f "$DOTFILES_DIR/tools/install-shell.sh" ]; then
        bash "$DOTFILES_DIR/tools/install-shell.sh"
    fi
}

install_devops_tools() {
    log_step "Installing DevOps toolchain"

    local tool_scripts=(
        "install-containers.sh"
        "install-kubernetes.sh"
        "install-iac.sh"
    )

    for script in "${tool_scripts[@]}"; do
        if [ -f "$DOTFILES_DIR/tools/$script" ]; then
            bash "$DOTFILES_DIR/tools/$script"
        else
            log_warning "Tool script not found: $script"
        fi
    done
}
# Setup Oh My Zsh

setup_oh_my_zsh() {
    if [ -d "$HOME/.oh-my-zsh" ]; then
        log_info "Oh My Zsh already installed"
        return 0
    fi

    log_step "Installing Oh My Zsh"

    # Install Oh My Zsh non-interactively
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended

    log_success "Oh My Zsh installed"
}
# Post-installation

post_install() {
    log_step "Post-installation setup"

    # Add ~/.local/bin to current PATH if not already there
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        export PATH="$HOME/.local/bin:$PATH"
    fi

    # Make scripts executable
    if [ -d "$HOME/.local/bin" ]; then
        chmod +x "$HOME/.local/bin"/* 2>/dev/null || true
    fi

    log_success "Post-installation complete"
}
# Print Summary

print_summary() {
    echo ""
    log_success "Installation complete"
    echo ""
    echo "Next steps:"
    echo "  1. Restart shell or source config files"
    echo "  2. Run 'make test' to verify installation"
    echo ""
    if [ -d "$BACKUP_DIR" ]; then
        echo "Backup: $BACKUP_DIR"
    fi
    echo "Log: $LOG_FILE"
    echo ""
}
# Main Installation Flow

main() {
    local mode="${1:---minimal}"

    echo "Dotfiles Bootstrap"
    echo "Bootstrap started at $(date)" > "$LOG_FILE"

    # System detection
    detect_os
    check_required_commands

    # Backup existing files
    backup_existing_files

    # Install Stow
    install_stow

    # Install based on mode
    case "$mode" in
        --minimal)
            log_info "Mode: Minimal (configs only)"
            stow_minimal_modules
            setup_oh_my_zsh
            ;;
        --tools)
            log_info "Mode: Tools (configs + DevOps tools)"
            stow_minimal_modules
            stow_tools_modules
            setup_oh_my_zsh
            install_essential_tools
            install_devops_tools
            ;;
        --all)
            log_info "Mode: All (everything including secrets)"
            stow_minimal_modules
            stow_tools_modules
            setup_oh_my_zsh
            install_essential_tools
            install_devops_tools

            # Stow secrets if available
            if [ -d "$DOTFILES_DIR/secrets" ]; then
                stow_module "secrets"
            fi
            ;;
        --help)
            echo "Usage: $0 [--minimal|--tools|--all|--help]"
            echo ""
            echo "Options:"
            echo "  --minimal   Install only shell configs and scripts"
            echo "  --tools     Install configs + DevOps toolchain"
            echo "  --all       Install everything including secrets"
            echo "  --help      Show this help message"
            exit 0
            ;;
        *)
            log_error "Invalid option: $mode"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac

    # Post-installation
    post_install

    # Print summary
    print_summary
}

# Run main function
main "$@"
