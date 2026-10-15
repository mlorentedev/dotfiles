package lane

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeBackend records what a hand-off asks of the session backend. It has no
// remove method because Backend has none.
type fakeBackend struct {
	agent    string
	existing map[string]Worktree
	calls    []string
	typed    map[string]string
}

func (f *fakeBackend) CallingAgent() (string, error) { return f.agent, nil }
func (f *fakeBackend) Find(repo, branch string) (Worktree, bool, error) {
	f.calls = append(f.calls, "find "+repo+" "+branch)
	w, ok := f.existing[branch]
	return w, ok, nil
}
func (f *fakeBackend) Create(repo, branch, base, label string) (Started, error) {
	f.calls = append(f.calls, "create "+repo+" "+branch+" "+base+" "+label)
	return Started{Pane: "w1:p1"}, nil
}
func (f *fakeBackend) Open(repo string, wt Worktree) (Started, error) {
	f.calls = append(f.calls, "open "+repo+" "+wt.Path)
	return Started{Pane: "w2:p1"}, nil
}
func (f *fakeBackend) Type(pane, command string) error {
	f.calls = append(f.calls, "type "+pane)
	if f.typed == nil {
		f.typed = map[string]string{}
	}
	f.typed[pane] = command
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
	b := &fakeBackend{agent: "claude"}
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
	cmd := b.typed["w1:p1"]
	if !strings.HasPrefix(cmd, "dotf lane exec --agent claude --prompt-file ") {
		t.Errorf("typed %q", cmd)
	}
	got, err := os.ReadFile(strings.TrimPrefix(cmd, "dotf lane exec --agent claude --prompt-file "))
	if err != nil || string(got) != "Work on #7 BUG-7: first thing on fix/first-thing; then 2 more." {
		t.Errorf("prompt file = %q, %v", got, err)
	}
}

// Twice is once: after the first run a workspace shows the branch, so the
// second finds it open and starts nothing.
func TestNext_RunTwiceStartsOneAgent(t *testing.T) {
	b := &fakeBackend{agent: "claude", existing: map[string]Worktree{"fix/first-thing": {Path: "/w/first", Branch: "fix/first-thing", Open: true}}}
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
	b := &fakeBackend{agent: "pi", existing: map[string]Worktree{"fix/first-thing": {Path: "/w/first", Branch: "fix/first-thing"}}}
	out, err := Handoff(opts(t), b, queue)
	if err != nil || !out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	want := []string{"find /repos/dotfiles fix/first-thing", "open /repos/dotfiles /w/first", "type w2:p1"}
	if !reflect.DeepEqual(b.calls, want) {
		t.Errorf("calls = %q, want %q", b.calls, want)
	}
}

// The successor is the same kind of agent as the finisher unless told
// otherwise, and an agent the table lacks is refused naming the ones it has.
func TestNext_AgentDefaultsToTheCallingPaneAndUnknownIsRefused(t *testing.T) {
	b := &fakeBackend{agent: "agy"}
	out, err := Handoff(opts(t), b, queue)
	if err != nil || out.Agent != "agy" || !strings.Contains(b.typed["w1:p1"], "--agent agy ") {
		t.Fatalf("calling pane's agent: %+v, %v", out, err)
	}
	o := opts(t)
	o.Agent = "pi"
	if out, err := Handoff(o, &fakeBackend{agent: "agy"}, queue); err != nil || out.Agent != "pi" {
		t.Errorf("explicit agent: %+v, %v", out, err)
	}
	for _, agent := range []string{"codex", ""} {
		bb := &fakeBackend{agent: agent}
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
	if len(out.Commands) != 4 || !strings.HasPrefix(out.Commands[1], "git -C /repos/dotfiles worktree add -b fix/first-thing "+filepath.Join("/repos", "worktrees", "dotfiles", "first-thing")+" origin/main") ||
		!strings.HasPrefix(out.Commands[3], "dotf lane exec --agent claude --prompt-file ") {
		t.Errorf("commands = %q", out.Commands)
	}
	if _, err := Handoff(opts(t), nil, queue); err == nil {
		t.Error("no backend and no --agent cannot guess an agent")
	}
}

// A dry run decides and reports; it writes no prompt and creates nothing.
func TestNext_DryRunChangesNothing(t *testing.T) {
	b := &fakeBackend{agent: "claude"}
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

// A prompt path a shell would read differently is refused rather than quoted.
func TestNext_RefusesAPromptPathThatNeedsQuoting(t *testing.T) {
	o := opts(t)
	o.PromptDir = "/home/a b/lane"
	if _, err := Handoff(o, &fakeBackend{agent: "claude"}, queue); err == nil || !strings.Contains(err.Error(), "prompt path") {
		t.Errorf("want a refusal, got %v", err)
	}
}

func TestNext_NothingFreeIsNotAnError(t *testing.T) {
	closed := mapSource{7: {Number: 7, State: "CLOSED"}, 8: {Number: 8, State: "CLOSED"}, 9: {Number: 9, State: "CLOSED"}}
	out, err := Handoff(opts(t), &fakeBackend{agent: "claude"}, closed)
	if err != nil || out.Started || out.Note == "" {
		t.Errorf("Handoff = %+v, %v", out, err)
	}
}

// The successor runs the dotf that decided, by absolute path, unless that path
// is not a word every shell reads alike; then it is `dotf` on the pane's PATH.
func TestNext_TheSuccessorRunsTheBinaryThatDecided(t *testing.T) {
	for dotf, want := range map[string]string{
		"/opt/dotf/bin/dotf":         "/opt/dotf/bin/dotf lane exec ",
		"/home/a b/dotf":             "dotf lane exec ",
		`C:\Users\RUNNER~1\dotf.exe`: `C:\Users\RUNNER~1\dotf.exe lane exec `,
		"~/dotf":                     "dotf lane exec ",
		"":                           "dotf lane exec ",
	} {
		b := &fakeBackend{agent: "claude"}
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
