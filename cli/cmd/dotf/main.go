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
		// Not a bare 1: `dotf agent run` distinguishes "no pool could serve
		// this" from "the task failed", and that distinction has to survive the
		// process boundary or a composer cannot act on it. Everything else is
		// untagged and still exits 1.
		os.Exit(cmd.ExitCode(err))
	}
}
