package main

import (
	goerrors "errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/cmd"
	"github.com/mlorentedev/dotfiles/cli/internal/errors"
	"github.com/spf13/cobra"
)

var version = "dev"

// commit is stamped by goreleaser (-X main.commit). It stays EMPTY for a source
// build (`go build ./cmd/dotf`), and that emptiness is meaningful rather than a
// gap: a source build has no fixed provenance — the tree it came from may have
// moved or never been committed — so the honest answer is "unknown", not a
// guess. `dotf doctor` reads the difference and reports the two cases apart.
var commit = ""

func main() {
	info, _ := debug.ReadBuildInfo()
	rootCmd := cmd.New(resolveVersion(version, info), commit)
	os.Exit(run(rootCmd, os.Stderr))
}

// resolveVersion reports what `dotf version` prints when goreleaser did not
// stamp -X main.version. A binary built by `go install <module>@<query>` carries
// the module version Go resolved, and nothing else in it says which code it is,
// so that version is reported (without its "v", like a release).
//
// A build from a checkout keeps "dev". Go marks those with vcs.* settings, or
// with "(devel)" when it cannot read the repository (a linked worktree). "dev"
// is a contract: install-dotf.{sh,ps1} skip replacing a source build on it, and
// doctor skips the pin check for it. The commit stays empty either way. A module
// version holds at most a 12-character hash, and doctor's provenance check reads
// only a full stamp.
func resolveVersion(ldflag string, info *debug.BuildInfo) string {
	if ldflag != "dev" || info == nil {
		return ldflag
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return ldflag
	}
	for _, s := range info.Settings {
		if strings.HasPrefix(s.Key, "vcs.") {
			return ldflag
		}
	}
	return strings.TrimPrefix(v, "v")
}

func run(rootCmd *cobra.Command, stderr io.Writer) int {
	// Suppress Cobra's AUTOMATIC error printing — its "Error: context: ..."
	// wrapper around a TerminalFailure — while leaving the command's own stderr
	// pointed at the real one.
	//
	// This was `rootCmd.SetErr(io.Discard)`, which achieved the first half by
	// discarding EVERYTHING: `cmd.ErrOrStderr()` resolves through the root, so
	// every deliberate diagnostic any subcommand writes went to the void.
	// Measured on d4ea0f5: 16 call sites across 9 command files, including all
	// four `dotf secrets` subcommands, `harness mirror`, `harness presence` — and
	// `harness gate`, which then blocked a tool call with exit 2 and NO reason,
	// leaving the operator no way to know which skill to invoke.
	//
	// SilenceErrors on the root is the mechanism Cobra provides for exactly this:
	// ExecuteC checks the executed command's flag OR the root's before printing.
	// The per-command flag stays readable below, so the "did this command ask for
	// silence" branch is untouched.
	// Forcing the flag on the root pollutes the "did this command ask for
	// silence" read below whenever the root IS the executed command, so its
	// original value is snapshotted and restored for that one case.
	rootSilencedByAuthor := rootCmd.SilenceErrors
	rootCmd.SilenceErrors = true
	rootCmd.SetErr(stderr)

	executedCmd, err := rootCmd.ExecuteC()
	if err != nil {
		var tfe *errors.TerminalFailureError
		if goerrors.As(err, &tfe) {
			// Print exactly the JSON latch, without any Cobra "Error: " or wrapper prefixes.
			_, _ = fmt.Fprintln(stderr, tfe.Error())
		} else {
			// If the specific command didn't request silence, print the error.
			silenced := executedCmd.SilenceErrors
			if executedCmd == rootCmd {
				silenced = rootSilencedByAuthor
			}
			if !silenced {
				_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
			}
		}
		return cmd.ExitCode(err)
	}
	return 0
}
