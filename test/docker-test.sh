#!/usr/bin/env bash
# Test runner for Docker containers (Ubuntu 20.04, 22.04, 24.04)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOTFILES_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "\n${BLUE}==>${NC} ${YELLOW}$1${NC}\n"
}


# Test in Docker

test_ubuntu_version() {
    local version=$1
    local dockerfile="$SCRIPT_DIR/docker/Dockerfile.ubuntu-$version"

    if [ ! -f "$dockerfile" ]; then
        log_error "Dockerfile not found: $dockerfile"
        return 1
    fi

    log_step "Testing on Ubuntu $version"

    # Build Docker image
    log_info "Building Docker image for Ubuntu $version..."
    if docker build -f "$dockerfile" -t "dotfiles-test-ubuntu-$version" "$DOTFILES_DIR"; then
        log_success "Docker image built successfully"
    else
        log_error "Failed to build Docker image"
        return 1
    fi

    # Run tests in container
    log_info "Running tests in Docker container..."
    if docker run --rm "dotfiles-test-ubuntu-$version"; then
        log_success "Tests passed on Ubuntu $version"
        return 0
    else
        log_error "Tests failed on Ubuntu $version"
        return 1
    fi
}


# Main

main() {
    local version="${1:-22.04}"

    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        log_error "Docker not found. Please install Docker to run these tests."
        exit 1
    fi

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon not running. Please start Docker."
        exit 1
    fi

    echo "Running Docker tests"
    echo ""

    case "$version" in
        20.04|22.04|24.04)
            test_ubuntu_version "$version"
            ;;
        all)
            local failed=0
            for ver in 20.04 22.04 24.04; do
                if ! test_ubuntu_version "$ver"; then
                    ((failed++))
                fi
            done

            if [ $failed -eq 0 ]; then
                log_success "All Docker tests passed!"
                exit 0
            else
                log_error "$failed Docker test(s) failed"
                exit 1
            fi
            ;;
        *)
            log_error "Invalid Ubuntu version: $version"
            echo "Usage: $0 [20.04|22.04|24.04|all]"
            exit 1
            ;;
    esac
}

main "$@"
