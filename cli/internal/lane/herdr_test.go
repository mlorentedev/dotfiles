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
