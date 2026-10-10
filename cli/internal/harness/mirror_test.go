package harness

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mirrorRepo builds a checkout with a harness/ tree and a manifest declaring
// two targets outside it — the shape setup-linux.sh's bash block modelled.
func mirrorRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"),
		`{"targets":[{"file":"AGENTS.md"},{"file":"ai/claude/CLAUDE.md"},{"file":"AGENTS.md"}]}`)
	writeFile(t, filepath.Join(repo, "harness", "model-map.json"), `{"pools":{}}`)
	writeFile(t, filepath.Join(repo, "harness", "skills", "handoff", "SKILL.md"), "# handoff\n")
	writeFile(t, filepath.Join(repo, "AGENTS.md"), "# AGENTS\n")
	writeFile(t, filepath.Join(repo, "ai", "claude", "CLAUDE.md"), "# CLAUDE\n")
	return repo
}

func TestMirror_CopiesTheTreeAndEveryDeclaredTarget(t *testing.T) {
	repo, deploy := mirrorRepo(t), t.TempDir()

	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"harness/manifest.json", "harness/model-map.json", "harness/skills/handoff/SKILL.md",
		"AGENTS.md", "ai/claude/CLAUDE.md",
	} {
		if _, err := os.Stat(filepath.Join(deploy, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s not mirrored: %v", rel, err)
		}
	}
	if res.Updated != 5 || res.Unchanged != 0 {
		t.Errorf("first run: want 5 updated / 0 unchanged, got %d / %d", res.Updated, res.Unchanged)
	}
	// The duplicate declaration is one target, not two copies.
	if got := strings.Join(res.Targets, ","); got != "AGENTS.md,ai/claude/CLAUDE.md" {
		t.Errorf("targets: %s", got)
	}
}

// A re-run must report zero changes — that number is the idempotence evidence
// a setup run prints (#1266) — and must not touch an identical file's mtime.
func TestMirror_IsIdempotentAndDoesNotRewriteIdenticalFiles(t *testing.T) {
	repo, deploy := mirrorRepo(t), t.TempDir()
	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(deploy, "harness", "model-map.json")
	before, _ := os.Stat(target)

	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 0 || res.Unchanged != 5 {
		t.Errorf("re-run: want 0 updated / 5 unchanged, got %d / %d", res.Updated, res.Unchanged)
	}
	after, _ := os.Stat(target)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("an identical file was rewritten (mtime churned)")
	}

	// A changed source is the only thing that earns a write.
	writeFile(t, filepath.Join(repo, "harness", "model-map.json"), `{"pools":{"x":{}}}`)
	res, err = Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 {
		t.Errorf("one changed source: want 1 updated, got %d", res.Updated)
	}
	got, _ := os.ReadFile(target)
	if string(got) != `{"pools":{"x":{}}}` {
		t.Errorf("mirror not refreshed: %s", got)
	}
}

// A declared target the checkout lacks is named and reported as an error —
// after everything else was mirrored. Skipping it silently reproduces #1200:
// the drift check later evaluates a file the mirror never received.
func TestMirror_NamesADeclaredTargetTheCheckoutLacks(t *testing.T) {
	repo, deploy := mirrorRepo(t), t.TempDir()
	if err := os.Remove(filepath.Join(repo, "ai", "claude", "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}

	res, err := Mirror(repo, deploy)
	if !errors.Is(err, ErrMissingTargets) {
		t.Fatalf("want ErrMissingTargets, got %v", err)
	}
	if len(res.Missing) != 1 || res.Missing[0] != "ai/claude/CLAUDE.md" {
		t.Errorf("missing must name the target: %v", res.Missing)
	}
	if _, err := os.Stat(filepath.Join(deploy, "AGENTS.md")); err != nil {
		t.Error("the other target must still have been mirrored")
	}
	if _, err := os.Stat(filepath.Join(deploy, "harness", "model-map.json")); err != nil {
		t.Error("the harness tree must still have been mirrored")
	}
}

// Mirroring never prunes harness/: a record only the mirror has survives.
// Orphan removal there is `dotf doctor --fix`'s (#802). Only the trees of plain
// checkout copies are pruned by Mirror (prune_test.go).
func TestMirror_DoesNotPrune(t *testing.T) {
	repo, deploy := mirrorRepo(t), t.TempDir()
	orphan := filepath.Join(deploy, "harness", "skills", "retired", "SKILL.md")
	writeFile(t, orphan, "# retired\n")

	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Error("mirror must not prune; that is doctor --fix's job")
	}
}

// The checkout and the deploy dir being the same directory is a stated
// outcome, not a copy of a tree onto itself.
func TestMirror_RefusesWhenTheCheckoutIsTheDeployDir(t *testing.T) {
	repo := mirrorRepo(t)
	if _, err := Mirror(repo, repo); !errors.Is(err, ErrCheckoutIsDeployDir) {
		t.Fatalf("want ErrCheckoutIsDeployDir, got %v", err)
	}
}

// A target is a path inside the checkout, mirrored to the same path inside the
// deploy dir. One that climbs out with ".." or is absolute would write
// anywhere the user can (#1872), so the manifest is refused before any write.
func TestMirror_RefusesATargetThatEscapes(t *testing.T) {
	for _, target := range []string{"../escaped.md", "ai/../../escaped.md", "/tmp/escaped.md"} {
		t.Run(target, func(t *testing.T) {
			parent := t.TempDir()
			repo, deploy := filepath.Join(parent, "repo"), filepath.Join(parent, "deploy")
			writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"targets":[{"file":"`+target+`"}]}`)
			writeFile(t, filepath.Join(parent, "escaped.md"), "outside\n")

			_, err := Mirror(repo, deploy)
			if err == nil || !strings.Contains(err.Error(), target) {
				t.Fatalf("an escaping target must be refused by name, got %v", err)
			}
			if _, err := os.Stat(filepath.Join(deploy, "harness")); err == nil {
				t.Error("nothing may be mirrored from a manifest that is refused")
			}
		})
	}
}

func TestMirror_FailsLoudWithoutAManifest(t *testing.T) {
	repo, deploy := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "model-map.json"), `{}`)
	if _, err := Mirror(repo, deploy); err == nil || !strings.Contains(err.Error(), ManifestFile) {
		t.Fatalf("a missing manifest must name itself, got %v", err)
	}
}

func TestMirror_PreservesTheSourceMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows filesystems do not expose POSIX executable mode bits")
	}
	repo, deploy := mirrorRepo(t), t.TempDir()
	src := filepath.Join(repo, "harness", "skills", "handoff", "find-polluter.sh")
	writeFile(t, src, "#!/usr/bin/env bash\n")
	if err := os.Chmod(src, 0o775); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(deploy, "harness", "skills", "handoff", "find-polluter.sh")
	writeFile(t, dst, "#!/usr/bin/env bash\n")
	if err := os.Chmod(dst, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o775 {
		t.Fatalf("destination mode = %#o, want %#o", got, 0o775)
	}
}

func TestMirror_ReplacesReadOnlyDestinationOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows refuses to rename over an existing read-only file")
	}
	repo, deploy := mirrorRepo(t), t.TempDir()
	src := filepath.Join(repo, "harness", "model-map.json")
	if err := os.Chmod(src, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(src, 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, src, `{"pools":{"updated":{}}}`)
	if err := os.Chmod(src, 0o444); err != nil {
		t.Fatal(err)
	}

	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatalf("replacing a read-only destination: %v", err)
	}
	dst := filepath.Join(deploy, "harness", "model-map.json")
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"pools":{"updated":{}}}` {
		t.Fatalf("destination content = %s", got)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o444 {
		t.Fatalf("destination mode = %#o, want %#o", got, 0o444)
	}
	res, err := Mirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 0 || res.Unchanged != 5 {
		t.Fatalf("converged re-run: want 0 updated / 5 unchanged, got %d / %d", res.Updated, res.Unchanged)
	}
}

// PlanMirror answers "what would Mirror write" through the same comparison
// Mirror uses, and writes nothing — not even the deploy dir. A plan that
// creates directories has answered a different question than it was asked.
func TestPlanMirror_CountsWhatMirrorWouldWriteAndWritesNothing(t *testing.T) {
	repo := mirrorRepo(t)
	deploy := filepath.Join(t.TempDir(), "absent")

	plan, err := PlanMirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Updated != 5 || plan.Unchanged != 0 {
		t.Errorf("plan on an empty deploy dir: want 5 to write / 0 unchanged, got %d / %d", plan.Updated, plan.Unchanged)
	}
	if _, err := os.Stat(deploy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("PlanMirror created the deploy dir: %v", err)
	}

	if _, err := Mirror(repo, deploy); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanMirror(repo, deploy)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Updated != 0 || plan.Unchanged != 5 {
		t.Errorf("plan after a mirror: want 0 to write / 5 unchanged, got %d / %d", plan.Updated, plan.Unchanged)
	}
}
