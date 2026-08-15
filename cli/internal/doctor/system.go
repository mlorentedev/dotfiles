// Package doctor implements the `dotf doctor` post-setup diagnostics domain.
// It consolidates the two retired shell twins — scripts/healthcheck.sh (the
// 12-section sweep) and scripts/doctor.sh (the env-contract verifier with a
// --fix heal path) — into one cross-compiled checker (ADR-021, the first port).
//
// Design: the process-global / external surfaces (environment variables, PATH
// lookups, running `<tool> --version`) are abstracted behind System so checks
// are table-testable; the filesystem is hit directly via os/filepath, with
// tests building a real temp tree rooted at a fake $HOME.
package doctor

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// System abstracts the non-deterministic, process-global surfaces a check
// touches. The real implementation wires the os/exec stdlib; tests inject
// deterministic funcs. Filesystem access is intentionally NOT here — checks use
// os/filepath directly against paths derived from Getenv, and tests build a real
// temp tree, which is both more faithful and simpler than a virtual FS.
type System struct {
	// Getenv resolves an environment variable (os.Getenv in production).
	Getenv func(string) string
	// LookPath reports whether an executable is resolvable on PATH
	// (exec.LookPath in production); err != nil means "not found".
	LookPath func(string) (string, error)
	// CommandOutput runs name with args and returns combined stdout+stderr.
	// Used for the `<tool> --version` probes; faked in tests.
	CommandOutput func(name string, args ...string) (string, error)
	// CommandOutputDir is CommandOutput with a working directory — for tools that
	// resolve their target from the process cwd rather than an argument (e.g.
	// `pre-commit install`, which locates the git repo relative to where it runs).
	// Faked in tests.
	CommandOutputDir func(dir, name string, args ...string) (string, error)
	// HTTPGet issues a GET with the given request headers and returns the status
	// code + response headers (the body is never needed — the PAT-expiry check
	// reads only the status and the github-authentication-token-expiration
	// header). The network is the canonical "non-deterministic external surface"
	// this seam exists to isolate; tests inject canned responses. err != nil for
	// transport failures (offline, DNS, timeout).
	HTTPGet func(url string, headers map[string]string) (int, http.Header, error)
	// Now returns the current time (time.Now in production). A clock seam keeps
	// "days until expiry" deterministic under test.
	Now func() time.Time
	// GOOS is the target OS (runtime.GOOS in production). A seam so OS-gated
	// checks (skip POSIX-only tools / tmux / shell-rc symlinks on Windows) are
	// table-testable; the zero value "" behaves like a POSIX host.
	GOOS string
	// AgeRoundTrip proves the age key at keyPath actually decrypts: it derives
	// the recipient, encrypts a sentinel, and decrypts it back, returning a
	// non-nil error if any step fails or the bytes don't match. The real impl
	// (ageRoundTrip) shells out to age/age-keygen via the secrets seams — thin
	// I/O covered by a live smoke, never CI — so this exists as a seam the
	// secrets-tooling check calls and tests inject a fake round-trip into.
	AgeRoundTrip func(keyPath string) error
	// BWBackedSecrets counts the registry entries that resolve through Bitwarden
	// (backend: bw). It exists so the reach check can key its SEVERITY to real
	// exposure rather than to a flat policy: an unreachable vault is a WARN while
	// nothing depends on it and a FAIL the moment something does. The real impl
	// (bwBackedSecrets) reads env.RepoRegistryPath — the CHECKOUT-ONLY path —
	// never cfg.DotfilesDir and never env.ResolveRegistryPath. The deployed copy
	// lags the checkout during exactly the migration this check guards (ADR-030,
	// #635), which would hold the severity at WARN precisely as exposure begins;
	// ResolveRegistryPath is rejected for the same reason despite preferring the
	// checkout, because it falls back to the deployed copy when the checkout
	// registry is missing. Failing loud there is the point: the caller degrades
	// severity with a stated reason rather than trusting an unattributable count.
	BWBackedSecrets func() (int, error)
	// CommandOutputBounded is CommandOutput with a wall-clock deadline, for
	// subprocesses that touch the network. Plain CommandOutput has none, which is
	// fine for local probes but not for a command that can block on a stalled
	// connection: doctor is the last step of setup-linux.sh, so an unbounded hang
	// there hangs a bootstrap. It is the exec-side analogue of the 5s cap already
	// placed on the HTTPGet seam, and exists because that cap taught nothing to
	// the callers that shell out instead.
	//
	// It returns the two streams SEPARATELY, unlike CommandOutput's
	// CombinedOutput. Its callers parse machine-readable stdout (`bw status`
	// emits JSON there and human diagnostics on stderr), and merging the two
	// means one line of CLI chatter makes the parse fail — which is not a
	// theoretical concern: `bw`'s first invocation on a fresh machine prints
	// `Could not find data file, "…/data.json"; creating it instead.` to stderr,
	// so a merged read silently skipped every tier of the reach check on exactly
	// the freshly-provisioned box setup-linux.sh had just finished building.
	CommandOutputBounded func(d time.Duration, name string, args ...string) (stdout, stderr string, err error)
	// BWServeStatus reports the dotf-managed bw serve daemon's lock state:
	// "absent" (nothing reachable), "locked", or "unlocked". It never returns
	// a transport error for an absent daemon — that is itself the "absent"
	// state (secrets.BWServeDaemon.Status's own contract) — only a genuinely
	// unparseable response surfaces as err. CLI-024-secrets-bw-serve, AC4.
	BWServeStatus func() (string, error)
	// ResolveSecret resolves ONE registry entry to its plaintext value through the
	// same Loader `dotf secrets run` uses. It exists because the PAT-expiry check
	// used to read its token from the ambient environment, which ADR-028
	// guarantees is empty — so on a correctly configured machine the check SKIPped
	// every token and reported a remediation command that had been retired with
	// the loader. There is no environment state a health check could legitimately
	// require here; resolution is the only honest path (REFACTOR-012).
	//
	// It never materializes a file secret and never reports the value: callers use
	// it for one Authorization header. Returns secrets.ErrSecretAbsent (wrapped)
	// when the secret is genuinely not provisioned on this machine.
	ResolveSecret func(e secrets.Entry) (string, error)
}

// resolveSecret is the production ResolveSecret: the age store (checkout-first,
// ADR-030, same source as the registry it maps) plus a SERVE-ONLY Bitwarden
// reader.
//
// Serve-only is deliberate and is not the CLI's own wiring, which falls back to
// shelling out to `bw`. A shellout costs ~1.5s per secret against a locked vault
// — the latency that made `pi` hang for 45 seconds (BUG-080) — and doctor's
// callers include the last step of setup-linux.sh. Its caller gates bw entries on
// BWServeStatus before ever arriving here, so the fallback would only ever run in
// the case that is already known to be slow and already reported honestly.
func resolveSecret(e secrets.Entry) (string, error) {
	keyPath := os.Getenv("AGE_KEY_PATH")
	if keyPath == "" {
		keyPath = filepath.Join(env.Home(), ".config", "age", "key.txt")
	}
	l := &secrets.Loader{
		SecretsDir: env.ResolveSensitiveDir(),
		KeyPath:    keyPath,
		BW:         secrets.BWServeReader{Client: secrets.BWServeClient{}},
	}
	kv, err := l.EnvFor([]secrets.Entry{e}, nil)
	if err != nil {
		return "", err
	}
	if len(kv) == 0 {
		return "", fmt.Errorf("no value produced for %q", e.Var)
	}
	_, value, _ := strings.Cut(kv[0], "=")
	return value, nil
}

// realSystem wires System to the live OS.
func realSystem() *System {
	return &System{
		Getenv:   os.Getenv,
		LookPath: exec.LookPath,
		CommandOutput: func(name string, args ...string) (string, error) {
			out, err := exec.Command(name, args...).CombinedOutput()
			return string(out), err
		},
		CommandOutputDir: func(dir, name string, args ...string) (string, error) {
			cmd := exec.Command(name, args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			return string(out), err
		},
		HTTPGet: func(url string, headers map[string]string) (int, http.Header, error) {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return 0, nil, fmt.Errorf("build request %q: %w", url, err)
			}
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return 0, nil, fmt.Errorf("GET %q: %w", url, err)
			}
			defer func() { _ = resp.Body.Close() }()
			return resp.StatusCode, resp.Header, nil
		},
		Now:             time.Now,
		GOOS:            runtime.GOOS,
		AgeRoundTrip:    ageRoundTrip,
		BWBackedSecrets: bwBackedSecrets,
		BWServeStatus: func() (string, error) {
			return (&secrets.BWServeDaemon{Client: secrets.BWServeClient{}}).Status()
		},
		ResolveSecret: resolveSecret,
		CommandOutputBounded: func(d time.Duration, name string, args ...string) (string, string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()
			cmd := exec.CommandContext(ctx, name, args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if ctx.Err() != nil {
				return stdout.String(), stderr.String(), fmt.Errorf("%s timed out after %s", name, d)
			}
			return stdout.String(), stderr.String(), err
		},
	}
}

// env returns the value of key, or def when unset/empty. Mirrors the shell
// ${VAR:-default} idiom the twins lean on heavily.
func (s *System) env(key, def string) string {
	if v := s.Getenv(key); v != "" {
		return v
	}
	return def
}

// has reports whether name resolves on PATH (the `command -v` test).
//
// On Windows exec.LookPath only resolves names carrying a PATHEXT extension
// (.exe/.cmd/...), so an extensionless POSIX script installed on PATH — e.g.
// ~/.local/bin/bats, a bash script the setup lays down — reports as missing even
// though it runs fine under a shell. When LookPath misses on Windows, fall back
// to a `command -v`-style scan of PATH for a regular file whose name matches
// exactly. POSIX hosts resolve these through LookPath already, so the fallback is
// Windows-only: scanning the filesystem on POSIX would mask a genuinely-missing
// tool (BUG-052).
func (s *System) has(name string) bool {
	if _, err := s.LookPath(name); err == nil {
		return true
	}
	if s.GOOS == "windows" {
		return s.hasExtensionlessOnPath(name)
	}
	return false
}

// hasExtensionlessOnPath scans PATH for a regular file named exactly name. A
// name that already carries an extension would have been found by LookPath, so
// it is not re-scanned here.
func (s *System) hasExtensionlessOnPath(name string) bool {
	if filepath.Ext(name) != "" {
		return false
	}
	for _, dir := range s.pathEntries() {
		if dir == "" {
			continue
		}
		if isRegularFile(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

// home resolves the user's home directory, preferring HOME (POSIX) then
// USERPROFILE (Windows), matching the env-contract's OS-scoped vars.
func (s *System) home() string {
	if h := s.Getenv("HOME"); h != "" {
		return h
	}
	return s.Getenv("USERPROFILE")
}

// pathEntries splits PATH into its entries using the OS list separator.
func (s *System) pathEntries() []string {
	raw := s.Getenv("PATH")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, string(os.PathListSeparator))
}

// versionLine runs `<name> --version`, returning the first output line. A
// non-nil error means the probe itself failed (binary refused to run); callers
// treat that as "unparseable" rather than a hard failure, mirroring the twins.
func (s *System) versionLine(name string) (string, error) {
	out, err := s.CommandOutput(name, "--version")
	first := out
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		first = out[:i]
	}
	return strings.TrimSpace(first), err
}
