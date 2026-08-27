package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// piExtFixture builds a fake $HOME carrying a pi agent tree and returns it.
//
//	extensions/<name>/<file>  -> symlink to target, when target != ""
//	extensions/<name>/<file>  -> plain file, when target == ""
//
// installed names are unpacked under npm/node_modules, which is what
// installedPiPackages reads: the check's whole premise is that what is ON DISK
// and what settings.json DECLARES can disagree, so the fixture never writes the
// declaration.
type piExtEntry struct {
	dir    string
	file   string
	target string
}

func piExtFixture(t *testing.T, installed []string, entries []piExtEntry) string {
	t.Helper()
	home := t.TempDir()
	agent := filepath.Join(home, ".pi", "agent")

	for _, name := range installed {
		if err := os.MkdirAll(filepath.Join(agent, "npm", "node_modules", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, e := range entries {
		dir := filepath.Join(agent, "extensions", e.dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, e.file)
		if e.target == "" {
			if err := os.WriteFile(p, []byte("// plain extension\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		// The target must exist, so EvalSymlinks resolves rather than falling
		// back to the raw link — the production path this exercises.
		if err := os.MkdirAll(filepath.Dir(e.target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(e.target, []byte("// linked source\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(e.target, p); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func piExtRun(t *testing.T, home string, fix bool) string {
	t.Helper()
	var buf bytes.Buffer
	rep := NewReport(&buf, true)
	checkPiExtensions(piSys(home), &Config{DotfilesDir: t.TempDir()}, rep, fix)
	rep.flush()
	return buf.String()
}

func requireSymlinks(t *testing.T) {
	t.Helper()
	// Creating a symlink on Windows needs either developer mode or elevation,
	// and the Windows leg of CI compiles and runs this package. Skipping is
	// honest; the behaviour under test is a POSIX-only tree (~/.pi wired by
	// hand on Linux), so there is nothing to assert on the other platform.
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privileged on Windows; the shadow shape is POSIX-only")
	}
}

// The measured defect, 2026-08-26: extensions/subagent/index.ts symlinked into
// pi's own bundled examples under node_modules, while npm:pi-subagents@0.56.0 is
// installed. pi exited 1 with `Tool "subagent" conflicts`.
func TestPiExtensionsShadowFailsWhenPackageInstalled(t *testing.T) {
	requireSymlinks(t)
	nm := filepath.Join(t.TempDir(), "node_modules", "@earendil-works", "pi-coding-agent", "examples", "extensions", "subagent")
	home := piExtFixture(t,
		[]string{"pi-subagents"},
		[]piExtEntry{{dir: "subagent", file: "index.ts", target: filepath.Join(nm, "index.ts")}},
	)

	out := piExtRun(t, home, false)
	if !strings.Contains(out, "[FAIL]") {
		t.Fatalf("a shadow over an installed package must FAIL, got:\n%s", out)
	}
	for _, want := range []string{"subagent", "shadows the installed package"} {
		if !strings.Contains(out, want) {
			t.Errorf("diagnostic missing %q, got:\n%s", want, out)
		}
	}
	// ADR-025: a literal home must never be printed.
	if strings.Contains(out, home) {
		t.Errorf("diagnostic printed a literal $HOME:\n%s", out)
	}
}

// The same link with nothing installed under that name is unreproducible but is
// not breaking anything. Warning, not failure — and --fix must leave it alone.
func TestPiExtensionsShadowWarnsWhenNoPackageCollides(t *testing.T) {
	requireSymlinks(t)
	nm := filepath.Join(t.TempDir(), "node_modules", "some-pkg")
	link := piExtEntry{dir: "handwired", file: "index.ts", target: filepath.Join(nm, "index.ts")}
	home := piExtFixture(t, nil, []piExtEntry{link})

	out := piExtRun(t, home, true)
	if strings.Contains(out, "[FAIL]") {
		t.Errorf("a non-colliding hand-wired link must not FAIL, got:\n%s", out)
	}
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("a non-colliding hand-wired link must WARN, got:\n%s", out)
	}
	if _, err := os.Lstat(filepath.Join(home, ".pi", "agent", "extensions", "handwired", "index.ts")); err != nil {
		t.Errorf("--fix moved a link it only warned about: %v", err)
	}
}

// The scoping rule, and the reason it is symlink-into-node_modules rather than
// "anything the manifest does not declare": ~/.pi/agent/extensions/ has a live
// external writer. Three orca-*.ts files appeared there while this check was
// being written. Claiming the directory would report that writer as drift.
func TestPiExtensionsIgnoresPlainFilesAndUnrelatedLinks(t *testing.T) {
	requireSymlinks(t)
	elsewhere := filepath.Join(t.TempDir(), "sources", "thing.ts")
	home := piExtFixture(t,
		[]string{"pi-subagents"},
		[]piExtEntry{
			{dir: ".", file: "orca-titlebar-spinner.ts"},         // plain file, external writer
			{dir: "mine", file: "index.ts"},                      // plain hand-authored extension
			{dir: "linked", file: "index.ts", target: elsewhere}, // symlink, but not into node_modules
		},
	)

	out := piExtRun(t, home, false)
	if strings.Contains(out, "[FAIL]") || strings.Contains(out, "[WARN]") {
		t.Fatalf("no entry here is a shadow, got:\n%s", out)
	}
	if !strings.Contains(out, "[ OK ]") {
		t.Fatalf("expected an OK, got:\n%s", out)
	}
	for _, banned := range []string{"orca", "mine", "linked"} {
		if strings.Contains(out, banned) {
			t.Errorf("check reported %q, which it does not own:\n%s", banned, out)
		}
	}
}

func TestPiExtensionsSkipsWhenNoExtensionsDir(t *testing.T) {
	out := piExtRun(t, t.TempDir(), false)
	if !strings.Contains(out, "[SKIP]") {
		t.Fatalf("an absent extensions dir must SKIP, got:\n%s", out)
	}
}

// The property that makes quarantine a repair rather than a relocation: the
// destination is OUTSIDE the tree pi auto-discovers (docs/extensions.md lists
// `extensions/*/index.ts` as a discovery pattern, which a .disabled/ subdir
// would match), and therefore outside the tree this check re-reads. A second
// run must report the machine as clean.
func TestPiExtensionsFixQuarantinesOutsideTheScannedTree(t *testing.T) {
	requireSymlinks(t)
	nm := filepath.Join(t.TempDir(), "node_modules", "pi-coding-agent", "examples", "extensions", "subagent")
	target := filepath.Join(nm, "index.ts")
	home := piExtFixture(t,
		[]string{"pi-subagents"},
		[]piExtEntry{{dir: "subagent", file: "index.ts", target: target}},
	)

	out := piExtRun(t, home, true)
	if !strings.Contains(out, "[FIX ]") {
		t.Fatalf("--fix must repair a colliding shadow, got:\n%s", out)
	}

	agent := filepath.Join(home, ".pi", "agent")
	if _, err := os.Lstat(filepath.Join(agent, "extensions", "subagent", "index.ts")); !os.IsNotExist(err) {
		t.Errorf("the shadowing link is still in the scanned tree (err=%v)", err)
	}
	quarantined := filepath.Join(agent, piQuarantineDir, "subagent", "index.ts")
	if _, err := os.Lstat(quarantined); err != nil {
		t.Fatalf("quarantined link not at the documented path: %v", err)
	}
	if strings.HasPrefix(quarantined, filepath.Join(agent, "extensions")+string(filepath.Separator)) {
		t.Error("quarantine landed inside extensions/, where pi may still discover it")
	}
	// The target belongs to pi's npm package and is never in scope.
	if _, err := os.Stat(target); err != nil {
		t.Errorf("--fix touched the link target, which it must never do: %v", err)
	}

	// A second run sees a clean machine: the quarantine is not re-reported.
	again := piExtRun(t, home, false)
	if !strings.Contains(again, "[ OK ]") {
		t.Fatalf("after --fix the check must report OK, got:\n%s", again)
	}
}

// Two --fix runs where the first was restored by hand must not silently destroy
// the earlier quarantine. Two runs that both report success while the second
// overwrites the first is the failure mode this whole check exists to catch.
func TestPiExtensionsFixNeverClobbersAnEarlierQuarantine(t *testing.T) {
	requireSymlinks(t)
	nm := filepath.Join(t.TempDir(), "node_modules", "pi-subagents-src")
	home := piExtFixture(t,
		[]string{"pi-subagents"},
		[]piExtEntry{{dir: "subagent", file: "index.ts", target: filepath.Join(nm, "index.ts")}},
	)
	agent := filepath.Join(home, ".pi", "agent")

	if out := piExtRun(t, home, true); !strings.Contains(out, "[FIX ]") {
		t.Fatalf("first --fix did not repair:\n%s", out)
	}
	// Simulate the user restoring the link, then running --fix again.
	relinked := filepath.Join(agent, "extensions", "subagent", "index.ts")
	if err := os.Symlink(filepath.Join(nm, "index.ts"), relinked); err != nil {
		t.Fatal(err)
	}

	out := piExtRun(t, home, true)
	if !strings.Contains(out, "already quarantined") {
		t.Errorf("second --fix must refuse rather than clobber, got:\n%s", out)
	}
	if _, err := os.Lstat(relinked); err != nil {
		t.Errorf("the link was moved over an existing quarantine: %v", err)
	}
}
