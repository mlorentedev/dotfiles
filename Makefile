.PHONY: help install minimal tools all test test-docker lint clean backup uninstall update check

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)Dotfiles Makefile$(NC)"
	@echo ""
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'
	@echo ""

install: minimal ## Alias for minimal installation

minimal: ## Install minimal configuration (configs only)
	@echo "$(BLUE)Installing minimal configuration...$(NC)"
	@chmod +x bootstrap.sh
	@./bootstrap.sh --minimal
	@echo "$(GREEN)✓ Minimal installation complete$(NC)"

tools: ## Install with DevOps toolchain
	@echo "$(BLUE)Installing with tools...$(NC)"
	@chmod +x bootstrap.sh
	@./bootstrap.sh --tools
	@echo "$(GREEN)✓ Tools installation complete$(NC)"

all: ## Install everything including secrets
	@echo "$(BLUE)Installing everything...$(NC)"
	@chmod +x bootstrap.sh
	@./bootstrap.sh --all
	@echo "$(GREEN)✓ Full installation complete$(NC)"

test: ## Run all tests
	@echo "$(BLUE)Running all tests...$(NC)"
	@chmod +x test/run-all-tests.sh
	@./test/run-all-tests.sh

test-stow: ## Test stow functionality
	@echo "$(BLUE)Testing stow...$(NC)"
	@chmod +x test/test-stow.sh
	@./test/test-stow.sh

test-aliases: ## Test shell aliases
	@echo "$(BLUE)Testing aliases...$(NC)"
	@chmod +x test/test-aliases.sh
	@./test/test-aliases.sh

test-path: ## Test PATH configuration
	@echo "$(BLUE)Testing PATH...$(NC)"
	@chmod +x test/test-path.sh
	@./test/test-path.sh

test-docker: ## Run tests in Docker (Ubuntu 22.04)
	@echo "$(BLUE)Running Docker tests...$(NC)"
	@chmod +x test/docker-test.sh
	@./test/docker-test.sh 22.04

test-docker-all: ## Run tests in Docker (all Ubuntu versions)
	@echo "$(BLUE)Running Docker tests on all versions...$(NC)"
	@chmod +x test/docker-test.sh
	@./test/docker-test.sh all

lint: ## Run shellcheck on all scripts
	@echo "$(BLUE)Running shellcheck...$(NC)"
	@if command -v shellcheck > /dev/null; then \
		find . -name "*.sh" -type f -exec shellcheck {} + ; \
		shellcheck bootstrap.sh ; \
		echo "$(GREEN)✓ Shellcheck passed$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ shellcheck not installed$(NC)" ; \
	fi

clean: ## Remove all symlinks created by stow
	@echo "$(BLUE)Removing symlinks...$(NC)"
	@cd $(CURDIR) && stow -D bash zsh git shell-common scripts 2>/dev/null || true
	@echo "$(GREEN)✓ Symlinks removed$(NC)"

backup: ## Backup existing dotfiles
	@echo "$(BLUE)Creating backup...$(NC)"
	@mkdir -p ~/.dotfiles-backup-$(shell date +%Y%m%d-%H%M%S)
	@for file in .bashrc .bash_profile .zshrc .gitconfig .profile; do \
		if [ -f ~/$$file ] && [ ! -L ~/$$file ]; then \
			cp ~/$$file ~/.dotfiles-backup-$(shell date +%Y%m%d-%H%M%S)/ 2>/dev/null || true ; \
		fi ; \
	done
	@echo "$(GREEN)✓ Backup created$(NC)"

uninstall: clean ## Uninstall dotfiles (remove symlinks)
	@echo "$(BLUE)Uninstalling dotfiles...$(NC)"
	@echo "$(YELLOW)Symlinks removed. To restore backups, check ~/.dotfiles-backup-*$(NC)"

update: ## Pull latest changes and reinstall
	@echo "$(BLUE)Updating dotfiles...$(NC)"
	@git pull origin main || git pull origin master
	@./bootstrap.sh --minimal
	@echo "$(GREEN)✓ Update complete$(NC)"

check: ## Check system dependencies
	@echo "$(BLUE)Checking dependencies...$(NC)"
	@echo ""
	@echo "Required:"
	@command -v git > /dev/null && echo "  $(GREEN)✓$(NC) git" || echo "  $(YELLOW)✗$(NC) git"
	@command -v bash > /dev/null && echo "  $(GREEN)✓$(NC) bash" || echo "  $(YELLOW)✗$(NC) bash"
	@command -v curl > /dev/null && echo "  $(GREEN)✓$(NC) curl" || echo "  $(YELLOW)✗$(NC) curl"
	@command -v stow > /dev/null && echo "  $(GREEN)✓$(NC) stow" || echo "  $(YELLOW)✗$(NC) stow (will be installed)"
	@echo ""
	@echo "Optional:"
	@command -v zsh > /dev/null && echo "  $(GREEN)✓$(NC) zsh" || echo "  $(YELLOW)✗$(NC) zsh"
	@command -v eza > /dev/null && echo "  $(GREEN)✓$(NC) eza" || echo "  $(YELLOW)✗$(NC) eza"
	@command -v bat > /dev/null && echo "  $(GREEN)✓$(NC) bat" || echo "  $(YELLOW)✗$(NC) bat"
	@command -v fzf > /dev/null && echo "  $(GREEN)✓$(NC) fzf" || echo "  $(YELLOW)✗$(NC) fzf"
	@command -v zoxide > /dev/null && echo "  $(GREEN)✓$(NC) zoxide" || echo "  $(YELLOW)✗$(NC) zoxide"
	@command -v direnv > /dev/null && echo "  $(GREEN)✓$(NC) direnv" || echo "  $(YELLOW)✗$(NC) direnv"
	@command -v starship > /dev/null && echo "  $(GREEN)✓$(NC) starship" || echo "  $(YELLOW)✗$(NC) starship"
	@echo ""
	@echo "DevOps tools:"
	@command -v docker > /dev/null && echo "  $(GREEN)✓$(NC) docker" || echo "  $(YELLOW)✗$(NC) docker"
	@command -v kubectl > /dev/null && echo "  $(GREEN)✓$(NC) kubectl" || echo "  $(YELLOW)✗$(NC) kubectl"
	@command -v helm > /dev/null && echo "  $(GREEN)✓$(NC) helm" || echo "  $(YELLOW)✗$(NC) helm"
	@command -v terraform > /dev/null && echo "  $(GREEN)✓$(NC) terraform" || echo "  $(YELLOW)✗$(NC) terraform"
	@command -v ansible > /dev/null && echo "  $(GREEN)✓$(NC) ansible" || echo "  $(YELLOW)✗$(NC) ansible"
	@echo ""
