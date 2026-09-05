package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	cmd.AddCommand(newSecretsRotateCmd())
	cmd.AddCommand(newSecretsMigrateCmd())
	cmd.AddCommand(newSecretsSyncCmd())
	cmd.AddCommand(newSecretsRenderCmd())
	cmd.AddCommand(newSecretsVerifyCmd())
	cmd.AddCommand(newSecretsProbeCmd())
	cmd.AddCommand(newSecretsBackupCmd())
	cmd.AddCommand(newSecretsUnlockCmd())
	cmd.AddCommand(newSecretsLockCmd())
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

	// bwReader / bwWriter are TEST SEAMS ONLY: nil in production, where both halves
	// come from the pinned backend below. A test assigns one (or both) to inject a
	// fake with no age key and no unlocked Bitwarden.
	bwReader secrets.BWReader
	bwWriter bwWriteClient

	// bwBackendOnce / bwBackendPin implement BUG-084's pinning decision: the
	// daemon-vs-shellout choice is made ONCE per process — a `dotf` invocation is one
	// command — and both the read and write halves come from that single decision, so
	// a command cannot read through the daemon and write through the CLI. That split
	// is what BUG-084 was.
	//
	// Lazy (sync.Once, not a package initialiser) so commands that touch no secret —
	// `dotf env`, `dotf doctor`, `dotf spec` — never pay the daemon probe.
	bwBackendOnce sync.Once
	bwBackendPin  secrets.BWBackend
)

// bwBackend returns the process-wide pinned Bitwarden backend, probing the daemon on
// first use only.
func bwBackend() secrets.BWBackend {
	bwBackendOnce.Do(func() {
		bwBackendPin = secrets.SelectBWBackend(secrets.BWServeClient{})
	})
	return bwBackendPin
}

// bwRead / bwWrite are how every command reaches Bitwarden. They prefer an injected
// test seam and otherwise take the matched half of the pinned backend — so production
// code has no way to mix the two subjects even by accident.
func bwRead() secrets.BWReader {
	if bwReader != nil {
		return bwReader
	}
	return bwBackend().Reader
}

func bwWrite() bwWriteClient {
	if bwWriter != nil {
		return bwWriter
	}
	return bwBackend().Writer
}

// bwSync returns the cache-refresh half of the SAME pinned backend. Sync is pinned with
// the other two because the daemon and the CLI cache independently: syncing one leaves
// the other stale, so a rotation that syncs the wrong subject cannot prove its own write.
func bwSync() daemonSyncer {
	if bwSyncer != nil {
		return bwSyncer
	}
	return bwBackend().Syncer
}

// secretLoader builds the resolution engine wired with both backend seams, over the
// age store in sensitive/ (checkout-first, ADR-030 — same source as the registry it
// maps, so a repo-side rotation is seen without a redeploy) and Bitwarden via bwReader.
func secretLoader() *secrets.Loader {
	return &secrets.Loader{
		SecretsDir: env.ResolveSensitiveDir(),
		KeyPath:    ageKeyPath(),
		Decrypt:    ageDecryptor,
		BW:         bwRead(),
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
			// The partial door, deliberately: a health check must not be stopped by
			// the very kind of breakage it exists to report (BUG-086, #1004).
			reg, defects, err := loadRegistryPartial()
			if err != nil {
				return err
			}
			defects, only, err := scopeVerify(reg, defects, args)
			if err != nil {
				return err
			}
			loader := secretLoader()
			w := cmd.OutOrStdout()
			var ok, missing, failed int
			// A malformed entry is a FAILED row for that secret, not an abort. Its
			// vars are not expanded — the entry is exactly what could not be read, so
			// naming them would be guesswork.
			for _, d := range defects {
				failed++
				_, _ = fmt.Fprintf(w, "FAILED   %-30s registry: %v\n", d.ID, d.Err)
			}
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

// loadRegistryPartial is the health-check door: it returns the well-formed secrets plus
// a defect per malformed one, instead of failing on the first. ONLY `verify` uses it —
// every write path stays on loadRegistry's fail-loud behaviour, because a half-valid
// registry is precisely the state in which `set`/`migrate`/`render` must not run.
func loadRegistryPartial() (*secrets.Registry, []secrets.SecretDefect, error) {
	data, err := os.ReadFile(registryPath())
	if err != nil {
		return nil, nil, fmt.Errorf("read registry: %w", err)
	}
	return secrets.ParseRegistryPartial(data)
}

// scopeVerify splits verify's arguments across the two populations a partial load
// produces: defective secrets (which exist only as defects) and well-formed ones (which
// exist only in the registry).
//
// Without this, a scoped `verify <malformed-id>` would report "unknown id" — the entry
// was excluded from the registry precisely because it is the one being asked about.
//
// With no args, everything is in scope. With args, an entry the caller did not name is
// neither validated nor resolved (BUG-086 AC2): a defect elsewhere in the file must not
// make a scoped check fail.
func scopeVerify(reg *secrets.Registry, defects []secrets.SecretDefect, args []string) ([]secrets.SecretDefect, map[string]bool, error) {
	if len(args) == 0 {
		return defects, nil, nil
	}
	// Keyed to a SLICE, not a single defect: one id can carry several. A duplicate id
	// is itself a defect, so "one defect per id" is the assumption the data disproves.
	byID := make(map[string][]secrets.SecretDefect, len(defects))
	for _, d := range defects {
		byID[d.ID] = append(byID[d.ID], d)
	}
	var inScope []secrets.SecretDefect
	var healthy []string
	for _, tok := range args {
		if tok = strings.TrimSpace(tok); tok == "" {
			continue
		}
		ds, isDefect := byID[tok]
		inScope = append(inScope, ds...)

		// One token can name BOTH a defect and a valid entry, in two distinct ways,
		// and neither may short-circuit the other:
		//
		//   1. a duplicate id whose first definition validated and whose second did
		//      not — reporting only the defect hides the half that resolves;
		//   2. a NAME COLLISION: a malformed secret whose id is "FOO" alongside a
		//      well-formed secret exposing a var called "FOO". Two genuinely
		//      different secrets, both named by one token, both owed an answer.
		//
		// Case 2 was surfaced by the adversarial review as undocumented; it is pinned
		// by TestSecretsVerify_TokenThatIsBothADefectIdAndAValidVar.
		_, known := reg.Selector(tok)
		if known || !isDefect {
			// Unknown-and-not-a-defect falls through deliberately, so resolveOnly
			// produces the "unknown id" error rather than this silently accepting it.
			healthy = append(healthy, tok)
		}
	}
	if len(healthy) == 0 {
		// Every named id is defective: select no healthy entries, but still report.
		// A non-nil empty map means "filter to nothing", where nil would mean "all".
		return inScope, map[string]bool{}, nil
	}
	only, err := resolveOnly(reg, strings.Join(healthy, ","))
	if err != nil {
		return nil, nil, err
	}
	return inScope, only, nil
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

var (
	stdoutIsTerminal = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
	isAgentSession   = func() bool {
		return os.Getenv("CLAUDE_CODE") != "" ||
			os.Getenv("ANTIGRAVITY_AGENT") != "" ||
			os.Getenv("ANTIGRAVITY_CLI") != "" ||
			os.Getenv("AGENT_SESSION") != ""
	}
	clipboardRunner = func(text string) error {
		bin, args := clipboardCommand()
		cmd := exec.Command(bin, args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", bin, err)
		}
		return nil
	}
)

func clipboardCommand() (string, []string) {
	if runtime.GOOS == "darwin" {
		return "pbcopy", nil
	}
	if runtime.GOOS == "windows" {
		return "clip.exe", nil
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy", nil
		}
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		return "xclip", []string{"-selection", "clipboard"}
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return "xsel", []string{"--clipboard", "--input"}
	}
	return "wl-copy", nil
}

// newSecretsShowCmd resolves one secret's decrypted value. Supports:
// 1. --reveal: explicitly print decrypted value to stdout.
// 2. -c / --clip: copy to system clipboard without printing to stdout.
// 3. TTY masking: in interactive terminals without --reveal or -c, masks the secret and guides the user.
// 4. Agent isolation: refuses to print to stdout in AI agent sessions per ADR-028 doctrine.
func newSecretsShowCmd() *cobra.Command {
	var (
		reveal bool
		clip   bool
	)
	c := &cobra.Command{
		Use:   "show <id>",
		Short: "Print or copy one secret's decrypted value",
		Long: "show resolves one secret's decrypted value from its backend.\n\n" +
			"Flags:\n" +
			"  -c, --clip    copy the decrypted value to the system clipboard (never prints to stdout)\n" +
			"  --reveal      explicitly print the decrypted value to stdout\n\n" +
			"When stdout is an interactive terminal (TTY) and neither --reveal nor -c is given,\n" +
			"show masks the secret and guides you to use -c or --reveal (1Password op style).\n" +
			"In agent environments, printing to stdout is refused per ADR-028 doctrine.",
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

			if clip {
				if err := clipboardRunner(val); err != nil {
					return fmt.Errorf("copy to clipboard: %w", err)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Copied secret %q to clipboard\n", args[0])
				return nil
			}

			if isAgentSession() {
				return fmt.Errorf("refusing to print secret %q to stdout in an agent environment (ADR-028 doctrine); inject it via 'dotf secrets run -- <cmd>' instead", args[0])
			}

			if stdoutIsTerminal() && !reveal {
				masked := strings.Repeat("•", 12)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Secret %q: %s\n\nTo copy to clipboard:  dotf secrets show -c %s\nTo print in terminal:   dotf secrets show --reveal %s\nTo inject in command:   dotf secrets run -- <cmd>\n", args[0], masked, args[0], args[0])
				return nil
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), val)
			return nil
		},
	}
	c.Flags().BoolVarP(&clip, "clip", "c", false, "copy secret to system clipboard instead of printing")
	c.Flags().BoolVar(&reveal, "reveal", false, "reveal decrypted secret to stdout")
	return c
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
			injected, err := resolveInjectedSecrets(reg, sel)
			if err != nil {
				return err
			}
			childEnv := append(stripBackendAuth(os.Environ()), injected...)

			// SEC-002. exec.Cmd hands a child a real descriptor only when the
			// writer's dynamic type is *os.File; a redactWriter never is, so
			// every child got an os.Pipe and every INTERACTIVE child saw a
			// non-TTY stdout and declined to start. Where this process actually
			// owns a terminal, give the child a pty. Everywhere else -- CI, a
			// pipeline, `pi -p`, the whole reviewer pool -- nothing changes.
			var code int
			if interactiveChildSupported() && isTerminal(os.Stdout.Fd()) {
				// ONE writer on this path, because a pty merges the child's
				// stdout and stderr onto a single stream exactly as a terminal
				// does. Two writers would also each hold their own tail, and a
				// secret could interleave across the boundary between them.
				merged := newRedactWriter(cmd.OutOrStdout(), injected)
				code, err = runChildPTY(childArgv, childEnv, merged)
				_ = merged.Flush()
			} else {
				stdout := newRedactWriter(cmd.OutOrStdout(), injected)
				stderr := newRedactWriter(cmd.ErrOrStderr(), injected)
				code, err = runChild(childArgv, childEnv, cmd.InOrStdin(), stdout, stderr)
				_ = stdout.Flush()
				_ = stderr.Flush()
			}
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

// resolveInjectedSecrets evaluates the registry and returns the slice of KEY=VALUE strings
// to be injected into the child process.
func resolveInjectedSecrets(reg *secrets.Registry, only map[string]bool) ([]string, error) {
	return secretLoader().EnvFor(reg.Entries(env.Home()), only)
}

// buildChildEnv flattens the registry to entries, resolves the selected secrets
// (per-backend), and returns the child environment: the parent env with the backend
// unlock credentials stripped, plus the granted KEY=VALUE pairs. The child gets only
// the secrets it was granted — never the master credential that opens the whole
// vault (defense in depth; cf. 1Password's `op run` + `env -u OP_SERVICE_ACCOUNT_TOKEN`).
func buildChildEnv(reg *secrets.Registry, only map[string]bool) ([]string, error) {
	injected, err := resolveInjectedSecrets(reg, only)
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
//
// SIGINT and SIGTERM are forwarded to the child rather than killing us out from
// under it. Without this, Go's default disposition terminates `dotf` on the first
// signal and leaves the child ORPHANED: a Ctrl-C returns the prompt while the
// command keeps running, and `kill <dotf-pid>` never reaches what actually holds
// the secret. The case that forced it is CLI-042 AC7 — the hive daemon runs as
// `dotf secrets run -- hive serve`, so this process is a systemd unit's MainPID
// and a supervisor for the first time. systemd's default KillMode=control-group
// would SIGTERM the whole cgroup and paper over the gap there; every other caller
// (a shell, `timeout`, a CI runner) signals the process, not a cgroup, and got
// the orphan.
//
// Scoped to the two signals a supervisor actually sends. SIGKILL and SIGSTOP
// cannot be caught, and forwarding the rest (SIGHUP, SIGWINCH, job control)
// would mean re-implementing a shell's terminal handling for a wrapper that owns
// no terminal.
// isTerminal decides which of the two child paths `secrets run` takes. It is a
// package variable rather than a direct call so both branches are reachable in
// a test: CI has no controlling terminal, so the pty branch would otherwise be
// unreachable there and only ever exercised by hand on a developer's box --
// which is precisely how SEC-002 shipped.
var isTerminal = func(fd uintptr) bool { return term.IsTerminal(int(fd)) }

func runChild(argv, environ []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if err := assertSafeChildCommand(argv); err != nil {
		return 1, err
	}
	c := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is the user's own command
	c.Env = environ
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	if err := c.Start(); err != nil {
		return 1, fmt.Errorf("launch %s: %w", argv[0], err)
	}

	// Buffered so a signal arriving between Start and the goroutine's first
	// receive is delivered rather than dropped.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sigs:
				// Best-effort: the child may have already exited, in which case
				// the signal has nowhere to go and the Wait below is what matters.
				_ = c.Process.Signal(s)
			case <-done:
				return
			}
		}
	}()

	err := c.Wait()
	close(done)
	signal.Stop(sigs)

	if err == nil {
		return 0, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 1, fmt.Errorf("wait %s: %w", argv[0], err)
}

// ageKeyPath resolves the age identity: $AGE_KEY_PATH, else ~/.config/age/key.txt
// (the load-secrets default).
func ageKeyPath() string {
	if p := os.Getenv("AGE_KEY_PATH"); p != "" {
		return p
	}
	return filepath.Join(env.Home(), ".config", "age", "key.txt")
}

// assertSafeChildCommand refuses to launch commands whose primary purpose is to dump
// the environment or inspect secrets to standard output (e.g. `env`, `printenv`, `export`).
// Enforces ADR-028 and cross-agent doctrine: "Never dump a secrets store to standard output".
func assertSafeChildCommand(argv []string) error {
	if len(argv) == 0 {
		return errors.New("no command given after --")
	}
	base := strings.ToLower(filepath.Base(argv[0]))
	if base == "env" || base == "printenv" || base == "export" {
		return fmt.Errorf("refusing to run introspection command %q under dotf secrets run: never dump decrypted secrets to stdout (ADR-028 doctrine)", base)
	}

	// Catch shell wrappers executing introspection: `sh -c "env | grep..."`, `bash -lc "'env'"`, `bash -c "set"`, etc.
	if (base == "sh" || base == "bash" || base == "zsh" || base == "dash" || base == "ksh" || base == "busybox") && len(argv) >= 2 {
		for i := 1; i < len(argv); i++ {
			arg := argv[i]
			isCFlag := arg == "-c" || (strings.HasPrefix(arg, "-") && strings.Contains(arg, "c"))
			if isCFlag && i+1 < len(argv) {
				cmdStr := strings.TrimSpace(argv[i+1])
				// Normalize by stripping quotes and backslashes that bypass regex boundaries
				cleanCmd := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(cmdStr, "'", ""), "\"", ""), "\\", "")
				dangerousList := []string{"env", "printenv", "export", "set", "declare"}
				for _, dangerous := range dangerousList {
					pattern := `(?i)(?:^|[\s;|` + "`" + `&$()=])` + regexp.QuoteMeta(dangerous) + `(?:$|[\s;|` + "`" + `&$()])`
					matchedRaw, _ := regexp.MatchString(pattern, cmdStr)
					matchedClean, _ := regexp.MatchString(pattern, cleanCmd)
					if matchedRaw || matchedClean {
						return fmt.Errorf("refusing to run introspection shell snippet containing %q under dotf secrets run: never dump decrypted secrets to stdout (ADR-028 doctrine)", dangerous)
					}
				}
			}
		}
	}
	return nil
}

// redactWriter intercepts output emitted by the child process and replaces any byte sequence
// corresponding to an injected secret value (len >= 6) with [REDACTED:<KEY>].
// Enforces ADR-028 doctrine: "Never dump a secrets store to standard output".
type redactWriter struct {
	target       io.Writer
	pairs        [][2][]byte
	tail         []byte
	maxSecretLen int
	mu           sync.Mutex
}

func newRedactWriter(target io.Writer, injected []string) *redactWriter {
	var pairs [][2][]byte
	maxLen := 0
	for _, kv := range injected {
		key, val, ok := strings.Cut(kv, "=")
		if ok && len(val) >= 6 {
			valBytes := []byte(val)
			repBytes := []byte("[REDACTED:" + key + "]")
			pairs = append(pairs, [2][]byte{valBytes, repBytes})
			if len(valBytes) > maxLen {
				maxLen = len(valBytes)
			}
		}
	}
	return &redactWriter{
		target:       target,
		pairs:        pairs,
		maxSecretLen: maxLen,
	}
}

func (r *redactWriter) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pairs) == 0 {
		return r.target.Write(p)
	}

	data := p
	if len(r.tail) > 0 {
		data = append(r.tail, p...)
		r.tail = nil
	}

	for _, pair := range r.pairs {
		data = bytes.ReplaceAll(data, pair[0], pair[1])
	}

	// Withhold ONLY a trailing run that could still grow into a secret, never a
	// fixed window. The previous rule held back maxSecretLen-1 bytes on every
	// write regardless of content, which is invisible when the consumer is a
	// pipe drained at Flush() and fatal when it is a terminal: with a 64-byte
	// key, the last 63 bytes of every frame -- typically the cursor
	// positioning -- stayed dark until more output happened to arrive. A TUI
	// driven that way renders late and misplaced (SEC-002).
	if hold := r.holdBack(data); hold > 0 {
		split := len(data) - hold
		r.tail = make([]byte, hold)
		copy(r.tail, data[split:])
		_, err := r.target.Write(data[:split])
		return len(p), err
	}

	r.tail = nil
	_, err := r.target.Write(data)
	return len(p), err
}

// holdBack returns how many trailing bytes of data must be withheld because
// they are a PROPER prefix of some secret -- i.e. bytes that are not a leak
// yet but would be if the next write completed them. Zero in the common case,
// so a frame ending in ordinary output goes out whole and immediately.
//
// Only a proper prefix counts: a suffix that already contains a whole secret
// was replaced by the ReplaceAll pass above and is no longer present to match.
func (r *redactWriter) holdBack(data []byte) int {
	maxHold := r.maxSecretLen - 1
	if maxHold > len(data) {
		maxHold = len(data)
	}
	// Longest first: the held run must cover every secret it could complete,
	// and a shorter match would release bytes that a longer one still needs.
	for n := maxHold; n > 0; n-- {
		suffix := data[len(data)-n:]
		for _, pair := range r.pairs {
			if len(suffix) < len(pair[0]) && bytes.HasPrefix(pair[0], suffix) {
				return n
			}
		}
	}
	return 0
}

func (r *redactWriter) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.tail) == 0 {
		return nil
	}
	for _, pair := range r.pairs {
		r.tail = bytes.ReplaceAll(r.tail, pair[0], pair[1])
	}
	_, err := r.target.Write(r.tail)
	r.tail = nil
	return err
}
