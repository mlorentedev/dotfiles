package herdr

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// fakeServer answers each `herdr <args>` from a table keyed on the first two
// args, and records every call in order.
type fakeServer struct {
	calls   [][]string
	answers map[string]fakeAnswer
}

type fakeAnswer struct {
	out string
	err error
}

func (f *fakeServer) run(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	key := strings.Join(args[:min(2, len(args))], " ")
	if a, ok := f.answers[key]; ok {
		return []byte(a.out), a.err
	}
	return nil, errors.New("unexpected call: " + strings.Join(args, " "))
}

var errExit1 = errors.New("exit status 1")

// Measured on herdr 0.9.3 in a throwaway session (#2503).
const (
	serverUp    = `{"status":"running","running":true,"version":"0.9.3"}`
	serverDown  = `{"status":"not_running","running":false,"version":null}`
	nameFree    = `{"error":{"code":"agent_not_found","message":"agent target x not found"},"id":"cli:agent:get"}`
	opened      = `{"id":"cli:worktree:open","result":{"already_open":false,"root_pane":{"pane_id":"w2:p1"},"type":"worktree_opened"}}`
	started     = `{"id":"cli:agent:start","result":{"type":"agent_started"}}`
	prompted    = `{"id":"cli:agent:prompt","result":{"type":"agent_prompted"}}`
	notReady    = `{"error":{"code":"agent_not_ready","message":"agent blocked during startup"},"id":"cli:agent:start"}`
	blockedResp = `{"error":{"code":"agent_blocked","message":"agent is waiting at a dialog"},"id":"cli:agent:prompt"}`
)

func healthy() *fakeServer {
	return &fakeServer{answers: map[string]fakeAnswer{
		"status server": {out: serverUp},
		"agent get":     {out: nameFree, err: errExit1},
		"worktree open": {out: opened},
		"agent start":   {out: started},
		"agent prompt":  {out: prompted},
	}}
}

func TestPreflightNamesTheStateItRefuses(t *testing.T) {
	for _, tc := range []struct {
		name, harness, want string
		edit                func(f *fakeServer)
	}{
		{"unknown kind", "vim", "not an agent kind", nil},
		{"herdr absent", "claude", "herdr is not installed", func(f *fakeServer) {
			f.answers["status server"] = fakeAnswer{err: exec.ErrNotFound}
		}},
		{"server down", "claude", "no herdr server is running", func(f *fakeServer) {
			f.answers["status server"] = fakeAnswer{out: serverDown}
		}},
		{"agent get fails otherwise", "claude", "server went away", func(f *fakeServer) {
			f.answers["agent get"] = fakeAnswer{out: `{"error":{"code":"server_not_running","message":"server went away"}}`, err: errExit1}
		}},
		{"name taken", "claude", "already live", func(f *fakeServer) {
			f.answers["agent get"] = fakeAnswer{out: `{"id":"cli:agent:get","result":{"type":"agent_info"}}`}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := healthy()
			if tc.edit != nil {
				tc.edit(f)
			}
			err := Preflight(f.run, tc.harness, "feat-foo")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Preflight = %v, want an error naming %q", err, tc.want)
			}
		})
	}
	if err := Preflight(healthy().run, "claude", "feat-foo"); err != nil {
		t.Errorf("healthy herdr refused: %v", err)
	}
}

func TestLaunchOpensStartsAndPromptsWithoutWaiting(t *testing.T) {
	f := healthy()
	err := Launch(f.run, LaunchOptions{Repo: "/r", Worktree: "/w", Harness: "agy", Name: "feat-foo", Prompt: "do it"})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"herdr", "worktree", "open", "--cwd", "/r", "--path", "/w", "--no-focus"},
		{"herdr", "agent", "start", "feat-foo", "--kind", "agy", "--pane", "w2:p1"},
		{"herdr", "agent", "prompt", "feat-foo", "do it"},
	}
	if !slices.EqualFunc(f.calls, want, slices.Equal[[]string]) {
		t.Errorf("calls =\n%q\nwant\n%q", f.calls, want)
	}
	for _, c := range f.calls {
		if slices.Contains(c, "--wait") {
			t.Errorf("%q waits on the agent; add must return", c)
		}
	}
}

func TestLaunchWithoutAPromptOnlyStarts(t *testing.T) {
	f := healthy()
	if err := Launch(f.run, LaunchOptions{Repo: "/r", Worktree: "/w", Harness: "claude", Name: "n"}); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 {
		t.Errorf("calls = %q, want open and start only", f.calls)
	}
}

func TestLaunchFailsOnANotReadyOrBlockedAgent(t *testing.T) {
	for _, tc := range []struct {
		key, out, want string
		calls          int
	}{
		{"agent start", notReady, "agent_not_ready", 2},
		{"agent prompt", blockedResp, "agent_blocked", 3},
	} {
		t.Run(tc.want, func(t *testing.T) {
			f := healthy()
			f.answers[tc.key] = fakeAnswer{out: tc.out, err: errExit1}
			err := Launch(f.run, LaunchOptions{Repo: "/r", Worktree: "/w", Harness: "claude", Name: "n", Prompt: "p"})
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "not sent") {
				t.Fatalf("Launch = %v, want a failure naming %s and that the prompt was not sent", err, tc.want)
			}
			if len(f.calls) != tc.calls {
				t.Errorf("calls = %q, want %d", f.calls, tc.calls)
			}
		})
	}
}

// A failure whose stdout is not herdr's error JSON (a crash, a signal, a
// pipe error) still names why: the runner's error stands in for the message.
func TestAFailureWithoutHerdrsErrorJSONStillSaysWhy(t *testing.T) {
	killed := errors.New("signal: killed")
	for _, key := range []string{"agent get", "worktree open", "agent start", "agent prompt"} {
		t.Run(key, func(t *testing.T) {
			f := healthy()
			f.answers[key] = fakeAnswer{err: killed}
			err := Preflight(f.run, "claude", "n")
			if key != "agent get" {
				err = Launch(f.run, LaunchOptions{Repo: "/r", Worktree: "/w", Harness: "claude", Name: "n", Prompt: "p"})
			}
			if err == nil || !strings.Contains(err.Error(), "signal: killed") {
				t.Fatalf("got %v, want the runner's error in the message", err)
			}
		})
	}
}

func TestAgentName(t *testing.T) {
	for in, want := range map[string]string{
		"feat-foo":       "feat-foo",
		"fix-2527-login": "fix-2527-login",
		"feat-a-very-long-branch-slug-that-goes-past-32": "feat-a-very-long-branch-slug-tha",
		"feat-abcdefghijklmnopqrstuvwxyz-0":              "feat-abcdefghijklmnopqrstuvwxyz",
		"2527-hotfix":                                    "wt-2527-hotfix",
	} {
		if got := AgentName(in); got != want {
			t.Errorf("AgentName(%q) = %q, want %q", in, got, want)
		}
		if !agentNameShape.MatchString(AgentName(in)) {
			t.Errorf("AgentName(%q) = %q breaks herdr's name rule", in, AgentName(in))
		}
	}
}

// Every dotfiles harness must be a kind herdr can start, or --agent refuses
// one of the six for no reason the operator can see.
func TestEveryHarnessIsAKind(t *testing.T) {
	m, err := harness.LoadModelMap("../../..")
	if err != nil {
		t.Fatal(err)
	}
	hs := harness.Harnesses(m)
	if len(hs) == 0 {
		t.Fatal("model-map lists no harnesses")
	}
	for _, h := range hs {
		if !slices.Contains(Kinds, h) {
			t.Errorf("harness %q is not a herdr agent kind", h)
		}
	}
}
