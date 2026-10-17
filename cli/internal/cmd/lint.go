package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/pwsh"
)

func newLintCmd() *cobra.Command {
	lint := &cobra.Command{
		Use:   "lint",
		Short: "Run CI's lint jobs on this machine",
	}
	lint.AddCommand(newLintPowerShellCmd())
	return lint
}

func newLintPowerShellCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Run CI's lint-powershell job here: PSScriptAnalyzer at the versions.conf pin",
		Long: "powershell analyzes the checkout's .ps1 and .psm1 files with the\n" +
			"PSScriptAnalyzer versions.conf pins, under .PSScriptAnalyzerSettings.psd1 at\n" +
			"severity Error and Warning, as CI's lint-powershell job does. It\n" +
			"reads the files a commit of this tree would ship: tracked, and untracked but\n" +
			"not ignored. Exit 1 on any finding, 2 when the analyzer is not installed at\n" +
			"its pin (`dotf converge --only pwsh-modules` installs it).",
		Example:      "  dotf lint powershell",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runLintPowerShell(cmd.OutOrStdout(), cmd.ErrOrStderr(), repo)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "checkout to lint (default: the one the working directory is in)")
	return cmd
}

func runLintPowerShell(stdout, stderr io.Writer, repo string) error {
	if repo == "" {
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return errors.New("not inside a git checkout: run from the dotfiles checkout or pass --repo")
		}
		repo = strings.TrimSpace(string(out))
	}
	raw, err := os.ReadFile(filepath.Join(repo, "versions.conf"))
	if err != nil {
		return err
	}
	version, err := analyzerPin(raw)
	if err != nil {
		return err
	}
	files, err := lintFiles(repo)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		_, _ = fmt.Fprintln(stdout, "No PowerShell files to analyze.")
		return nil
	}
	if _, err := exec.LookPath(pwsh.Command); err != nil {
		return withExitCode(2, errors.New("pwsh is not on PATH: run `dotf converge --only tools`"))
	}
	c := exec.Command(pwsh.Command, "-NoProfile", "-NonInteractive", "-Command", pwsh.LintScript(version))
	c.Dir, c.Stdout, c.Stderr = repo, stdout, stderr
	c.Env = append(os.Environ(), pwsh.LintFilesEnv+"="+strings.Join(files, "\n"))
	var exit *exec.ExitError
	if err := c.Run(); errors.As(err, &exit) {
		return withExitCode(exit.ExitCode(), fmt.Errorf("lint-powershell failed (exit %d)", exit.ExitCode()))
	} else if err != nil {
		return err
	}
	return nil
}

// analyzerPin is the PSScriptAnalyzer version versions.conf marks for
// pwsh-modules: the copy converge installs is the one the lint may import.
func analyzerPin(versionsConf []byte) (string, error) {
	mods, err := pwsh.ParseModules(versionsConf)
	if err != nil {
		return "", err
	}
	for _, m := range mods {
		if m.Name == pwsh.Analyzer {
			return m.Version, nil
		}
	}
	return "", fmt.Errorf("versions.conf marks no %q module, so nothing installs the analyzer this lint runs", pwsh.ModuleMarker+" "+pwsh.Analyzer)
}

// lintFiles lists, sorted and relative to repo, the PowerShell files a commit
// of the working tree would carry: tracked or untracked-unignored, and present.
func lintFiles(repo string) ([]string, error) {
	out, err := exec.Command("git", "-C", repo, "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, fmt.Errorf("listing the checkout's files: %w", err)
	}
	var files []string
	for _, p := range pwsh.Lintable(strings.Split(string(bytes.TrimRight(out, "\x00")), "\x00")) {
		if _, err := os.Stat(filepath.Join(repo, p)); err == nil {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files, nil
}
