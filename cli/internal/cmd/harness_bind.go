package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// newHarnessBindCmd is the third manifest mode beside `deploy` and `presence`,
// and the only writer of the `hooks` key in a harness settings file.
//
// WHY IT REPLACES SHELL RATHER THAN JOINING IT. `merge_claude_settings()` in
// setup-linux.sh does `.hooks.SessionStart = $tmpl.hooks.SessionStart` — an
// ASSIGNMENT, not a merge. Simulated against a copy of the deployed file on
// 2026-08-27: SessionStart went from 2 groups to 1, deleting a live third-party
// hook. setup-windows.ps1 carries the identical defect at its own lines. Adding
// a second writer beside them would not have fixed it; the assignment had to go,
// and then the file needs exactly one owner. That is this repository's most
// repeated lesson, now on its seventh surface: THE WRITER TOUCHES ONLY WHAT IT
// OWNS, and ownership is by marker, never by position.
//
// Emission is data-driven from `agents.bind` so adding a harness is a manifest
// edit. A target declaring `emit: false` is skipped visibly rather than
// forgotten, and one naming `requires_command` is skipped when that binary is
// absent — an uninstalled harness is not a failure.
func newHarnessBindCmd() *cobra.Command {
	var (
		harnessName string
		repoRoot    string
		homeDir     string
		dotfPath    string
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "bind",
		Short: "Emit this repository's hooks into each harness's settings file",
		Long: `bind writes the hooks declared in harness/manifest.json's ` + "`agents.bind`" + ` into
each harness's own settings file, merging by marker so a third party's entries
are never reordered, rewritten or removed.

It is the only writer of the ` + "`hooks`" + ` key. A setup script that also assigns to
it will delete whatever it does not know about, which is what this replaced.

Re-running is idempotent: an unchanged file is not rewritten, and a CHANGED
command replaces our entry in place rather than appending a second one.`,
		Example: `  dotf harness bind
  dotf harness bind --harness claude
  dotf harness bind --dry-run`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := repoRoot
			if root == "" {
				root = env.ResolveHarnessRoot()
			}
			home := homeDir
			if home == "" {
				h, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("resolve home: %w", err)
				}
				home = h
			}
			binary := dotfPath
			if binary == "" {
				binary = resolveDotfPath(home)
			}
			binary = hookBinaryToken(binary, runtime.GOOS)

			targets, err := harness.LoadBindTargets(root)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			wrote := 0
			for _, t := range targets {
				if harnessName != "" && t.Agent != harnessName {
					continue
				}
				if !t.Emits() {
					_, _ = fmt.Fprintf(out, "skip %s: declared emit:false (%s)\n", t.Agent, t.Format)
					continue
				}
				if t.RequiresCommand != "" {
					if _, err := exec.LookPath(t.RequiresCommand); err != nil {
						_, _ = fmt.Fprintf(out, "skip %s: %s is not installed\n", t.Agent, t.RequiresCommand)
						continue
					}
				}
				changed, err := bindOne(t, home, binary, dryRun)
				if err != nil {
					return fmt.Errorf("%s: %w", t.Agent, err)
				}
				switch {
				case !changed:
					_, _ = fmt.Fprintf(out, "ok   %s: hooks already current\n", t.Agent)
				case dryRun:
					_, _ = fmt.Fprintf(out, "would update %s: %s\n", t.Agent, t.File)
				default:
					wrote++
					_, _ = fmt.Fprintf(out, "bind %s: %s\n", t.Agent, t.File)
				}
			}
			if harnessName != "" && wrote == 0 && !dryRun {
				// Not an error: "already current" is the steady state, and the
				// idempotence doctrine wants changed=0 on a re-run.
				return nil
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessName, "harness", "", "bind only this harness (default: every declared target)")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "root containing harness/manifest.json")
	cmd.Flags().StringVar(&homeDir, "home", "", "home directory the settings files live under (default: the user's)")
	cmd.Flags().StringVar(&dotfPath, "dotf-path", "", "absolute path emitted into the hook commands (default: resolved)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would change without writing")
	return cmd
}

// bindOne merges one target's hooks into its settings file.
func bindOne(t harness.BindTarget, home, binary string, dryRun bool) (bool, error) {
	cmds, err := t.HookCommands(binary)
	if err != nil {
		return false, err
	}
	path := filepath.Join(home, filepath.FromSlash(t.File))

	doc := map[string]any{}
	raw, readErr := os.ReadFile(path) // #nosec G304 -- path comes from the manifest's own declaration
	switch {
	case readErr == nil:
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &doc); err != nil {
				// Refuse rather than bootstrap over it. A settings file that is
				// present but unparseable is a file someone is editing, and
				// replacing it loses their work — the opposite of the merge's
				// whole purpose.
				return false, fmt.Errorf("%s is not valid JSON, refusing to overwrite it: %w", path, err)
			}
		}
	case os.IsNotExist(readErr):
		// Bootstrapping an absent file is fine: there is nothing to preserve.
	default:
		return false, fmt.Errorf("read %s: %w", path, readErr)
	}

	merged, changed, err := harness.MergeHooks(doc, cmds)
	if err != nil {
		return false, err
	}
	if !changed || dryRun {
		return changed, nil
	}

	encoded, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode %s: %w", path, err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	// Temp + rename IN THE SAME DIRECTORY: a rename across filesystems is not
	// atomic, and a half-written settings file is a harness that will not start.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".settings-*.json")
	if err != nil {
		return false, fmt.Errorf("create temp beside %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("write %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return false, fmt.Errorf("chmod %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return false, fmt.Errorf("rename onto %s: %w", path, err)
	}
	return true, nil
}

// resolveDotfPath picks the absolute binary path emitted into hook commands.
//
// It prefers ~/.local/bin/dotf — the path the installer writes and the one the
// session hooks already carried — over whatever happens to be on the PATH of the
// process running setup. Those differ exactly when it matters: a `go run` or a
// build-tree binary would otherwise be baked into a hook that outlives it.
//
// The `.exe` suffix is not cosmetic. Adoption of the entry the setup scripts
// wrote before this command existed is by EXACT command equality (sameCommand),
// and setup-windows.ps1 emitted `"…\dotf.exe" mem session-start`. Resolving to a
// suffix-less path there would match nothing and append a SECOND session-start
// hook on the first Windows run — the duplicate this merge exists to prevent.
func resolveDotfPath(home string) string {
	name := "dotf"
	if runtime.GOOS == "windows" {
		name = "dotf.exe"
	}
	installed := filepath.Join(home, ".local", "bin", name)
	if _, err := os.Stat(installed); err == nil {
		return installed
	}
	if p, err := exec.LookPath("dotf"); err == nil {
		abs, err := filepath.Abs(p)
		if err == nil {
			return abs
		}
		return p
	}
	return installed
}

// hookBinaryToken renders the binary path as it appears inside a hook command
// line, quoting it where the shell that runs the hook would otherwise split it.
//
// goos is a parameter rather than a read of runtime.GOOS so both branches are
// testable from either OS — the Windows leg of this behaviour cannot be
// exercised on the machine that develops it otherwise.
//
// Windows is quoted unconditionally, matching byte-for-byte what
// setup-windows.ps1 already deployed, because anything else fails to adopt that
// entry and duplicates it. Elsewhere the path is bare — the shape setup-linux.sh
// deployed — unless it contains a space, where quoting is the only correct
// rendering and the unquoted entry it declines to adopt was broken anyway.
func hookBinaryToken(path, goos string) string {
	if goos == "windows" || strings.ContainsAny(path, " \t") {
		return `"` + path + `"`
	}
	return path
}
