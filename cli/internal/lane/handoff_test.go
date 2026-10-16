package lane

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// fakeBackend records what a hand-off asks of the session backend. It has no
// remove method because Backend has none.
type fakeBackend struct {
	agent    string
	existing map[string]Worktree
	calls    []string
	typed    map[string]string
	created  string // the path Create answers with, as herdr's answer does
	last     string // the worktree the last Create or Open made available
	// leased records, per pane, whether the worktree had its lease when the
	// agent's command was typed: the lease must exist before anything runs.
	leased map[string]bool
}

func newFake(t *testing.T, agent string) *fakeBackend {
	t.Helper()
	return &fakeBackend{agent: agent, created: t.TempDir()}
}

func (f *fakeBackend) CallingAgent() (string, error) { return f.agent, nil }
func (f *fakeBackend) Find(repo, branch string) (Worktree, bool, error) {
	f.calls = append(f.calls, "find "+repo+" "+branch)
	w, ok := f.existing[branch]
	return w, ok, nil
}
func (f *fakeBackend) Create(repo, branch, base, label string) (Started, error) {
	f.calls = append(f.calls, "create "+repo+" "+branch+" "+base+" "+label)
	f.last = f.created
	return Started{Pane: "w1:p1", Worktree: Worktree{Path: f.created, Branch: branch, Open: true}}, nil
}
func (f *fakeBackend) Open(repo string, wt Worktree) (Started, error) {
	f.calls = append(f.calls, "open "+repo+" "+wt.Path)
	f.last = wt.Path
	return Started{Pane: "w2:p1", Worktree: Worktree{Path: wt.Path, Branch: wt.Branch, Open: true}}, nil
}
func (f *fakeBackend) Type(pane, command string) error {
	f.calls = append(f.calls, "type "+pane)
	if f.typed == nil {
		f.typed = map[string]string{}
	}
	f.typed[pane] = command
	if f.leased == nil {
		f.leased = map[string]bool{}
	}
	_, err := os.Stat(filepath.Join(f.last, worktree.MetadataFileName))
	f.leased[pane] = err == nil
	return nil
}

type mapSource map[int]Issue

func (m mapSource) Issue(n int) (Issue, error) {
	if is, ok := m[n]; ok {
		return is, nil
	}
	return Issue{}, errors.New("no such issue")
}

func opts(t *testing.T) Options {
	t.Helper()
	return Options{
		Config:    Config{Queue: []int{7, 8, 9}, BranchPrefix: "fix", Base: "origin/main"},
		Template:  "Work on #{{.Number}} {{.Title}} on {{.Branch}}; then {{len .Remaining}} more.",
		Parent:    "/repos/dotfiles",
		PromptDir: filepath.Join(t.TempDir(), "lane"),
		// An off-default root: a printed path built from anything but this is wrong.
		WorktreeRoot: filepath.Join("/srv", "wt"),
		GitDir:       t.TempDir(),
		Now:          func() time.Time { return leaseNow },
	}
}

var queue = mapSource{
	7: {Number: 7, Title: "BUG-7: first thing", State: "OPEN"},
	8: {Number: 8, Title: "BUG-8: second thing", State: "OPEN"},
	9: {Number: 9, Title: "BUG-9: third thing", State: "OPEN"},
}

// The caller may stand in a linked worktree: the successor is created through
// the repository's PARENT checkout, and the only thing the backend is asked is
// to find, create and type. Nothing in the interface can touch the caller's own
// worktree or branch.
func TestNext_FromALinkedWorktreeCreatesThroughTheParentAndLeavesTheCallerAlone(t *testing.T) {
	b := newFake(t, "claude")
	o := opts(t)
	out, err := Handoff(o, b, queue)
	if err != nil || !out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	want := []string{
		"find /repos/dotfiles fix/first-thing",
		"create /repos/dotfiles fix/first-thing origin/main #7 first-thing",
		"type w1:p1",
	}
	if !reflect.DeepEqual(b.calls, want) {
		t.Errorf("calls = %q, want %q", b.calls, want)
	}
	if cmd := b.typed["w1:p1"]; cmd != "dotf lane exec --agent claude --prompt-file "+PromptFile {
		t.Errorf("typed %q", cmd)
	}
	got, err := os.ReadFile(filepath.Join(b.created, PromptFile))
	if err != nil || string(got) != "Work on #7 BUG-7: first thing on fix/first-thing; then 2 more." {
		t.Errorf("prompt file = %q, %v", got, err)
	}
}

// Twice is once: after the first run a workspace shows the branch, so the
// second finds it open and starts nothing.
func TestNext_RunTwiceStartsOneAgent(t *testing.T) {
	shown := t.TempDir()
	b := &fakeBackend{agent: "claude", existing: map[string]Worktree{"fix/first-thing": {Path: shown, Branch: "fix/first-thing", Open: true}}}
	out, err := Handoff(opts(t), b, queue)
	if err != nil || out.Started || !strings.Contains(out.Note, "no second agent") {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	for _, c := range b.calls {
		if strings.HasPrefix(c, "create") || strings.HasPrefix(c, "type") || strings.HasPrefix(c, "open") {
			t.Errorf("unexpected call %q", c)
		}
	}
}

// A worktree that exists but that no workspace shows is opened, not recreated.
func TestNext_ReusesAWorktreeNothingShows(t *testing.T) {
	idle := t.TempDir()
	b := &fakeBackend{agent: "pi", existing: map[string]Worktree{"fix/first-thing": {Path: idle, Branch: "fix/first-thing"}}}
	out, err := Handoff(opts(t), b, queue)
	if err != nil || !out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	want := []string{"find /repos/dotfiles fix/first-thing", "open /repos/dotfiles " + idle, "type w2:p1"}
	if !reflect.DeepEqual(b.calls, want) {
		t.Errorf("calls = %q, want %q", b.calls, want)
	}
}

// The successor is the same kind of agent as the finisher unless told
// otherwise, and an agent the table lacks is refused naming the ones it has.
func TestNext_AgentDefaultsToTheCallingPaneAndUnknownIsRefused(t *testing.T) {
	b := newFake(t, "agy")
	out, err := Handoff(opts(t), b, queue)
	if err != nil || out.Agent != "agy" || !strings.Contains(b.typed["w1:p1"], "--agent agy ") {
		t.Fatalf("calling pane's agent: %+v, %v", out, err)
	}
	o := opts(t)
	o.Agent = "pi"
	if out, err := Handoff(o, newFake(t, "agy"), queue); err != nil || out.Agent != "pi" {
		t.Errorf("explicit agent: %+v, %v", out, err)
	}
	for _, agent := range []string{"codex", ""} {
		bb := newFake(t, agent)
		_, err = Handoff(opts(t), bb, queue)
		if err == nil || !strings.Contains(err.Error(), "agy, claude, pi") {
			t.Errorf("pane agent %q: want a refusal naming the known agents, got %v", agent, err)
		}
		if len(bb.calls) != 0 {
			t.Errorf("pane agent %q: refused after touching the backend: %q", agent, bb.calls)
		}
	}
}

// Without a backend nothing is created; the person gets the commands, exit 0.
func TestNext_NoBackendPrintsTheCommands(t *testing.T) {
	o := opts(t)
	o.Agent = "claude"
	out, err := Handoff(o, nil, queue)
	if err != nil || out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	// The worktree goes where `dotf worktree add` puts it (the layout rule is
	// worktree.ResolveWorktreePath's), so the branch's slug is the directory.
	dir := worktree.ResolveWorktreePath(o.WorktreeRoot, "dotfiles", "fix/first-thing")
	if want := filepath.Join("/srv", "wt", "dotfiles", "fix-first-thing"); dir != want {
		t.Fatalf("layout rule = %q, want %q", dir, want)
	}
	if len(out.Commands) != 4 || !strings.HasPrefix(out.Commands[1], "git -C /repos/dotfiles worktree add -b fix/first-thing "+dir+" origin/main") ||
		out.Commands[2] != "cd "+dir ||
		!strings.HasPrefix(out.Commands[3], "dotf lane exec --agent claude --prompt-file ") {
		t.Errorf("commands = %q", out.Commands)
	}
}

// With no backend to ask and no --agent, nothing may pick a harness: the
// commands are printed with a placeholder and the exit is still success (AC5).
func TestNext_NoBackendAndNoAgentPrintsAPlaceholder(t *testing.T) {
	o := opts(t)
	out, err := Handoff(o, nil, queue)
	if err != nil || out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	want := "dotf lane exec --agent <agy|claude|pi> --prompt-file "
	if len(out.Commands) != 4 || !strings.HasPrefix(out.Commands[3], want) {
		t.Errorf("commands = %q, want the last to start %q", out.Commands, want)
	}
	if !strings.Contains(out.Note, "replace <agy|claude|pi> with the agent to start") {
		t.Errorf("note = %q", out.Note)
	}
}

// A root that does not resolve would print a path in the wrong tree; refuse.
func TestNext_NoBackendWithoutAWorktreeRootRefuses(t *testing.T) {
	o := opts(t)
	o.Agent, o.WorktreeRoot = "claude", ""
	if _, err := Handoff(o, nil, queue); err == nil || !strings.Contains(err.Error(), "WORKTREE_ROOT") {
		t.Errorf("err = %v, want one naming WORKTREE_ROOT", err)
	}
}

// A binary path no shell reads alike is replaced by bare `dotf`; the outcome
// says so instead of reporting success on a different release.
func TestNext_ASubstitutedBinaryIsSaidSo(t *testing.T) {
	b := newFake(t, "claude")
	o := opts(t)
	o.Dotf = "/home/a b/dotf"
	out, err := Handoff(o, b, queue)
	if err != nil || !out.Started || !strings.Contains(out.Note, "runs `dotf` from its pane's PATH") {
		t.Errorf("Handoff = %+v, %v", out, err)
	}
	o.Dotf = "/opt/dotf/bin/dotf"
	if out, err = Handoff(o, newFake(t, "claude"), queue); err != nil || out.Note != "" {
		t.Errorf("a typeable binary needs no note, got %+v, %v", out, err)
	}
}

// A dry run decides and reports; it writes no prompt and creates nothing.
func TestNext_DryRunChangesNothing(t *testing.T) {
	b := newFake(t, "claude")
	o := opts(t)
	o.DryRun = true
	out, err := Handoff(o, b, queue)
	if err != nil || out.Started || len(out.Commands) == 0 {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	if _, err := os.Stat(o.PromptDir); !os.IsNotExist(err) {
		t.Errorf("a dry run wrote %s", o.PromptDir)
	}
	for _, c := range b.calls {
		if !strings.HasPrefix(c, "find") {
			t.Errorf("dry run called %q", c)
		}
	}
}

// The line typed into a pane carries no path anyone chose (#2537): a repository,
// a temp directory or a worktree root with a space in it must not reach the
// pane's shell. The prompt goes into the new worktree under a fixed name, and
// the line names it relative to the pane's directory.
func TestNext_APathWithASpaceReachesNoPane(t *testing.T) {
	spaced := filepath.Join(t.TempDir(), "Jane Doe")
	b := newFake(t, "claude")
	b.created = filepath.Join(spaced, "worktrees", "my repo", "fix-first-thing")
	if err := os.MkdirAll(b.created, 0o755); err != nil {
		t.Fatal(err)
	}
	o := opts(t)
	o.Parent = filepath.Join(spaced, "my repo")
	o.GitDir = filepath.Join(spaced, "my repo", ".git")
	o.PromptDir = filepath.Join(o.GitDir, "dotf-lane")
	out, err := Handoff(o, b, queue)
	if err != nil || !out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	cmd := b.typed["w1:p1"]
	if want := "dotf lane exec --agent claude --prompt-file " + PromptFile; cmd != want {
		t.Errorf("typed %q, want %q", cmd, want)
	}
	if strings.Contains(cmd, "Jane") || strings.Contains(cmd, spaced) {
		t.Errorf("the typed line carries a chosen path: %q", cmd)
	}
	if got, err := os.ReadFile(filepath.Join(b.created, PromptFile)); err != nil || !strings.HasPrefix(string(got), "Work on #7") {
		t.Errorf("prompt file = %q, %v", got, err)
	}
	// The file is excluded from git status in every worktree, like the lease.
	ex, err := os.ReadFile(filepath.Join(o.GitDir, "info", "exclude"))
	if err != nil || !strings.Contains(string(ex), PromptFile+"\n") {
		t.Errorf("info/exclude = %q, %v", ex, err)
	}
}

// A person pastes the printed commands into their own shell, so a path with a
// space is double-quoted there. A character live inside double quotes is
// refused: no quoting of it means the same in all four shells.
func TestNext_NoBackendQuotesAPathWithASpace(t *testing.T) {
	o := opts(t)
	o.Agent, o.DryRun = "claude", true // nothing is written: the paths are not real
	o.Parent = "/repos/my repo"
	o.WorktreeRoot = "/srv/my wt"
	o.PromptDir = "/repos/my repo/.git/dotf-lane"
	out, err := Handoff(o, nil, queue)
	if err != nil {
		t.Fatal(err)
	}
	dir := `"` + worktree.ResolveWorktreePath(o.WorktreeRoot, "my repo", "fix/first-thing") + `"`
	want := []string{
		`git -C "/repos/my repo" fetch origin`,
		`git -C "/repos/my repo" worktree add -b fix/first-thing ` + dir + ` origin/main`,
		`cd ` + dir,
		`dotf lane exec --agent claude --prompt-file "/repos/my repo/.git/dotf-lane/7.md"`,
	}
	if !reflect.DeepEqual(out.Commands, want) {
		t.Errorf("commands =\n%q\nwant\n%q", out.Commands, want)
	}
	for _, bad := range []string{`/repos/a"b`, "/repos/$HOME x", "/repos/a`b c"} {
		o.Parent = bad
		if _, err := Handoff(o, nil, queue); err == nil {
			t.Errorf("Parent %q: want a refusal", bad)
		}
	}
}

// A backslash is a path separator on Windows only. zsh and bash read it as an
// escape, so a POSIX path holding one is not a plain word (#2587).
func TestPlainWord_BackslashIsAPathSeparatorOnlyOnWindows(t *testing.T) {
	for _, tc := range []struct {
		goos, path string
		want       bool
	}{
		{"windows", `C:\Users\RUNNER~1\dotf.exe`, true},
		{"linux", `C:\Users\RUNNER~1\dotf.exe`, false},
		{"darwin", `/opt/a\b/dotf`, false},
		{"darwin", "/opt/dotf/bin/dotf", true},
		{"linux", "/home/a b/dotf", false},
		{"windows", "~/dotf", false},
	} {
		if got := plainWordFor(tc.goos).MatchString(tc.path); got != tc.want {
			t.Errorf("%s %q: plain = %v, want %v", tc.goos, tc.path, got, tc.want)
		}
	}
}

func TestNext_NothingFreeIsNotAnError(t *testing.T) {
	closed := mapSource{7: {Number: 7, State: "CLOSED"}, 8: {Number: 8, State: "CLOSED"}, 9: {Number: 9, State: "CLOSED"}}
	out, err := Handoff(opts(t), newFake(t, "claude"), closed)
	if err != nil || out.Started || out.Note == "" {
		t.Errorf("Handoff = %+v, %v", out, err)
	}
}

// The successor runs the dotf that decided, by absolute path, unless that path
// is not a word every shell reads alike; then it is `dotf` on the pane's PATH.
func TestNext_TheSuccessorRunsTheBinaryThatDecided(t *testing.T) {
	for dotf, want := range map[string]string{
		"/opt/dotf/bin/dotf": "/opt/dotf/bin/dotf lane exec ",
		"/home/a b/dotf":     "dotf lane exec ",
		"~/dotf":             "dotf lane exec ",
		"":                   "dotf lane exec ",
	} {
		b := newFake(t, "claude")
		o := opts(t)
		o.Dotf = dotf
		if _, err := Handoff(o, b, queue); err != nil || !strings.HasPrefix(b.typed["w1:p1"], want) {
			t.Errorf("Dotf %q: typed %q, %v; want prefix %q", dotf, b.typed["w1:p1"], err, want)
		}
	}
}

// With no backend a dry run still writes nothing.
func TestNext_NoBackendDryRunWritesNoPrompt(t *testing.T) {
	o := opts(t)
	o.Agent, o.DryRun = "claude", true
	if _, err := Handoff(o, nil, queue); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(o.PromptDir); !os.IsNotExist(err) {
		t.Errorf("a dry run wrote %s", o.PromptDir)
	}
}
