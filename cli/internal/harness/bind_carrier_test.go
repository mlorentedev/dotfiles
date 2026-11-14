package harness

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// carrierGaps are the harnesses whose manifest entry declares a session-start
// presence that nothing deployed carries yet, so no session of theirs is told
// whether it stands in a lane (HARNESS-195 AC3). The test below asserts this
// table is exact: a gap not listed fails it, and so does a row whose harness
// now has a carrier, so the slice that adds one has to delete its row.
var carrierGaps = map[string]string{
	// Lands in feat/lane-line-agy-carrier, only after a release carries
	// `mem session-start --format agy-hook`: the manifest is mirrored on merge,
	// and a pinned binary that lacks the flag would fail the hook on every turn.
	"agy": "bind_named binds only a PreToolUse gate; no PreInvocation hook runs the session line",
}

// laneLineCommands are the commands that print the line, as the manifest
// spells them (a dotf subcommand suffix).
var laneLineCommands = []string{"mem session-start", "mem resume"}

func carrierDeployDsts(t *testing.T, root string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "ai", "deploy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Configs []struct {
			Dst string `json:"dst"`
		} `json:"configs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var dsts []string
	for _, c := range doc.Configs {
		dsts = append(dsts, filepath.ToSlash(c.Dst))
	}
	return dsts
}

// hasCarrier reports whether something dotf deploys carries target's presence
// event to the session: an emitted hook on that event that runs the line's
// command, or, for a target bind does not emit (a TypeScript extension), an
// ai/deploy.json entry that writes the file. The deploy dst spells the
// directory with a token, so it is matched on the file's last two path
// elements, extensions/dotfiles-gate.ts, rather than on the whole path.
func hasCarrier(target BindTarget, deployDsts []string) bool {
	if target.Emits() {
		for _, h := range target.EmitHooks {
			if h.Event != target.Events["presence"] {
				continue
			}
			for _, c := range laneLineCommands {
				if strings.HasPrefix(h.Command, c) {
					return true
				}
			}
		}
		return false
	}
	file := filepath.ToSlash(target.File)
	file = path.Join(path.Base(path.Dir(file)), path.Base(file))
	for _, dst := range deployDsts {
		if strings.HasSuffix(dst, "/"+file) {
			return true
		}
	}
	return false
}

func TestEveryPresenceTargetHasADeployedCarrier(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	targets, err := LoadBindTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	dsts := carrierDeployDsts(t, root)
	var missing []string
	declared := 0
	for _, target := range targets {
		if target.Events["presence"] == "" {
			continue
		}
		declared++
		if !hasCarrier(target, dsts) {
			missing = append(missing, target.Agent)
		}
	}
	if declared == 0 {
		t.Fatal("no manifest target declares a presence event: the test reads the wrong key")
	}
	var want []string
	for agent := range carrierGaps {
		want = append(want, agent)
	}
	sort.Strings(missing)
	sort.Strings(want)
	if strings.Join(missing, ",") != strings.Join(want, ",") {
		t.Errorf("harnesses with a presence event and no deployed carrier = %v, but carrierGaps lists %v:\n"+
			"a carrier that now exists means deleting its row, one that is missing means adding the carrier", missing, want)
	}
}

// hasCarrier is the oracle of the test above, so it gets its own cases: a
// guard that cannot say yes passes by listing every harness as a gap.
func TestHasCarrier(t *testing.T) {
	no := false
	hook := func(event, command string) []EmitHook { return []EmitHook{{ID: "x", Event: event, Command: command}} }
	cases := []struct {
		name   string
		target BindTarget
		dsts   []string
		want   bool
	}{
		{"an emitted session-start hook", BindTarget{Events: map[string]string{"presence": "SessionStart"}, EmitHooks: hook("SessionStart", "mem session-start")}, nil, true},
		{"an emitted hook on another event", BindTarget{Events: map[string]string{"presence": "SessionStart"}, EmitHooks: hook("PreToolUse", "mem session-start")}, nil, false},
		{"an emitted hook that runs something else", BindTarget{Events: map[string]string{"presence": "SessionStart"}, EmitHooks: hook("SessionStart", "harness gate")}, nil, false},
		{"an extension a deploy entry writes", BindTarget{Emit: &no, File: ".pi/agent/extensions/dotfiles-gate.ts"}, []string{"{HOME}/.pi/agent/extensions/dotfiles-gate.ts"}, true},
		{"an extension nothing writes", BindTarget{Emit: &no, File: ".pi/agent/extensions/dotfiles-gate.ts"}, []string{"{HOME}/.pi/agent/models.json"}, false},
	}
	for _, c := range cases {
		if got := hasCarrier(c.target, c.dsts); got != c.want {
			t.Errorf("%s: hasCarrier = %v, want %v", c.name, got, c.want)
		}
	}
}
