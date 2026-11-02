package harness

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const skillRecord = `---
name: demo
description: a demo skill
generated: true
generated_from: 00_meta/skills/demo/SKILL.md
generated_sha: 0000000000000000
paths:
  - "**/*.go"
allowed-tools:
  - Read
id: DEMO-001
---

# Demo

Body.

---

After a rule.
`

func TestRenderSkillRecord(t *testing.T) {
	sha := PresenceSHA(skillRecord)
	cases := []struct {
		name, kind, want string
	}{
		{"skill keeps the native keys and their continuation lines", RenderSkill, "---\n" +
			"generated: true\ngenerated_from: src\ngenerated_sha: " + sha + "\n" +
			"name: demo\ndescription: a demo skill\nallowed-tools:\n  - Read\n---\n\n# Demo\n\nBody.\n\n---\n\nAfter a rule.\n"},
		{"command drops name too", RenderCommand, "---\n" +
			"generated: true\ngenerated_from: src\ngenerated_sha: " + sha + "\n" +
			"description: a demo skill\nallowed-tools:\n  - Read\n---\n\n# Demo\n\nBody.\n\n---\n\nAfter a rule.\n"},
		{"prompt keeps the body's rule", RenderPrompt,
			"<!-- generated: true; from: src; sha256:" + sha + "; edit the vault source + re-run setup -->\n\n" +
				"\n# Demo\n\nBody.\n\n---\n\nAfter a rule.\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RenderSkillRecord(tc.kind, []byte(skillRecord), "src"); got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

// awk reads a final line without a newline as a line and ends every line it
// prints with one, and drops whatever precedes the first key.
func TestRenderSkillRecord_AwkLineSemantics(t *testing.T) {
	got := RenderSkillRecord(RenderSkill, []byte("---\n# note\nname: x\n---\nlast"), "s")
	if !strings.HasSuffix(got, "---\nlast\n") || strings.Contains(got, "# note") {
		t.Errorf("got:\n%s", got)
	}
}

func TestRenderSkillRecord_CRLFRendersTheLFBytes(t *testing.T) {
	crlf := strings.ReplaceAll(skillRecord, "\n", "\r\n")
	for _, kind := range []string{RenderSkill, RenderCommand, RenderPrompt} {
		if RenderSkillRecord(kind, []byte(crlf), "s") != RenderSkillRecord(kind, []byte(skillRecord), "s") {
			t.Errorf("%s: a CRLF record renders different bytes from its LF form", kind)
		}
	}
}

func TestSkillTargetsAgent(t *testing.T) {
	cases := []struct {
		name, front, agent string
		want, wantErr      bool
	}{
		{"absent means every agent", "name: x", "pi", true, false},
		{"inline list, listed", "targets: [claude, pi]", "pi", true, false},
		{"inline list names copilot, not pi (#1733)", "targets: [copilot]", "pi", false, false},
		{"block list is read", "targets:\n  - opencode\n  - agy", "agy", true, false},
		{"a single name", "targets: claude", "claude", true, false},
		{"names are case-sensitive, as the shell's match was", "targets: [Claude]", "claude", false, false},
		{"an empty list is nobody", "targets: []", "claude", false, false},
		{"a null value is nobody", "targets:", "claude", false, false},
		{"a non-name entry is an error", "targets: [{a: b}]", "claude", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SkillTargetsAgent([]byte("---\n"+tc.front+"\n---\nbody\n"), tc.agent)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Errorf("got %v, %v; want %v, error %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

// Every committed record must parse, or the deploy refuses it.
func TestSkillTargetsAgent_EveryCommittedRecordParses(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "..", "..", "harness", "skills", "*", "SKILL.md"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no committed records found: %v", err)
	}
	for _, p := range dirs {
		raw, _ := os.ReadFile(p) //nolint:gosec // committed records
		if _, err := SkillTargetsAgent(raw, "claude"); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func skillsRepo(t *testing.T) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"skills":{
		"vault_subpath":"00_meta/skills","record_dir":"harness/skills","deploy":[
		{"agent":"claude","render":"skill","dir":".claude/skills"},
		{"agent":"opencode","render":"command","dir":".config/opencode/commands"},
		{"agent":"copilot","render":"skill","dir":".copilot/skills","requires_command":"copilot"}]}}`)
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", "SKILL.md"), skillRecord)
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", "tools", "helper.sh"), "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(repo, "harness", "skills", "demo", "tools", "helper.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "harness", "skills", "only-claude", "SKILL.md"), "---\nname: only-claude\ntargets: [claude]\n---\nx\n")
	return repo, home
}

func countStatus(out []SkillOutcome, status string) int {
	n := 0
	for _, o := range out {
		if o.Status == status {
			n++
		}
	}
	return n
}

func TestDeploySkills_FreshThenConverged(t *testing.T) {
	repo, home := skillsRepo(t)
	out, err := DeploySkills(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	if countStatus(out, SkillDeployed) != 3 || countStatus(out, SkillSkipped) != 1 {
		t.Fatalf("outcomes: %+v", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "commands", "only-claude.md")); err == nil {
		t.Error("a claude-only record reached opencode")
	}
	helper, err := os.Stat(filepath.Join(home, ".claude", "skills", "demo", "tools", "helper.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && helper.Mode().Perm() != 0o755 {
		t.Errorf("helper mode %v, want 0755", helper.Mode().Perm())
	}
	again, err := DeploySkills(repo, home, none, false)
	if err != nil || countStatus(again, SkillDeployed) != 0 || countStatus(again, SkillCurrent) != 3 {
		t.Errorf("second run: %+v, %v", again, err)
	}
}

func TestDeploySkills_DryRunWritesNothing(t *testing.T) {
	repo, home := skillsRepo(t)
	out, err := DeploySkills(repo, home, none, true)
	if err != nil || countStatus(out, SkillDeployed) != 3 {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("a dry run wrote %d entries", len(entries))
	}
}

func TestDeploySkills_ReplacesASymlinkedSkillDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	repo, home := skillsRepo(t)
	vault := t.TempDir()
	writeFile(t, filepath.Join(vault, "SKILL.md"), "vault copy\n")
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".claude", "skills", "demo")
	if err := os.Symlink(vault, link); err != nil {
		t.Fatal(err)
	}
	if _, err := DeploySkills(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink != 0 {
		t.Error("the symlink is still there")
	}
	if got, _ := os.ReadFile(filepath.Join(vault, "SKILL.md")); string(got) != "vault copy\n" {
		t.Error("the deploy wrote through the symlink into its target")
	}
}

func TestDeploySkills_PrunesOnlyGeneratedOutputs(t *testing.T) {
	repo, home := skillsRepo(t)
	stale := filepath.Join(home, ".claude", "skills", "retired", "SKILL.md")
	handMade := filepath.Join(home, ".claude", "skills", "mine", "SKILL.md")
	shadow := filepath.Join(home, ".config", "opencode", "commands", "demo.md")
	uninstalled := filepath.Join(home, ".copilot", "skills", "retired", "SKILL.md")
	writeFile(t, stale, "---\ngenerated: true\n---\n")
	writeFile(t, handMade, "---\nname: mine\n---\n")
	writeFile(t, shadow, "hand-written, no marker\n")
	writeFile(t, uninstalled, "---\ngenerated: true\n---\n")

	out, err := DeploySkills(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]bool{stale: false, handMade: true, uninstalled: false} {
		if _, err := os.Stat(p); (err == nil) != want {
			t.Errorf("%s: exists=%v, want %v", p, err == nil, want)
		}
	}
	// The unmarked command is reported, and the render does not touch it either
	// way: it is overwritten as an output, never pruned as one.
	if countStatus(out, SkillUnmanaged) != 0 || countStatus(out, SkillPruned) != 2 {
		t.Errorf("outcomes: %+v", out)
	}
}

func TestDeploySkills_ReportsAnUnmarkedCopyItCannotPrune(t *testing.T) {
	repo, home := skillsRepo(t)
	unmarked := filepath.Join(home, ".copilot", "skills", "demo", "SKILL.md")
	writeFile(t, unmarked, "---\nname: demo\n---\nold copy\n")
	out, err := DeploySkills(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	if countStatus(out, SkillUnmanaged) != 1 {
		t.Errorf("outcomes: %+v", out)
	}
	if _, err := os.Stat(unmarked); err != nil {
		t.Error("an unmarked copy was removed")
	}
}

// A record owns its deployed directory: a helper the record no longer ships
// leaves every agent's copy, so a renamed or deleted script is not offered to
// the agent forever (#2851). Top-level dotfiles stay, as the copy skips them.
func TestDeploySkills_DropsAuxFilesTheRecordNoLongerShips(t *testing.T) {
	repo, home := skillsRepo(t)
	if _, err := DeploySkills(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	deployed := filepath.Join(home, ".claude", "skills", "demo")
	finder := filepath.Join(deployed, ".DS_Store")
	writeFile(t, finder, "x")
	if err := os.RemoveAll(filepath.Join(repo, "harness", "skills", "demo", "tools")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", "renamed.sh"), "#!/bin/sh\n")

	out, err := DeploySkills(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(deployed, "tools")); err == nil {
		t.Error("the dropped helper's directory is still deployed")
	}
	for _, keep := range []string{"SKILL.md", "renamed.sh", ".DS_Store"} {
		if _, err := os.Stat(filepath.Join(deployed, keep)); err != nil {
			t.Errorf("%s is gone: %v", keep, err)
		}
	}
	if countStatus(out, SkillDeployed) != 1 {
		t.Errorf("the claude demo output should report deployed: %+v", out)
	}
	again, err := DeploySkills(repo, home, none, false)
	if err != nil || countStatus(again, SkillDeployed) != 0 {
		t.Errorf("second run: %+v, %v", again, err)
	}
}

// A directory whose SKILL.md is not ours yet keeps its other files on the run
// that first renders into it; from the next run, the record owns it.
func TestDeploySkills_AnUnmarkedDirKeepsItsFilesUntilTheRenderOwnsIt(t *testing.T) {
	repo, home := skillsRepo(t)
	deployed := filepath.Join(home, ".claude", "skills", "demo")
	extra := filepath.Join(deployed, "notes.md")
	writeFile(t, filepath.Join(deployed, "SKILL.md"), "---\nname: demo\n---\nhand-written\n")
	writeFile(t, extra, "mine\n")
	if _, err := DeploySkills(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(extra); err != nil {
		t.Fatal("the first render into an unmarked directory removed a file in it")
	}
	if _, err := DeploySkills(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(extra); err == nil {
		t.Error("the record owns the directory after its first render, and the extra file is still there")
	}
}

// A record file that is a symlink is copied as its target's bytes, so it is
// still shipped: the next run must not remove what this run wrote.
func TestDeploySkills_KeepsAnAuxFileTheRecordShipsAsASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	repo, home := skillsRepo(t)
	rec := filepath.Join(repo, "harness", "skills", "demo")
	writeFile(t, filepath.Join(rec, "shared", "ref.md"), "reference\n")
	if err := os.Symlink(filepath.Join("shared", "ref.md"), filepath.Join(rec, "ref.md")); err != nil {
		t.Fatal(err)
	}
	for run := 1; run <= 2; run++ {
		out, err := DeploySkills(repo, home, none, false)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "demo", "ref.md")); err != nil || string(got) != "reference\n" {
			t.Fatalf("run %d: ref.md = %q, %v", run, got, err)
		}
		if run == 2 && countStatus(out, SkillDeployed) != 0 {
			t.Errorf("the second run does not converge: %+v", out)
		}
	}
}

func TestDeploySkills_ADryRunKeepsTheDroppedAuxFile(t *testing.T) {
	repo, home := skillsRepo(t)
	if _, err := DeploySkills(repo, home, none, false); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, "harness", "skills", "demo", "tools")); err != nil {
		t.Fatal(err)
	}
	out, err := DeploySkills(repo, home, none, true)
	if err != nil || countStatus(out, SkillDeployed) != 1 {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "demo", "tools", "helper.sh")); err != nil {
		t.Error("a dry run removed the helper")
	}
}

// Prune owns an output by its provenance, not by a substring: a hand-written
// skill that quotes the marker in its body is not removed with its directory.
func TestIsGeneratedOutput(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"a rendered skill", "---\ngenerated: true\ngenerated_from: x\nname: a\n---\nbody\n", true},
		{"a CRLF rendered skill", "---\r\ngenerated: true\r\n---\r\n", true},
		{"a rendered prompt", "<!-- generated: true; from: x; sha256:0; edit the vault source + re-run setup -->\n\nbody\n", true},
		{"the marker quoted in the body", "---\nname: mine\n---\nWrite `generated: true` in the frontmatter.\ngenerated: true\n", false},
		{"the marker in a later block", "---\nname: mine\n---\n---\ngenerated: true\n---\n", false},
		{"a frontmatter that never closes", "---\ngenerated: true\n", false},
		{"no frontmatter", "generated: true\n", false},
		{"a value after the marker", "---\ngenerated: true-ish\n---\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isGeneratedOutput([]byte(tc.body)); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDeploySkills_KeepsAHandWrittenSkillThatQuotesTheMarker(t *testing.T) {
	repo, home := skillsRepo(t)
	mine := filepath.Join(home, ".claude", "skills", "mine", "SKILL.md")
	writeFile(t, mine, "---\nname: mine\n---\nRecords carry generated: true in their frontmatter.\n")
	out, err := DeploySkills(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); err != nil || countStatus(out, SkillPruned) != 0 {
		t.Errorf("a hand-written skill was pruned: %+v", out)
	}
}

func TestDeploySkills_ABadRecordFailsAfterTheRest(t *testing.T) {
	repo, home := skillsRepo(t)
	writeFile(t, filepath.Join(repo, "harness", "skills", "broken", "SKILL.md"), "---\ntargets: [\n---\n")
	out, err := DeploySkills(repo, home, none, false)
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("error: %v", err)
	}
	if countStatus(out, SkillDeployed) != 3 {
		t.Errorf("the other records did not deploy: %+v", out)
	}
}

func TestLoadSkills_RejectsAnUnknownRender(t *testing.T) {
	p := filepath.Join(t.TempDir(), "manifest.json")
	writeFile(t, p, `{"skills":{"record_dir":"r","deploy":[{"agent":"a","render":"html","dir":"d"}]}}`)
	if _, err := LoadSkills(p); err == nil {
		t.Error("an unknown render kind was accepted")
	}
}
