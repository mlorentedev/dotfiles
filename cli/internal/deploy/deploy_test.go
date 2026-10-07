package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoWithSource(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "ai", "pi", "models.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func piConfig() Config {
	return Config{Name: "pi", Src: "ai/pi/models.json", Dst: "{HOME}/.pi/agent/models.json", Mode: "0600"}
}

func noResolve(string) string { return "" }

func TestDeploy_InstallsWithDeclaredMode(t *testing.T) {
	root := repoWithSource(t, `{"providers":{"nan":{"apiKey":"${NAN_API_KEY}"}}}`)
	home := t.TempDir()

	res, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("a first deploy must report a change")
	}

	got, err := os.ReadFile(res.Dst)
	if err != nil {
		t.Fatalf("nothing was installed: %v", err)
	}
	if !strings.Contains(string(got), "${NAN_API_KEY}") {
		t.Errorf("the source was not installed verbatim: %s", got)
	}
	if runtime.GOOS == "windows" {
		return // POSIX permission bits are not meaningful here
	}
	info, err := os.Stat(res.Dst)
	if err != nil {
		t.Fatal(err)
	}
	// The declared mode is the point: a config that may carry a credential must
	// not land 0644 because nobody thought about it.
	if info.Mode().Perm() != 0o600 {
		t.Errorf("want mode 0600 as declared, got %o", info.Mode().Perm())
	}
}

// Rewriting an identical file churns mtime on every setup run, which makes "did
// this change?" unanswerable for an operator and for any check watching it.
func TestDeploy_IsIdempotentAndDoesNotRewrite(t *testing.T) {
	root := repoWithSource(t, `{"a":1}`)
	home := t.TempDir()

	first, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(first.Dst)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Deploy(piConfig(), root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Error("an unchanged source must report no change")
	}
	after, err := os.Stat(first.Dst)
	if err != nil {
		t.Fatal(err)
	}
	// Asserted on mtime, not on the printed output: a command that says "in sync"
	// while rewriting the file has told the truth about its intent and not about
	// the filesystem.
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("an in-sync config was rewritten; mtime moved")
	}
}

func TestDeploy_DryRunWritesNothing(t *testing.T) {
	root := repoWithSource(t, `{"a":1}`)
	home := t.TempDir()

	res, err := Deploy(piConfig(), root, home, noResolve, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed {
		t.Error("dry-run must still report that a change WOULD happen")
	}
	if _, err := os.Stat(res.Dst); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dry-run wrote to disk: %v", err)
	}
}

// render is CALLED, never reimplemented — the whole point of the port is that
// one substitution implementation exists.
func TestDeploy_RunsTheRendererOnTheStagedCopyOnly(t *testing.T) {
	root := repoWithSource(t, `{"key":"{env:NAN_API_KEY}"}`)
	home := t.TempDir()
	cfg := piConfig()
	cfg.Render = true

	var renderedPath string
	render := func(p string) error {
		renderedPath = p
		return os.WriteFile(p, []byte(`{"key":"RESOLVED"}`), 0o600)
	}

	res, err := Deploy(cfg, root, home, noResolve, render, false)
	if err != nil {
		t.Fatal(err)
	}
	if renderedPath == "" {
		t.Fatal("render was declared but never called")
	}
	// The renderer must see the STAGED copy, never the repo source: rendering the
	// source would write a credential into the checkout.
	if strings.HasPrefix(renderedPath, root) {
		t.Errorf("render was pointed at the repo source (%s) — that would put a secret in the checkout", renderedPath)
	}
	src, err := os.ReadFile(filepath.Join(root, "ai", "pi", "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "{env:NAN_API_KEY}") {
		t.Error("the repo source was modified by a deploy")
	}
	got, err := os.ReadFile(res.Dst)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "RESOLVED") {
		t.Errorf("the rendered copy was not the one installed: %s", got)
	}
}

// Each failure names the config. "deploy failed" sends an operator to read code.
func TestDeploy_ErrorsNameTheConfig(t *testing.T) {
	home := t.TempDir()
	for _, tt := range []struct {
		name string
		cfg  Config
		root string
		want string
	}{
		{"missing source", piConfig(), t.TempDir(), "source ai/pi/models.json"},
		{
			"unresolvable destination",
			Config{Name: "pi", Src: "ai/pi/models.json", Dst: "{NO_SUCH_VAR}/x.json"},
			repoWithSource(t, "{}"), "unresolvable path variable",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Deploy(tt.cfg, tt.root, home, noResolve, nil, false)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("want %q in the error, got: %v", tt.want, err)
			}
			if !strings.Contains(err.Error(), `"pi"`) {
				t.Errorf("the error must name the config, got: %v", err)
			}
		})
	}
}

// An unresolvable token must not silently become an empty segment: a path
// collapsing to "/models.json" is how a deploy lands where nobody looks.
func TestExpandDst_UnresolvableTokenIsAnErrorNotAnEmptySegment(t *testing.T) {
	_, err := ExpandDst("{PI_AGENT_DIR}/models.json", "/home/u", noResolve)
	if err == nil {
		t.Fatal("want an error for an unresolvable token")
	}
	if !strings.Contains(err.Error(), "PI_AGENT_DIR") {
		t.Errorf("the error must name the variable, got: %v", err)
	}

	got, err := ExpandDst("{PI_AGENT_DIR}/models.json", "/home/u", func(string) string { return "/opt/pi" })
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.FromSlash("/opt/pi/models.json") {
		t.Errorf("want the resolved path, got %q", got)
	}
}

func TestParseManifest_ValidatesAndNamesTheEntry(t *testing.T) {
	for _, tt := range []struct{ name, body, want string }{
		{"version", `{"version":9,"configs":[]}`, "version 9 unsupported"},
		{"empty name", `{"version":3,"configs":[{"src":"a","dst":"b"}]}`, "empty name"},
		{"duplicate", `{"version":3,"configs":[{"name":"pi","src":"a","dst":"b"},{"name":"pi","src":"c","dst":"d"}]}`, "duplicate config name"},
		{"empty src", `{"version":3,"configs":[{"name":"pi","dst":"b"}]}`, "empty src"},
		{"bad mode", `{"version":3,"configs":[{"name":"pi","src":"a","dst":"b","mode":"rwx"}]}`, "not octal"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tt.body))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("want %q, got: %v", tt.want, err)
			}
		})
	}
}

// The shipped manifest must parse, or the command is broken on every machine.
func TestParseManifest_ShippedManifestIsValid(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ManifestRel))
	if err != nil {
		t.Skipf("manifest not reachable from the test's working directory: %v", err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("the shipped %s does not parse: %v", ManifestRel, err)
	}
	pi := m.Lookup("pi")
	if pi == nil {
		t.Fatal("the shipped manifest must declare pi")
	}
	if pi.Mode != "0600" {
		t.Errorf("pi may carry a credential; want mode 0600, got %q", pi.Mode)
	}
}

// The first deploy over a file the machine already had keeps that file once as
// <dst>.pre-dotf, so adopting a hand-made config (an rc file, say) never loses
// it. A later deploy never overwrites the backup: it holds the machine's own
// version, not dotf's.
func TestDeploy_KeepsThePreviousFileOnceBeforeReplacingIt(t *testing.T) {
	root := repoWithSource(t, "repo v1\n")
	home := t.TempDir()
	c := piConfig()
	dst, err := ExpandDst(c.Dst, home, noResolve)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("hand-made\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	backup := dst + BackupSuffix
	if res.BackedUp != backup {
		t.Errorf("outcome names backup %q, want %q", res.BackedUp, backup)
	}
	if got, _ := os.ReadFile(backup); string(got) != "hand-made\n" {
		t.Errorf("backup holds %q", got)
	}

	// A changed source redeploys; the machine's original stays the backup.
	if err := os.WriteFile(filepath.Join(root, c.Src), []byte("repo v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = Deploy(c, root, home, noResolve, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.BackedUp != "" {
		t.Errorf("a second replacement made another backup: %q", res.BackedUp)
	}
	if got, _ := os.ReadFile(backup); string(got) != "hand-made\n" {
		t.Errorf("the backup was overwritten: %q", got)
	}
}

func TestDeploy_AFreshDestinationNeedsNoBackup(t *testing.T) {
	root := repoWithSource(t, "repo\n")
	res, err := Deploy(piConfig(), root, t.TempDir(), noResolve, nil, false)
	if err != nil || res.BackedUp != "" {
		t.Fatalf("want no backup on a fresh destination, got %q, %v", res.BackedUp, err)
	}
	if _, err := os.Stat(res.Dst + BackupSuffix); !os.IsNotExist(err) {
		t.Error("a backup file was created for a destination that did not exist")
	}
}

// The backup keeps the machine's previous content, so it must not outlive the
// protection the manifest gives the file that replaces it: a 0644 copy of a
// settings file that may carry a credential, left beside the new 0600 one.
func TestDeploy_TheBackupNeverHoldsMoreAccessThanTheDeclaredMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	for _, render := range []bool{false, true} {
		root := repoWithSource(t, "repo\n")
		home := t.TempDir()
		c := piConfig() // declares 0600
		c.Render = render
		dst, err := ExpandDst(c.Dst, home, noResolve)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte("token=old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dst, 0o644); err != nil { // umask-proof
			t.Fatal(err)
		}

		res, err := Deploy(c, root, home, noResolve, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(res.BackedUp)
		if err != nil {
			t.Fatalf("render=%v: no backup: %v", render, err)
		}
		if got := info.Mode().Perm(); got&^0o600 != 0 {
			t.Errorf("render=%v: backup mode %o is wider than the declared 0600", render, got)
		}
	}
}
