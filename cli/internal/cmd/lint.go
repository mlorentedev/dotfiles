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

	"github.com/mlorentedev/dotfiles/cli/internal/docpaths"
	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/pwsh"
)

func newLintCmd() *cobra.Command {
	lint := &cobra.Command{
		Use:   "lint",
		Short: "Run CI's lint jobs on this machine",
	}
	lint.AddCommand(newLintPowerShellCmd())
	lint.AddCommand(newLintDocPathsCmd())
	lint.AddCommand(newLintFleetCmd())
	return lint
}

func newLintDocPathsCmd() *cobra.Command {
	var repo, policy string
	cmd := &cobra.Command{
		Use:          "doc-paths [file...]",
		Short:        "Verify live repo paths named by active instruction files",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, files []string) error {
			return runLintDocPaths(cmd.OutOrStdout(), cmd.ErrOrStderr(), repo, policy, files)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "checkout to lint (default: the one the working directory is in)")
	cmd.Flags().StringVar(&policy, "policy", "", "policy file (default: <repo>/harness/doc-paths.json)")
	return cmd
}

func runLintDocPaths(stdout, stderr io.Writer, repo, policyPath string, files []string) error {
	repoExplicit := repo != ""
	var prefix string
	if !repoExplicit {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
		root, rootPrefix, err := docpaths.RepositoryLocation(cwd)
		if err != nil {
			return err
		}
		repo, prefix = root, rootPrefix
	}
	repo, err := filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	if policyPath == "" {
		policyPath = filepath.Join(repo, "harness", "doc-paths.json")
	}
	policy, err := docpaths.LoadPolicy(policyPath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		files, err = docpaths.Discover(repo, policy)
		if err != nil {
			return err
		}
	} else {
		base := repo
		if prefix != "" {
			base = filepath.Join(repo, filepath.FromSlash(prefix))
		}
		files, err = repoRelativeDocPaths(repo, base, files)
		if err != nil {
			return err
		}
	}
	report, err := docpaths.CheckFiles(docpaths.CheckOptions{
		RepoRoot: repo, VaultRoot: envpkg.ResolvePath("VAULT_PATH"), Policy: policy,
	}, files)
	if err != nil {
		return err
	}
	return printDocPathReport(stdout, stderr, report)
}

func repoRelativeDocPaths(repo, base string, files []string) ([]string, error) {
	out := make([]string, 0, len(files))
	for _, file := range files {
		path := file
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, filepath.FromSlash(file))
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return nil, fmt.Errorf("resolve explicit file %q: %w", file, err)
		}
		slash := filepath.ToSlash(filepath.Clean(rel))
		if slash == ".." || strings.HasPrefix(slash, "../") || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("explicit file %q is outside repository root %s", file, repo)
		}
		out = append(out, slash)
	}
	return out, nil
}

func printDocPathReport(stdout, stderr io.Writer, report docpaths.Report) error {
	byFile := make(map[string][]docpaths.Finding, len(report.Checked))
	for _, finding := range report.Findings {
		byFile[finding.File] = append(byFile[finding.File], finding)
	}
	for _, file := range report.Checked {
		if len(byFile[file]) == 0 {
			_, _ = fmt.Fprintf(stdout, "doc-paths: OK %s\n", file)
		}
	}
	for _, finding := range report.Findings {
		_, _ = fmt.Fprintf(stderr, "%s: %s: %s\n", finding.File, finding.Message, finding.Token)
	}
	if len(report.Findings) > 0 {
		return withExitCode(1, fmt.Errorf("%d doc-path finding(s)", len(report.Findings)))
	}
	return nil
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
