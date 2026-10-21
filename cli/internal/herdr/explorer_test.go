package herdr

import (
	"errors"
	"strings"
	"testing"
)

// explorerFake answers `herdr`/`yazi` calls keyed on the FULL argv, which is what
// lets a test say "the invoking pane runs zsh, the other one runs yazi". The
// package's older fake keys on two words and cannot (#2815 review).
type explorerFake struct {
	calls   [][]string
	answers map[string]fakeAnswer
}

func (f *explorerFake) run(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	key := name + " " + strings.Join(args, " ")
	if a, ok := f.answers[key]; ok {
		return []byte(a.out), a.err
	}
	// Longest-prefix fallback, so a test names the command it cares about
	// ("herdr pane split") while another names one pane exactly.
	best := ""
	for k := range f.answers {
		if strings.HasPrefix(key, k) && len(k) > len(best) {
			best = k
		}
	}
	if best != "" {
		a := f.answers[best]
		return []byte(a.out), a.err
	}
	return nil, errors.New("unexpected call: " + key)
}

func (f *explorerFake) called(want string) bool {
	for _, c := range f.calls {
		if strings.Join(c, " ") == want {
			return true
		}
	}
	return false
}

func (f *explorerFake) joined() []string {
	out := make([]string, 0, len(f.calls))
	for _, c := range f.calls {
		out = append(out, strings.Join(c, " "))
	}
	return out
}

const yaziVersion = "Yazi\n    Version: 26.9.1 (8dd895c 2026-09-01)\n"

func explorerServer(extra map[string]fakeAnswer) *explorerFake {
	answers := map[string]fakeAnswer{
		"yazi --version":                {out: yaziVersion},
		"herdr pane split":              {out: `{"result":{"pane":{"pane_id":"w1:p2"}}}`},
		"herdr pane rename w1:p2 files": {out: `{"result":{"pane":{"pane_id":"w1:p2","label":"files"}}}`},
		"herdr pane run w1:p2 yazi":     {out: `{"result":{"type":"ok"}}`},
		"herdr pane close w1:p2":        {out: `{"result":{"type":"ok"}}`},
	}
	for k, v := range extra {
		answers[k] = v
	}
	return &explorerFake{answers: answers}
}

func TestExplorerContextRefusesOutsideHerdr(t *testing.T) {
	t.Setenv("HERDR_ACTIVE_PANE_ID", "")
	t.Setenv("HERDR_PANE_ID", "")
	if _, err := ExplorerContext(); err == nil || !strings.Contains(err.Error(), "HERDR_PANE_ID") {
		t.Fatalf("want a refusal naming the variable, got %v", err)
	}
}

func TestExplorerContextPrefersTheCustomCommandVariables(t *testing.T) {
	t.Setenv("HERDR_ACTIVE_PANE_ID", "w9:p3")
	t.Setenv("HERDR_PANE_ID", "w1:p1")
	t.Setenv("HERDR_ACTIVE_PANE_CWD", "/repo/wt")
	ctx, err := ExplorerContext()
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if ctx.Pane != "w9:p3" || ctx.Cwd != "/repo/wt" {
		t.Fatalf("want the custom-command variables, got %+v", ctx)
	}
}

func TestValidateExplorerRatio(t *testing.T) {
	for _, tc := range []struct {
		ratio float64
		ok    bool
	}{{0.3, true}, {0.05, true}, {0, false}, {1, false}, {1.5, false}, {-0.2, false}} {
		err := ValidateExplorerRatio(tc.ratio)
		if tc.ok && err != nil {
			t.Errorf("ratio %v: %v", tc.ratio, err)
		}
		if !tc.ok && (err == nil || !strings.Contains(err.Error(), "--ratio")) {
			t.Errorf("ratio %v must be refused, got %v", tc.ratio, err)
		}
	}
}

// The tab comes from herdr's answer for the pane, never from the launch-time
// HERDR_TAB_ID a moved pane still exports (#2815 review).
func TestToggleExplorerClosesTheExplorerInThePanesTab(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[` +
			`{"pane_id":"w1:p1","tab_id":"w1:t1"},` +
			`{"pane_id":"w1:p2","tab_id":"w1:t1","label":"files"},` +
			`{"pane_id":"w9:p1","tab_id":"w9:t3","label":"files"}]}}`},
	})
	line, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !srv.called("herdr pane close w1:p2") || !strings.Contains(line, "closed the explorer") {
		t.Fatalf("want this tab's explorer closed, got %q calls=%v", line, srv.joined())
	}
	if srv.called("herdr pane close w9:p1") {
		t.Fatalf("another tab's explorer was closed: %v", srv.joined())
	}
}

// A labelled explorer in another tab is not this tab's, so it is left alone and
// this one opens.
func TestToggleExplorerIgnoresAnotherTabsExplorer(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[` +
			`{"pane_id":"w1:p1","tab_id":"w1:t1"},` +
			`{"pane_id":"w9:p1","tab_id":"w9:t3","label":"files"}]}}`},
	})
	if _, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if srv.called("herdr pane close w9:p1") {
		t.Fatalf("another tab's explorer was closed: %v", srv.joined())
	}
	if !srv.called("herdr pane split --pane w1:p1 --direction right --cwd /repo/wt --ratio 0.30") {
		t.Fatalf("want a split in this tab, got %v", srv.joined())
	}
}

func TestToggleExplorerOpensWhenNoneIsOpen(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list":                      {out: `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"}]}}`},
		"herdr pane process-info --pane w1:p1": {out: `{"result":{"process_info":{"foreground_processes":[{"name":"zsh"}]}}}`},
	})
	line, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.45)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !srv.called("herdr pane split --pane w1:p1 --direction right --cwd /repo/wt --ratio 0.45") {
		t.Fatalf("want the split with the pane's cwd and ratio, got %v", srv.joined())
	}
	if !srv.called("herdr pane rename w1:p2 files") {
		t.Fatalf("want the created pane labelled, got %v", srv.joined())
	}
	if !srv.called("herdr pane run w1:p2 yazi") || !strings.Contains(line, "opened the explorer") {
		t.Fatalf("want yazi started, got %q calls=%v", line, srv.joined())
	}
}

// Closing by label means a labelled pane is ours even when nothing is running in
// it any more: that is the empty pane a later press would stack beside.
func TestToggleExplorerClosesALabelledPaneWithYaziGone(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"},{"pane_id":"w1:p2","tab_id":"w1:t1","label":"files"}]}}`},
	})
	if _, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !srv.called("herdr pane close w1:p2") {
		t.Fatalf("want the labelled pane closed, got %v", srv.joined())
	}
	if srv.called("herdr pane split --pane w1:p1 --direction right --cwd /repo/wt --ratio 0.30") {
		t.Fatalf("a labelled pane was cleared, so nothing may be split: %v", srv.joined())
	}
}

// A yazi the user opened themselves is not this explorer, and closing it — with
// whatever it was opened for — is worse than refusing to toggle.
func TestToggleExplorerRefusesToCloseAStrangersYazi(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list":                      {out: `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"},{"pane_id":"w1:p3","tab_id":"w1:t1"}]}}`},
		"herdr pane process-info --pane w1:p3": {out: `{"result":{"process_info":{"foreground_processes":[{"name":"yazi"}]}}}`},
	})
	_, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3)
	if err == nil || !strings.Contains(err.Error(), "w1:p3") {
		t.Fatalf("want the stranger's pane named, got %v", err)
	}
	for _, c := range srv.joined() {
		if strings.Contains(c, "pane close") || strings.Contains(c, "pane split") {
			t.Fatalf("nothing may be closed or split while a stranger's yazi is there: %v", srv.joined())
		}
	}
}

// The explorer is a pane people work in, so the key is pressed from inside it as
// often as from beside it. The label, not the pane's identity, says it is ours:
// pressing again must close it, not dock a second one (round 2 review).
func TestToggleExplorerClosesItselfWhenPressedFromInside(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[` +
			`{"pane_id":"w1:p1","tab_id":"w1:t1"},` +
			`{"pane_id":"w1:p2","tab_id":"w1:t1","label":"files"}]}}`},
		"herdr pane process-info --pane w1:p1": {out: `{"result":{"process_info":{"foreground_processes":[{"name":"zsh"}]}}}`},
	})
	line, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p2", Cwd: "/repo/wt"}, 0.3)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !srv.called("herdr pane close w1:p2") || !strings.Contains(line, "closed the explorer") {
		t.Fatalf("want the focused explorer closed, got %q calls=%v", line, srv.joined())
	}
	for _, c := range srv.joined() {
		if strings.HasPrefix(c, "herdr pane split") {
			t.Fatalf("a second explorer was docked: %v", srv.joined())
		}
	}
}

// The pane the command was typed into is never a candidate unless it IS the
// explorer: someone who runs yazi by hand where they press the key keeps it.
func TestToggleExplorerNeverTouchesAnUnlabelledInvokingPane(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"}]}}`},
	})
	if _, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if srv.called("herdr pane close w1:p1") {
		t.Fatalf("the invoking pane was closed: %v", srv.joined())
	}
	if !srv.called("herdr pane run w1:p2 yazi") {
		t.Fatalf("want the explorer docked beside it: %v", srv.joined())
	}
}

func TestToggleExplorerRefusesWithoutYazi(t *testing.T) {
	srv := &explorerFake{answers: map[string]fakeAnswer{
		"yazi --version": {out: "not found", err: errors.New("exec: yazi: executable file not found in $PATH")},
	}}
	_, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3)
	if err == nil || !strings.Contains(err.Error(), "dotf tools sync") {
		t.Fatalf("want the remedy named, got %v", err)
	}
}

// A split that cannot be labelled or filled is taken back, so the failure does
// not leave the empty pane the next press would stack beside.
func TestToggleExplorerTakesBackAPaneItCannotFill(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail string
		want string
	}{
		{"label fails", "herdr pane rename w1:p2 files", "label the explorer"},
		{"yazi will not start", "herdr pane run w1:p2 yazi", "start yazi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			answers := map[string]fakeAnswer{
				"herdr pane list": {out: `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"}]}}`},
			}
			answers[tc.fail] = fakeAnswer{err: errExit1, out: `{"error":{"code":"pane_gone","message":"pane is gone"}}`}
			srv := explorerServer(answers)
			_, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if !srv.called("herdr pane close w1:p2") {
				t.Fatalf("the split must be taken back: %v", srv.joined())
			}
		})
	}
}

func TestToggleExplorerRejectsASplitWithoutPaneID(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"}]}}`},
		"herdr pane split --pane w1:p1 --direction right --cwd /repo/wt --ratio 0.30": {out: `{"result":{"pane":{}}}`},
	})
	if _, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3); err == nil || !strings.Contains(err.Error(), "no pane id") {
		t.Fatalf("want the empty split reported, got %v", err)
	}
}

// A pane herdr does not list is a pane in another server, not something to guess
// about.
func TestToggleExplorerRefusesAPaneHerdrDoesNotList(t *testing.T) {
	srv := explorerServer(map[string]fakeAnswer{
		"herdr pane list": {out: `{"result":{"panes":[]}}`},
	})
	if _, err := ToggleExplorer(srv.run, PaneContext{Pane: "w1:p1", Cwd: "/repo/wt"}, 0.3); err == nil || !strings.Contains(err.Error(), "does not list pane") {
		t.Fatalf("want the unknown pane reported, got %v", err)
	}
}
