package main

import (
	"os"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
)

// version is overridden at release time by goreleaser via
// -ldflags "-X main.version=<tag>". It stays in package main — not in
// internal/cmd — so that ldflags path never silently breaks (CLI-002 R2).
var version = "dev"

func main() {
	if err := cmd.New(version).Execute(); err != nil {
		os.Exit(1)
	}
}
