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
			"ambient shell); `show` prints one value; `set` writes one value into Bitwarden\n" +
			"(idempotent); `migrate` cuts a secret over age→bw behind a parity gate; `render`\n" +
			"materializes {env:VAR} placeholders in a config file; `verify` health-checks\n" +
			"resolution without printing values; `ls` lists ids (ADR-028 §2).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newSecretsRunCmd())
	cmd.AddCommand(newSecretsLsCmd())
	cmd.AddCommand(newSecretsShowCmd())
	cmd.AddCommand(newSecretsSetCmd())
	cmd.AddCommand(newSecretsMigrateCmd())
	cmd.AddCommand(newSecretsSyncCmd())
	cmd.AddCommand(newSecretsRenderCmd())
	cmd.AddCommand(newSecretsVerifyCmd())
	return cmd
}

// registryPath resolves secrets/registry.yaml for READS — repo checkout first, then
// the deployed copy (ADR-030). registryWritePath is the WRITE seam: it resolves the
// checkout's registry and fails loud if none is found, so a mutation lands in the
// version-controlled SSOT and not the throwaway deployed copy (#635). Both are vars so
// tests can point them at a fixture. ageDecryptor is the age decrypt seam (nil →
// AgeDecrypt); bwReader is the Bitwarden read seam (BWGet in production), all
// overridable so command tests inject fakes with no age key and no unlocked Bitwarden.
var (
	registryPath      = env.ResolveRegistryPath
	registryWritePath = env.RepoRegistryPath
	ageDecryptor      secrets.Decryptor
	bwReader          secrets.BWReader = secrets.BWGet{}
)

// secretLoader builds the resolution engine wired with both backend seams, over the
// age store in sensitive/ (checkout-first, ADR-030 — same source as the registry it
// maps, so a repo-side rotation is seen without a redeploy) and Bitwarden via bwReader.
func secretLoader() *secrets.Loader {
	return &secrets.Loader{
		SecretsDir: env.ResolveSensitiveDir(),
		KeyPath:    ageKeyPath(),
		Decrypt:    ageDecryptor,
		BW:         bwReader,
	}
}

// backendOf is the display label for an entry's backend ("age" when unset).
func backendOf(e secrets.Entry) string {
	if e.Backend == "" {
		return "age"
	}
	return e.Backend
}

// newSecretsVerifyCmd resolves each selected registry secret and reports OK / MISSING
// / FAILED per var — never the value (a read-only health check). No args verifies all
// entries; ids/var-names scope it via the --only selector. Exit is non-zero when any
// secret FAILED; --require-all also fails on a MISSING secret.
func newSecretsVerifyCmd() *cobra.Command {
	var requireAll bool
	c := &cobra.Command{
		Use:   "verify [id...]",
		Short: "Resolve each registry secret and report OK/MISSING/FAILED (no values printed)",
		Long: "verify resolves each registry secret through its backend and reports a per-var\n" +
			"status without printing the value or materializing any file:\n" +
			"  OK       resolved to a non-empty value\n" +
			"  MISSING  not provisioned on this machine (age file absent) — tolerated\n" +
			"  FAILED   a real failure (wrong key, locked Bitwarden, empty value, typo)\n" +
			"No args verifies all; ids/var-names scope it. Exit is non-zero on any FAILED\n" +
			"(--require-all also fails on MISSING).",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := loadRegistry()
			if err != nil {
				return err
			}
			only, err := resolveOnly(reg, strings.Join(args, ","))
			if err != nil {
				return err
			}
			loader := secretLoader()
			w := cmd.OutOrStdout()
			var ok, missing, failed int
			for _, e := range reg.Entries(env.Home()) {
				if only != nil && !only[e.Var] {
					continue
				}
				switch err := loader.Verify(e); {
				case err == nil:
					ok++
					_, _ = fmt.Fprintf(w, "OK       %-30s %s\n", e.Var, backendOf(e))
				case errors.Is(err, secrets.ErrSecretAbsent):
					missing++
					_, _ = fmt.Fprintf(w, "MISSING  %-30s %s\n", e.Var, backendOf(e))
				default:
					failed++
					_, _ = fmt.Fprintf(w, "FAILED   %-30s %s: %v\n", e.Var, backendOf(e), err)
				}
			}
			_, _ = fmt.Fprintf(w, "\n%d ok, %d missing, %d failed\n", ok, missing, failed)
			if failed > 0 || (requireAll && missing > 0) {
				return fmt.Errorf("verify: %d failed, %d missing", failed, missing)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&requireAll, "require-all", false, "also fail when a secret is MISSING (not provisioned here)")
	return c
}

func loadRegistry() (*secrets.Registry, error) {
	data, err := os.ReadFile(registryPath())
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	return secrets.ParseRegistry(data)
}

// newSecretsLsCmd lists registry ids with plane + exposed vars — never values. The CI
// upload path is `dotf secrets sync ci` (backend-agnostic), which owns selection end to
// end; the former `--pairs` (VAR<TAB>age-source, age-only) was the backend leak it retired.
func newSecretsLsCmd() *cobra.Command {
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
			for i := range reg.Secrets {
				s := &reg.Secrets[i]
				_, _ = fmt.Fprintf(w, "%-26s %-9s %s\n", s.ID, s.Plane, strings.Join(s.Vars(), ","))
			}
			return nil
		},
	}
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
