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
		Long: "secrets reads the registry (secrets/registry.yaml) and exposes the mapped\n" +
			"secrets on demand. `run` injects them into one child process only (never the\n" +
			"ambient shell); `show` prints one value; `render` materializes {env:VAR}\n" +
			"placeholders in a config file; `ls` lists ids (ADR-028 §2).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newSecretsRunCmd())
	cmd.AddCommand(newSecretsLsCmd())
	cmd.AddCommand(newSecretsShowCmd())
	cmd.AddCommand(newSecretsRenderCmd())
	return cmd
}

// registryPath resolves secrets/registry.yaml (the mapping SSOT); a var so tests
// can point it at a fixture. ageDecryptor is the age decrypt seam (nil → AgeDecrypt);
// bwReader is the Bitwarden read seam (BWGet in production), both overridable so
// command tests inject fakes with no age key and no unlocked Bitwarden.
var (
	registryPath = func() string { return filepath.Join(env.DotfilesDir(env.Home()), "secrets", "registry.yaml") }
	ageDecryptor secrets.Decryptor
	bwReader     secrets.BWReader = secrets.BWGet{}
)

// secretLoader builds the resolution engine wired with both backend seams, over the
// age store in <dotfiles>/sensitive and the operator's Bitwarden via bwReader.
func secretLoader() *secrets.Loader {
	return &secrets.Loader{
		SecretsDir: filepath.Join(env.DotfilesDir(env.Home()), "sensitive"),
		KeyPath:    ageKeyPath(),
		Decrypt:    ageDecryptor,
		BW:         bwReader,
	}
}

func loadRegistry() (*secrets.Registry, error) {
	data, err := os.ReadFile(registryPath())
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	return secrets.ParseRegistry(data)
}

// newSecretsLsCmd lists registry ids with plane + exposed vars — never values.
// With --pairs it instead prints one "VAR\t<age-source>" line per age env secret
// (file and bw secrets excluded), the machine-readable form github-secrets-manager.sh
// consumes in place of the retired env-mapping.conf. bw secrets carry no age source,
// so the age-based CI-push path skips them (rethought in the migration follow-up).
func newSecretsLsCmd() *cobra.Command {
	var pairs bool
	c := &cobra.Command{
		Use:          "ls",
		Short:        "List registry secret ids with plane and exposed vars (no values)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if pairs {
				for _, e := range reg.Entries(env.Home()) {
					if e.IsFile || e.Backend == "bw" {
						continue // file secrets aren't GitHub Actions secrets; bw has no age source
					}
					_, _ = fmt.Fprintf(w, "%s\t%s\n", e.Var, e.File)
				}
				return nil
			}
			for i := range reg.Secrets {
				s := &reg.Secrets[i]
				_, _ = fmt.Fprintf(w, "%-26s %-9s %s\n", s.ID, s.Plane, strings.Join(s.Vars(), ","))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&pairs, "pairs", false, "print VAR<TAB>age-source for each env secret (file secrets excluded)")
	return c
}

// newSecretsShowCmd prints one secret's decrypted value to stdout (no trailing
// newline, for `KEY=$(dotf secrets show <id>)`). Single-env secrets only — file
// and multi-var secrets must go through `run` (a value to stdout would be ambiguous).
func newSecretsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "show <id>",
		Short:        "Print one secret's decrypted value to stdout, no trailing newline",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			entry, err := reg.ShowEntry(args[0])
			if err != nil {
				return err
			}
			kv, err := secretLoader().EnvFor([]secrets.Entry{entry}, nil)
			if err != nil {
				return err
			}
			_, val, _ := strings.Cut(kv[0], "=") // EnvFor scrubs newlines → capture-friendly
			_, _ = fmt.Fprint(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

// newSecretsRenderCmd materializes a config file in place: it substitutes every
// {env:VAR} placeholder whose VAR is a registry-exposed env secret with that
// secret's decrypted value (ADR-028 / SDD-009). It is the Go replacement for the
// substitute_env_placeholders / Substitute-EnvPlaceholders shell twins, wired
// into setup for opencode.jsonc and pi's models.json. Unmapped placeholders and
// genuinely-absent secrets are left intact for the runtime resolver; a real
// resolution failure is surfaced with its specific cause (and fatal under --strict).
func newSecretsRenderCmd() *cobra.Command {
	var strict bool
	c := &cobra.Command{
		Use:   "render <file>",
		Short: "Substitute {env:VAR} placeholders in <file> with decrypted secret values (in place)",
		Long: "render rewrites <file> in place, replacing each {env:VAR} placeholder whose\n" +
			"VAR is a registry-mapped env secret (secrets/registry.yaml) with the resolved\n" +
			"value. Placeholders with no registry mapping, or whose secret is genuinely\n" +
			"absent on this machine, are left intact for the runtime resolver — setup\n" +
			"completes. A real failure (wrong age key, locked vault, empty value, bw\n" +
			"item/field typo) is reported with its specific cause and leaves the placeholder\n" +
			"intact; --strict turns any such failure into a non-zero exit. Atomic write, 0600.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			res, err := secrets.Render(args[0], reg, secretLoader(), env.Home())
			if err != nil {
				return err
			}
			errOut := cmd.ErrOrStderr()
			if len(res.Unmapped) > 0 {
				_, _ = fmt.Fprintf(errOut, "render: unmapped placeholders left for runtime resolution: %s\n", strings.Join(res.Unmapped, " "))
			}
			if len(res.Missing) > 0 {
				_, _ = fmt.Fprintf(errOut, "render: secrets not provisioned here, left intact: %s\n", strings.Join(res.Missing, " "))
			}
			for _, u := range res.Unresolved {
				_, _ = fmt.Fprintf(errOut, "warning: render: %s could not be resolved: %v\n", u.Var, u.Err)
			}
			if strict && len(res.Unresolved) > 0 {
				return fmt.Errorf("render: %d placeholder(s) failed to resolve (--strict)", len(res.Unresolved))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&strict, "strict", false, "exit non-zero if any mapped secret fails to resolve (absent secrets still tolerated)")
	return c
}

func newSecretsRunCmd() *cobra.Command {
	var only string
	c := &cobra.Command{
		Use:   "run [--only VAR,...] -- <cmd> [args...]",
		Short: "Decrypt mapped secrets and run <cmd> with them in its environment only",
		Long: "run decrypts the mapped secrets (secrets/registry.yaml, over the age store)\n" +
			"and launches <cmd> with them added to ITS environment — the parent shell is\n" +
			"never touched. File secrets (@VAR=file>dest) are materialized to dest (0600)\n" +
			"and VAR points at the path. --only scopes the injection to named vars. Backend\n" +
			"unlock credentials (e.g. BW_SESSION) are stripped from the child env, so <cmd>\n" +
			"gets only the granted secrets, not the key to the whole vault. The child's exit\n" +
			"code is propagated.\n\n" +
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

			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			sel, err := resolveOnly(reg, only)
			if err != nil {
				return err
			}
			childEnv, err := buildChildEnv(reg, sel)
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
	c.Flags().StringVar(&only, "only", "", "comma-separated registry ids or env-var names to inject (default: all mapped)")
	return c
}

// buildChildEnv flattens the registry to entries, resolves the selected secrets
// (per-backend), and returns the child environment: the parent env with the backend
// unlock credentials stripped, plus the granted KEY=VALUE pairs. The child gets only
// the secrets it was granted — never the master credential that opens the whole
// vault (defense in depth; cf. 1Password's `op run` + `env -u OP_SERVICE_ACCOUNT_TOKEN`).
func buildChildEnv(reg *secrets.Registry, only map[string]bool) ([]string, error) {
	injected, err := secretLoader().EnvFor(reg.Entries(env.Home()), only)
	if err != nil {
		return nil, err
	}
	return append(stripBackendAuth(os.Environ()), injected...), nil
}

// backendAuthVars are credentials that unlock a secret backend (the vault keys
// themselves, not resolved secrets). dotf secrets run strips them from the child
// environment so a launched tool cannot turn around and read the whole vault.
var backendAuthVars = map[string]bool{
	"BW_SESSION":               true, // Bitwarden unlock token (opens the whole vault)
	"BW_PASSWORD":              true, // Bitwarden master password (API-key login)
	"BW_CLIENTSECRET":          true, // Bitwarden API client secret
	"OP_SERVICE_ACCOUNT_TOKEN": true, // 1Password service-account token
}

// stripBackendAuth returns environ without any backend-unlock credential (matched by
// the KEY before '='). The input slice is not mutated.
func stripBackendAuth(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if backendAuthVars[name] {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// resolveOnly expands a comma-separated --only value into the set of env-var names
// to inject. Each token is a registry id (→ all the secret's vars) or an env/file
// var name (→ just itself). Empty → nil (= all mapped). Unknown token → error.
func resolveOnly(reg *secrets.Registry, s string) (map[string]bool, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	set := map[string]bool{}
	for _, tok := range strings.Split(s, ",") {
		if tok = strings.TrimSpace(tok); tok == "" {
			continue
		}
		vars, ok := reg.Selector(tok)
		if !ok {
			return nil, fmt.Errorf("--only: unknown id or env var %q", tok)
		}
		for _, v := range vars {
			set[v] = true
		}
	}
	// An explicit --only that resolves to nothing (e.g. "," or "  ,  ") must never
	// silently inject zero secrets — that runs the child unauthenticated (#612 A3).
	if len(set) == 0 {
		return nil, fmt.Errorf("--only %q selected no secrets", s)
	}
	return set, nil
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
