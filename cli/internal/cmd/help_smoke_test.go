package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// allCommands returns root plus every runnable subcommand reachable from it,
// depth first, skipping the auto-generated help/completion commands. The root is
// included so its own help/Long (also user-facing) is guarded too.
func allCommands(root *cobra.Command) []*cobra.Command {
	out := []*cobra.Command{root}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
				continue
			}
			out = append(out, sub)
			walk(sub)
		}
	}
	walk(root)
	return out
}

// argsFor turns a command into the arg path that invokes it (drops the "dotf"
// root token): `dotf secrets backup` -> ["secrets", "backup"].
func argsFor(c *cobra.Command) []string {
	parts := strings.Fields(c.CommandPath())
	if len(parts) > 0 {
		parts = parts[1:]
	}
	return parts
}

// TestEveryCommandHelpRenders smokes the real --help for every subcommand. The
// Short/Long/Example literals are otherwise untested strings — a behavior-green
// change can ship a broken help block and no test would notice. Rendering
// --help exercises the template and every literal, so a break fails here instead
// of in a user's terminal. --help short-circuits before RunE, so this has no
// side effects.
func TestEveryCommandHelpRenders(t *testing.T) {
	for _, c := range allCommands(New("dev")) {
		name := c.CommandPath()
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := execute(t, append(argsFor(c), "--help")...)
			if err != nil {
				t.Fatalf("%s --help errored: %v", name, err)
			}
			if !strings.Contains(stdout+stderr, "Usage:") {
				t.Errorf("%s --help produced no usage block:\n%s%s", name, stdout, stderr)
			}
		})
	}
}

// docRef matches repo-relative documentation paths (docs/….md) embedded in help
// text. Scoped to docs/ on purpose: those are concrete, unambiguous repo files,
// unlike vault paths (MEMORY.md) or template patterns (specs/<id>/…).
var docRef = regexp.MustCompile(`docs/[\w./-]+\.md`)

// TestHelpDocReferencesExist guards the exact bug the lesson names: a command's
// help text pointing at a docs/ file that does not exist (a merged change once
// shipped a Long referencing a since-removed guide-secrets-recover.md). It scans
// every command's Short/Long/Example for docs/….md references and asserts each
// file is present in the repo. The test runs from cli/internal/cmd, so repo root
// is three levels up.
func TestHelpDocReferencesExist(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	seen := map[string]bool{}
	for _, c := range allCommands(New("dev")) {
		text := strings.Join([]string{c.Short, c.Long, c.Example}, "\n")
		for _, ref := range docRef.FindAllString(text, -1) {
			if seen[ref] {
				continue
			}
			seen[ref] = true
			if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(ref))); err != nil {
				t.Errorf("%s help references %q, which does not exist in the repo: %v", c.CommandPath(), ref, err)
			}
		}
	}
	if len(seen) == 0 {
		t.Skip("no docs/ references found in any command help — nothing to guard yet")
	}
}
