package lane

import (
	"errors"
	"strings"
	"testing"
)

func fakeRun(answers map[string]string, seen *[]string) Runner {
	return func(name string, args ...string) (string, error) {
		line := name + " " + strings.Join(args, " ")
		*seen = append(*seen, line)
		for prefix, out := range answers {
			if strings.HasPrefix(line, prefix) {
				return out, nil
			}
		}
		return "", errors.New("unexpected: " + line)
	}
}

// The shapes below are herdr 0.9.3's own answers, recorded on 2026-10-14.
const (
	herdrList = `{"id":"cli:worktree:list","result":{"type":"worktree_list","worktrees":[
	 {"branch":"main","path":"/p/dotfiles","open_workspace_id":"w9"},
	 {"branch":"fix/a","path":"/p/worktrees/dotfiles/a"},
	 {"branch":"fix/b","path":"/p/worktrees/dotfiles/b","open_workspace_id":"wB"}]}}`
	herdrCreated = `{"id":"cli:worktree:create","result":{"type":"worktree_created","root_pane":{"pane_id":"wG:p1"},
	 "worktree":{"branch":"fix/x","path":"/p/worktrees/dotfiles/x","open_workspace_id":"wG"}}}`
	herdrCurrent = `{"id":"cli:pane:current","result":{"pane":{"agent":"pi","pane_id":"wC:p1"},"type":"pane_current"}}`
)

func TestHerdr_FindsABranchAndKnowsWhetherAWorkspaceShowsIt(t *testing.T) {
	var seen []string
	h := Herdr{Run: fakeRun(map[string]string{"herdr worktree list": herdrList}, &seen)}
	for _, tc := range []struct {
		branch string
		found  bool
		open   bool
	}{{"fix/a", true, false}, {"fix/b", true, true}, {"fix/zzz", false, false}} {
		wt, ok, err := h.Find("/p/dotfiles", tc.branch)
		if err != nil || ok != tc.found || wt.Open != tc.open {
			t.Errorf("Find(%s) = %+v, %v, %v", tc.branch, wt, ok, err)
		}
	}
	if seen[0] != "herdr worktree list --cwd /p/dotfiles" {
		t.Errorf("asked %q", seen[0])
	}
}

func TestHerdr_CreateAnswersWithTheRootPaneAndCallingAgentIsRead(t *testing.T) {
	var seen []string
	h := Herdr{Run: fakeRun(map[string]string{"herdr worktree create": herdrCreated, "herdr pane current": herdrCurrent, "herdr pane run": "{}"}, &seen)}
	st, err := h.Create("/p/dotfiles", "fix/x", "origin/main", "#1 x")
	if err != nil || st.Pane != "wG:p1" || st.Worktree.Path != "/p/worktrees/dotfiles/x" {
		t.Fatalf("Create = %+v, %v", st, err)
	}
	if want := "herdr worktree create --cwd /p/dotfiles --branch fix/x --base origin/main --label #1 x --no-focus"; seen[0] != want {
		t.Errorf("asked %q, want %q", seen[0], want)
	}
	if a, err := h.CallingAgent(); err != nil || a != "pi" {
		t.Errorf("CallingAgent = %q, %v", a, err)
	}
	if err := h.Type("wG:p1", "dotf lane exec"); err != nil || seen[len(seen)-1] != "herdr pane run wG:p1 dotf lane exec" {
		t.Errorf("Type: %v, %q", err, seen[len(seen)-1])
	}
	h = Herdr{Run: fakeRun(map[string]string{"herdr worktree create": `{"id":"x","error":{"code":"linked_worktree_source"}}`}, &seen)}
	if _, err := h.Create("/p/w", "b", "o", "l"); err == nil {
		t.Error("an answer with no root pane is an error")
	}
}

// The shapes are herdr 0.9.3's own answers (recorded 2026-10-16): a pane is
// idle when its foreground process group is the shell and nothing else. herdr's
// "agent" field is not read, because it names claude and leaves pi and agy out.
func TestHerdr_AnIdlePaneIsOneWhoseForegroundIsOnlyItsShell(t *testing.T) {
	const list = `{"id":"cli:pane:list","result":{"panes":[{"pane_id":"wB:p1","workspace_id":"wB"},{"pane_id":"wB:p2"}]}}`
	shell := `{"id":"cli:pane:process_info","result":{"process_info":{"foreground_processes":[{"argv":["-zsh"],"pid":86131}],"pane_id":"wB:p1","shell_pid":86131}}}`
	busy := `{"id":"cli:pane:process_info","result":{"process_info":{"foreground_processes":[{"argv":["pi"],"pid":9},{"argv":["node"],"pid":10}],"pane_id":"wB:p1","shell_pid":86131}}}`
	for name, tc := range map[string]struct {
		info string
		idle bool
	}{"shell only": {shell, true}, "an agent in the foreground": {busy, false}} {
		t.Run(name, func(t *testing.T) {
			var seen []string
			h := Herdr{Run: fakeRun(map[string]string{"herdr pane list --workspace wB": list, "herdr pane process-info --pane wB:p1": tc.info}, &seen)}
			pane, idle, err := h.IdlePane(Worktree{Workspace: "wB"})
			if err != nil || pane != "wB:p1" || idle != tc.idle {
				t.Errorf("IdlePane = %q, %v, %v; want wB:p1, %v", pane, idle, err, tc.idle)
			}
		})
	}
	h := Herdr{Run: fakeRun(nil, new([]string))}
	if _, _, err := h.IdlePane(Worktree{}); err == nil {
		t.Error("a worktree no workspace shows has no pane to probe")
	}
	// fakeRun matches by prefix, so these two keys answer the real calls; the
	// error must come from the missing shell pid and not from an unanswered call.
	noPID := Herdr{Run: fakeRun(map[string]string{"herdr pane list": list, "herdr pane process-info": `{"result":{"process_info":{}}}`}, new([]string))}
	if _, _, err := noPID.IdlePane(Worktree{Workspace: "wB"}); err == nil || !strings.Contains(err.Error(), "no shell pid") {
		t.Errorf("an answer with no shell pid cannot be read as idle, got %v", err)
	}
}
