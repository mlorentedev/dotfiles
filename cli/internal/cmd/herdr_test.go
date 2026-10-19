package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// herdrPaneEnv clears every variable paneContext reads, so a test states the
// whole environment it depends on: this package's own tests run inside a herdr
// pane often enough that inheriting them would make the "not inside herdr" case
// pass for the wrong reason.
func herdrPaneEnv(t *testing.T, pane, tab, cwd string) {
	t.Helper()
	t.Setenv("HERDR_ACTIVE_PANE_ID", "")
	t.Setenv("HERDR_ACTIVE_TAB_ID", "")
	t.Setenv("HERDR_ACTIVE_PANE_CWD", "")
	t.Setenv("HERDR_PANE_ID", pane)
	t.Setenv("HERDR_TAB_ID", tab)
	t.Setenv("HERDR_ACTIVE_PANE_CWD", cwd)
}

type herdrCall struct {
	args []string
}

// stubHerdr replaces the herdr CLI with a recorded table: responses maps the
// first two arguments ("pane list") to the stdout that call returns.
func stubHerdr(t *testing.T, responses map[string]string) *[]herdrCall {
	t.Helper()
	var calls []herdrCall
	origRun, origLook := herdrRun, herdrLookPath
	t.Cleanup(func() { herdrRun, herdrLookPath = origRun, origLook })
	herdrLookPath = func(string) (string, error) { return "/usr/local/bin/yazi", nil }
	herdrRun = func(args ...string) ([]byte, error) {
		calls = append(calls, herdrCall{args: append([]string(nil), args...)})
		key := args[0]
		if len(args) > 1 {
			key += " " + args[1]
		}
		out, ok := responses[key]
		if !ok {
			t.Fatalf("unexpected herdr call %q", key)
		}
		return []byte(out), nil
	}
	return &calls
}

func runHerdrExplorer(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := newHerdrCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(append([]string{"explorer"}, args...))
	err := c.Execute()
	return out.String(), err
}

const herdrOnePaneList = `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"}]}}`

// Outside a herdr pane the command must say what it needs, not guess a pane: a
// split aimed at the wrong pane reshapes a tab the user is not looking at.
func TestHerdrExplorerRefusesOutsideHerdr(t *testing.T) {
	herdrPaneEnv(t, "", "", "")
	stubHerdr(t, map[string]string{})
	out, err := runHerdrExplorer(t)
	if err == nil || !strings.Contains(err.Error(), "HERDR_PANE_ID") {
		t.Fatalf("want a refusal naming HERDR_PANE_ID, got err=%v out=%q", err, out)
	}
}

// A machine that has not run `dotf tools sync` yet has no yazi. Opening an empty
// pane would look like the feature is broken; the remedy is one command away.
func TestHerdrExplorerRefusesWithoutYazi(t *testing.T) {
	herdrPaneEnv(t, "w1:p1", "w1:t1", "/repo")
	stubHerdr(t, map[string]string{})
	herdrLookPath = func(string) (string, error) { return "", errors.New("not found") }
	out, err := runHerdrExplorer(t)
	if err == nil || !strings.Contains(err.Error(), "dotf tools sync") {
		t.Fatalf("want a refusal naming the remedy, got err=%v out=%q", err, out)
	}
}

func TestHerdrExplorerOpensOnTheRightWhenNoneIsOpen(t *testing.T) {
	herdrPaneEnv(t, "w1:p1", "w1:t1", "/repo/wt")
	calls := stubHerdr(t, map[string]string{
		"pane list":         herdrOnePaneList,
		"pane process-info": `{"result":{"process_info":{"foreground_processes":[{"name":"zsh","argv0":"zsh"}]}}}`,
		"pane split":        `{"result":{"pane":{"pane_id":"w1:p2"}}}`,
		"pane run":          `{"result":{"type":"ok"}}`,
	})
	out, err := runHerdrExplorer(t)
	if err != nil {
		t.Fatalf("explorer: %v\n%s", err, out)
	}
	if len(*calls) != 4 {
		t.Fatalf("want list, process-info, split, run; got %d calls: %+v", len(*calls), *calls)
	}
	split := strings.Join((*calls)[2].args, " ")
	for _, want := range []string{"pane split", "--pane w1:p1", "--direction right", "--cwd /repo/wt", "--ratio 0.30"} {
		if !strings.Contains(split, want) {
			t.Fatalf("split argv %q missing %q", split, want)
		}
	}
	if got := strings.Join((*calls)[3].args, " "); got != "pane run w1:p2 yazi" {
		t.Fatalf("want yazi started in the new pane, got %q", got)
	}
}

// The toggle is the whole reason the command exists: a second press must not
// leave two explorers eating the tab.
func TestHerdrExplorerClosesTheOneAlreadyRunning(t *testing.T) {
	herdrPaneEnv(t, "w1:p1", "w1:t1", "/repo/wt")
	calls := stubHerdr(t, map[string]string{
		"pane list":         `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"},{"pane_id":"w1:p2","tab_id":"w1:t1"},{"pane_id":"w2:p1","tab_id":"w2:t1"}]}}`,
		"pane process-info": `{"result":{"process_info":{"foreground_processes":[{"name":"yazi","argv0":"yazi"}]}}}`,
		"pane close":        `{"result":{"type":"ok"}}`,
	})
	out, err := runHerdrExplorer(t)
	if err != nil {
		t.Fatalf("toggle closed: %v\n%s", err, out)
	}
	if len(*calls) != 3 {
		t.Fatalf("want list, one process-info for the tab's first pane, close; got %+v", *calls)
	}
	if got := strings.Join((*calls)[2].args, " "); got != "pane close w1:p1" {
		t.Fatalf("want the running explorer closed, got %q", got)
	}
	if !strings.Contains(out, "closed the explorer") {
		t.Fatalf("a toggle that closed something must say so: %q", out)
	}
}

// A pane running yazi in ANOTHER tab is not this tab's explorer, and closing it
// would close a pane the user opened somewhere else.
func TestHerdrExplorerIgnoresOtherTabsExplorer(t *testing.T) {
	herdrPaneEnv(t, "w1:p1", "w1:t1", "/repo/wt")
	calls := stubHerdr(t, map[string]string{
		"pane list":         `{"result":{"panes":[{"pane_id":"w2:p1","tab_id":"w2:t1"}]}}`,
		"pane split":        `{"result":{"pane":{"pane_id":"w1:p2"}}}`,
		"pane run":          `{"result":{"type":"ok"}}`,
		"pane process-info": `{"result":{"process_info":{"foreground_processes":[{"name":"yazi"}]}}}`,
	})
	if _, err := runHerdrExplorer(t); err != nil {
		t.Fatalf("explorer: %v", err)
	}
	for _, c := range *calls {
		if len(c.args) > 1 && c.args[1] == "process-info" && c.args[3] == "w2:p1" {
			t.Fatalf("the other tab's pane was inspected, so it could have been closed: %+v", c.args)
		}
	}
}

func TestHerdrExplorerRejectsASplitWithoutPaneID(t *testing.T) {
	herdrPaneEnv(t, "w1:p1", "w1:t1", "/repo/wt")
	stubHerdr(t, map[string]string{
		"pane list":         herdrOnePaneList,
		"pane process-info": `{"result":{"process_info":{"foreground_processes":[{"name":"zsh"}]}}}`,
		"pane split":        `{"result":{"pane":{}}}`,
	})
	if _, err := runHerdrExplorer(t); err == nil || !strings.Contains(err.Error(), "no pane id") {
		t.Fatalf("want the empty split reported, got %v", err)
	}
}

// The ratio is a flag, so the binding can widen the explorer without a code
// change; a hardcoded literal would argue with the flag.
func TestHerdrExplorerHonoursTheRatioFlag(t *testing.T) {
	herdrPaneEnv(t, "w1:p1", "w1:t1", "/repo/wt")
	calls := stubHerdr(t, map[string]string{
		"pane list":         herdrOnePaneList,
		"pane process-info": `{"result":{"process_info":{"foreground_processes":[{"name":"zsh"}]}}}`,
		"pane split":        `{"result":{"pane":{"pane_id":"w1:p2"}}}`,
		"pane run":          `{"result":{"type":"ok"}}`,
	})
	if _, err := runHerdrExplorer(t, "--ratio", "0.45"); err != nil {
		t.Fatalf("explorer: %v", err)
	}
	if got := strings.Join((*calls)[2].args, " "); !strings.HasSuffix(got, "--ratio 0.45") {
		t.Fatalf("want --ratio 0.45 in %q", got)
	}
}
