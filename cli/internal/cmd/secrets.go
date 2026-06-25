package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// newSecretsCmd is the `dotf secrets` noun: the on-demand (JIT) secrets path of
// ADR-028. `run` decrypts the age-mapped secrets and injects them into a single
// child process — never the ambient shell, which is the "not always exposed"
// objective the login-time load-secrets export violated.
func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "On-demand secrets — inject into a child process, never the shell (ADR-028)",
		Long: "secrets decrypts the age-encrypted secrets mapped in sensitive/env-mapping.conf\n" +
			"and exposes them on demand. `run` injects them into one child process only, so\n" +
			"they never live in the ambient shell environment (ADR-028 Phase 1).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newSecretsRunCmd())
	return cmd
}

func newSecretsRunCmd() *cobra.Command {
	var only string
	c := &cobra.Command{
		Use:   "run [--only VAR,...] -- <cmd> [args...]",
		Short: "Decrypt mapped secrets and run <cmd> with them in its environment only",
		Long: "run decrypts the mapped secrets (env-mapping.conf, over the age store as-is)\n" +
			"and launches <cmd> with them added to ITS environment — the parent shell is\n" +
			"never touched. File secrets (@VAR=file>dest) are materialized to dest (0600)\n" +
			"and VAR points at the path. --only scopes the injection to named vars. The\n" +
			"child's exit code is propagated.\n\n" +
			"Everything after -- is the command to run, e.g.:\n" +
			"  dotf secrets run -- goreleaser release\n" +
			"  dotf secrets run --only OPENAI_API_KEY -- python yt_metrics.py",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dash := cmd.ArgsLenAtDash()
			if dash < 0 || dash >= len(args) {
				return errors.New("usage: dotf secrets run [--only VAR,...] -- <cmd> [args...]")
			}
			childArgv := args[dash:]

			secretsDir := filepath.Join(env.DotfilesDir(env.Home()), "sensitive")
			childEnv, err := buildChildEnv(secretsDir, ageKeyPath(), parseOnly(only))
			if err != nil {
				return err
			}
			code, err := runChild(childArgv, childEnv, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if code != 0 {
				os.Exit(code) // propagate the child's failure to our caller (CI, &&-chains)
			}
			return nil
		},
	}
	c.Flags().StringVar(&only, "only", "", "comma-separated env var names to inject (default: all mapped)")
	return c
}

// buildChildEnv parses env-mapping.conf, decrypts the selected secrets, and
// returns the parent environment with the decrypted KEY=VALUE pairs appended.
func buildChildEnv(secretsDir, keyPath string, only map[string]bool) ([]string, error) {
	mapping := filepath.Join(secretsDir, "env-mapping.conf")
	f, err := os.Open(mapping)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", mapping, err)
	}
	defer func() { _ = f.Close() }()

	entries, err := secrets.ParseMapping(f, env.Home())
	if err != nil {
		return nil, err
	}
	loader := &secrets.Loader{SecretsDir: secretsDir, KeyPath: keyPath}
	injected, err := loader.EnvFor(entries, only)
	if err != nil {
		return nil, err
	}
	return append(os.Environ(), injected...), nil
}

// runChild runs argv with environ and inherited stdio, returning the child's exit
// code. A non-zero exit is the child's own status (not our error); only a failure
// to launch (binary missing, etc.) is returned as an error.
func runChild(argv, environ []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(argv) == 0 {
		return 1, errors.New("no command given after --")
	}
	c := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is the user's own command
	c.Env = environ
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	err := c.Run()
	if err == nil {
		return 0, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 1, fmt.Errorf("launch %s: %w", argv[0], err)
}

// ageKeyPath resolves the age identity: $AGE_KEY_PATH, else ~/.config/age/key.txt
// (the load-secrets default).
func ageKeyPath() string {
	if p := os.Getenv("AGE_KEY_PATH"); p != "" {
		return p
	}
	return filepath.Join(env.Home(), ".config", "age", "key.txt")
}

// parseOnly splits a comma-separated --only value into a set, or nil (= all).
func parseOnly(s string) map[string]bool {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	set := map[string]bool{}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = true
		}
	}
	return set
}
