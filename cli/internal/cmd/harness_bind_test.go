package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// liveShapedSettings mirrors the DEPLOYED ~/.claude/settings.json measured on
// 2026-08-27, not a fixture built to pass.
//
// Two properties of the real file drive every assertion below and neither is
// obvious: our SessionStart hook carries NO marker (it was written by the
// positional jq writer, before markers existed), and SessionStart holds a SECOND
// group belonging to Orca. A fixture with one group, or with our entry already
// marked, would exercise neither the adoption path nor the preservation one.
const liveShapedSettings = `{
  "model": "opus",
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "DOTF mem session-start", "timeout": 30 } ] },
      { "matcher": "", "hooks": [ { "type": "command", "command": "orca-session-start", "timeout": 5 } ] }
    ],
    "SessionEnd": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "DOTF mem session-end", "timeout": 30 } ] }
    ],
    "PreToolUse": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "orca-pre-tool", "timeout": 5 } ] }
    ],
    "Stop": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "orca-stop", "timeout": 5 } ] }
    ]
  }
}`

// bindFixture writes the settings file in the shape THIS OS's setup script
// deployed, and returns the command prefix bind is expected to emit.
//
// The two must agree or the test measures the wrong thing: setup-linux.sh wrote
// a bare path and setup-windows.ps1 wrote a quoted one, and adoption is by exact
// command equality. Hardcoding the bare form here would pass on Linux and, on
// the Windows leg of CI, assert that a duplicate is correct.
func bindFixture(t *testing.T, settings string) (home, raw, want string) {
	t.Helper()
	home = t.TempDir()
	raw = filepath.Join(home, ".local", "bin", dotfBinaryName())
	want = hookBinaryToken(raw, runtime.GOOS)
	if settings == "" {
		return home, raw, want
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// The fixture is JSON, so the quoted Windows form has to survive encoding:
	// marshal the token and splice it in without its closing quote, which the
	// fixture's own `DOTF mem session-start"` already supplies.
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("encode dotf path: %v", err)
	}
	body := []byte(replaceAll(settings, `"DOTF`, string(encoded[:len(encoded)-1])))
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), body, 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return home, raw, want
}

func dotfBinaryName() string {
	if runtime.GOOS == "windows" {
		return "dotf.exe"
	}
	return "dotf"
}

func replaceAll(s, old, new string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func readSettings(t *testing.T, home string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("settings is not valid JSON after bind: %v", err)
	}
	return doc
}

func hookCommands(t *testing.T, doc map[string]any, event string) []string {
	t.Helper()
	hooks, _ := doc["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	var out []string
	for _, g := range groups {
		group, _ := g.(map[string]any)
		inner, _ := group["hooks"].([]any)
		for _, h := range inner {
			obj, _ := h.(map[string]any)
			cmd, _ := obj["command"].(string)
			out = append(out, cmd)
		}
	}
	return out
}

// TestBindAdoptsTheUnmarkedMemHookInsteadOfDuplicating is the defect this
// command was built around, from the adopting side.
//
// The deployed SessionStart hook runs our exact command and carries no marker.
// `isOurs` does not recognise it (its substring fallback only covers the gate),
// so without `sameCommand`'s exact-command adoption rule, bind would APPEND a
// second entry and every session would run `dotf mem session-start` twice.
func TestBindAdoptsTheUnmarkedMemHookInsteadOfDuplicating(t *testing.T) {
	home, raw, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw); err != nil {
		t.Fatalf("bind: %v", err)
	}

	got := hookCommands(t, readSettings(t, home), "SessionStart")
	mem := 0
	for _, c := range got {
		if c == dotf+" mem session-start" {
			mem++
		}
	}
	if mem != 1 {
		t.Errorf("SessionStart carries %d copies of the mem hook, want exactly 1:\n%v", mem, got)
	}
}

// TestBindNeverTouchesAForeignHook is the defect from the destroying side.
//
// `merge_claude_settings` assigns `.hooks.SessionStart = $tmpl.hooks.SessionStart`.
// Simulated against a copy of the live file, that took SessionStart from two
// groups to one and deleted Orca's. This asserts the replacement does not: every
// foreign command present before bind is present after, and the count of foreign
// entries is unchanged.
func TestBindNeverTouchesAForeignHook(t *testing.T) {
	home, raw, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	before := readSettings(t, home)
	beforeForeign := harness.ForeignHookCount(before)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw); err != nil {
		t.Fatalf("bind: %v", err)
	}
	after := readSettings(t, home)

	for _, ev := range []string{"SessionStart", "PreToolUse", "Stop"} {
		for _, want := range hookCommands(t, before, ev) {
			if want == dotf+" mem session-start" || want == dotf+" mem session-end" {
				continue // ours; adopted and re-emitted with a marker
			}
			if !contains(hookCommands(t, after, ev), want) {
				t.Errorf("bind deleted a foreign %s hook: %q\nafter: %v",
					ev, want, hookCommands(t, after, ev))
			}
		}
	}

	// Ours become marked, so the foreign count DROPS by exactly the two we
	// adopted -- never below that, which would mean a third party's entry was
	// claimed or removed.
	if got, want := harness.ForeignHookCount(after), beforeForeign-2; got != want {
		t.Errorf("foreign hook count = %d, want %d (was %d before)", got, want, beforeForeign)
	}
}

// TestBindAppendsTheGateBesideOrcasPreToolUseGroup pins AC5: a new hook on an
// event a third party already owns is a NEW group, not an edit of theirs.
func TestBindAppendsTheGateBesideOrcasPreToolUseGroup(t *testing.T) {
	home, raw, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw); err != nil {
		t.Fatalf("bind: %v", err)
	}

	got := hookCommands(t, readSettings(t, home), "PreToolUse")
	if !contains(got, "orca-pre-tool") {
		t.Errorf("Orca's PreToolUse hook is gone: %v", got)
	}
	if !contains(got, dotf+" harness gate --harness claude") {
		t.Errorf("the gate was not emitted on PreToolUse: %v", got)
	}
}

// HARNESS-149: the bind keeps the mode of a Claude settings file it rewrites. S4
// stopped forcing 0600 on every write, because every target is also written by
// another tool: Claude Code and Orca write ~/.claude/settings.json. Only agy's
// case was pinned, so a regression on this target would have passed.
func TestBindKeepsTheModeOfTheClaudeSettingsItRewrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	home, raw, _ := bindFixture(t, liveShapedSettings)
	path := filepath.Join(home, ".claude", "settings.json")
	if err := os.Chmod(path, 0o664); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", repoRootForTest(t), "--home", home, "--dotf-path", raw); err != nil {
		t.Fatalf("bind: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == string(before) {
		t.Fatal("the fixture needed no change, so the write path was never exercised")
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o664 {
		t.Errorf("the bind re-permissioned a settings file other tools also write: mode %v, want 0664", got)
	}
}

// TestBindIsIdempotent is the doctrine's changed=0 on re-run, asserted on bytes
// rather than on the command's own report.
func TestBindIsIdempotent(t *testing.T) {
	home, raw, _ := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)
	args := []string{"harness", "bind", "--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw}

	if _, _, err := captureRealStreams(t, args...); err != nil {
		t.Fatalf("first bind: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	stdout, _, err := captureRealStreams(t, args...)
	if err != nil {
		t.Fatalf("second bind: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Errorf("a re-run rewrote the file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !containsSub(stdout, "already current") {
		t.Errorf("a no-op run did not say so: %q", stdout)
	}
}

// TestBindSkipsWhatTheManifestSaysNotToEmit pins that emit:false is honoured and
// SAID OUT LOUD. A silent skip is how a gap stops being visible.
func TestBindSkipsWhatTheManifestSaysNotToEmit(t *testing.T) {
	home, raw, _ := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	stdout, _, err := captureRealStreams(t, "harness", "bind",
		"--repo-root", root, "--home", home, "--dotf-path", raw)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	for _, agent := range []string{"pi", "opencode"} {
		if !containsSub(stdout, "skip "+agent) {
			t.Errorf("%s is declared emit:false and bind did not report skipping it:\n%s", agent, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".pi", "agent", "extensions", "dotfiles-gate.ts")); err == nil {
		t.Error("bind wrote a ts-extension it is declared not to emit")
	}
}

// TestBindRefusesToOverwriteUnparseableSettings: a file someone is mid-edit is
// not a file to bootstrap over.
func TestBindRefusesToOverwriteUnparseableSettings(t *testing.T) {
	home, raw, _ := bindFixture(t, `{"hooks": {`)
	root := repoRootForTest(t)

	_, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw)
	if err == nil {
		t.Fatal("want an error on unparseable settings, got none")
	}
	if !containsSub(err.Error(), "refusing to overwrite") {
		t.Errorf("error does not say it refused: %v", err)
	}
	after, readErr := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if readErr != nil || string(after) != `{"hooks": {` {
		t.Errorf("the unparseable file was modified: %q (%v)", after, readErr)
	}
}

// TestBindBootstrapsAnAbsentFile: nothing to preserve, so writing is safe.
func TestBindBootstrapsAnAbsentFile(t *testing.T) {
	home, raw, dotf := bindFixture(t, "")
	root := repoRootForTest(t)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw); err != nil {
		t.Fatalf("bind: %v", err)
	}
	got := hookCommands(t, readSettings(t, home), "PreToolUse")
	if !contains(got, dotf+" harness gate --harness claude") {
		t.Errorf("bootstrap did not emit the gate: %v", got)
	}
}

// TestBindDryRunWritesNothing.
func TestBindDryRunWritesNothing(t *testing.T) {
	home, raw, _ := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)
	path := filepath.Join(home, ".claude", "settings.json")

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	stdout, _, err := captureRealStreams(t, "harness", "bind", "--dry-run",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", raw)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Error("--dry-run wrote to the settings file")
	}
	if !containsSub(stdout, "would update") {
		t.Errorf("--dry-run did not report the pending change: %q", stdout)
	}
}

// TestManifestGateEventMatchesDeclaredActionEvent pins the one duplication the
// manifest carries: `events.action` documents where the gate goes (measured per
// harness), and `emit_hooks` is what actually emits there. Two places saying the
// same thing drift; this makes the drift a failing test instead of a silent
// mismatch between the documentation and the emission.
func TestManifestGateEventMatchesDeclaredActionEvent(t *testing.T) {
	targets, err := harness.LoadBindTargets(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load bind targets: %v", err)
	}
	for _, tgt := range targets {
		for _, h := range tgt.EmitHooks {
			if h.ID != "gate" {
				continue
			}
			if want := tgt.Events["action"]; h.Event != want {
				t.Errorf("%s: emit_hooks gate is on %q, events.action declares %q",
					tgt.Agent, h.Event, want)
			}
		}
	}
}

// TestHookBinaryTokenMatchesWhatEachSetupScriptDeployed is the Windows
// duplicate-hook defect, caught statically rather than on the Windows box.
//
// Adoption of the pre-bind entry is by EXACT command equality, so the token this
// renders must equal, byte for byte, what the setup script of that OS wrote:
//
//	setup-linux.sh    $HOME/.local/bin/dotf mem session-start      (bare)
//	setup-windows.ps1 "…\.local\bin\dotf.exe" mem session-start    (quoted)
//
// A bare token on Windows matches neither, and bind would append a SECOND
// session-start hook on the first run there. Table-driven over goos because that
// is the only way the Windows leg is exercised from the machine that develops it.
func TestHookBinaryTokenMatchesWhatEachSetupScriptDeployed(t *testing.T) {
	for _, tc := range []struct {
		name, path, goos, want string
	}{
		{"windows is quoted, matching Merge-ClaudeSettings",
			`C:\Users\m\.local\bin\dotf.exe`, "windows", `"C:\Users\m\.local\bin\dotf.exe"`},
		{"linux is bare, matching merge_claude_settings",
			"/home/m/.local/bin/dotf", "linux", "/home/m/.local/bin/dotf"},
		{"a space forces quoting even where the old entry was bare",
			"/home/two words/.local/bin/dotf", "linux", `"/home/two words/.local/bin/dotf"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hookBinaryToken(tc.path, tc.goos); got != tc.want {
				t.Errorf("hookBinaryToken(%q, %q) = %q, want %q", tc.path, tc.goos, got, tc.want)
			}
		})
	}
}

// TestResolveDotfPathCarriesTheWindowsSuffix pins the other half of the same
// defect: the path itself. `dotf` and `dotf.exe` are different commands to the
// equality check, so a suffix-less resolve on Windows duplicates just as surely
// as missing quotes.
func TestResolveDotfPathCarriesTheWindowsSuffix(t *testing.T) {
	home := t.TempDir()
	want := "dotf"
	if runtime.GOOS == "windows" {
		want = "dotf.exe"
	}
	if got := filepath.Base(resolveDotfPath(home)); got != want {
		t.Errorf("resolveDotfPath resolves to %q, want basename %q on %s", got, want, runtime.GOOS)
	}
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func containsSub(s, sub string) bool { return indexOf(s, sub) >= 0 }

// liveShapedAgyHooks mirrors ~/.gemini/config/hooks.json as measured 2026-09-24:
// Orca's group and nothing else. It is the file agy reads its hooks from; agy's
// own log reports it on every start ("loaded 1 named hooks from 1 hooks.json
// file(s)").
const liveShapedAgyHooks = `{
  "orca-status": {
    "PreInvocation":  [{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}],
    "PostInvocation": [{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}],
    "Stop":           [{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}],
    "PreToolUse":     [{"matcher":"*","hooks":[{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}]}],
    "PostToolUse":    [{"matcher":"*","hooks":[{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}]}]
  }
}
`

// liveShapedGeminiSettings is what ~/.gemini/settings.json held before the agy
// binding moved: Orca's Gemini CLI hooks, plus the gate the first binding wrote
// there on the belief that agy reads this file. It does not; Gemini CLI does.
const liveShapedGeminiSettings = `{
  "hooks": {
    "BeforeTool": [
      {"hooks":[{"type":"command","command":"sh /o/gemini-hook.sh","timeout":10000}]},
      {"hooks":[{"_managed":"dotfiles-harness:gate","command":"DOTF harness gate --harness agy","timeout":5,"type":"command"}]}
    ],
    "AfterTool": [{"hooks":[{"type":"command","command":"sh /o/gemini-hook.sh","timeout":10000}]}]
  },
  "theme": "dark"
}
`

// agyBindFixture lays out a home the way the measured machine looked and puts a
// stand-in `agy` on PATH: the target is emitted only when the binary is
// installed, and a test that needs the real one would pass on one machine only.
func agyBindFixture(t *testing.T, hooksJSON, settings string) (home, dotf string) {
	t.Helper()
	home = t.TempDir()
	raw := filepath.Join(home, ".local", "bin", dotfBinaryName())
	dotf = hookBinaryToken(raw, runtime.GOOS)

	write := func(rel, body string, mode os.FileMode) {
		p := filepath.Join(home, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		// WriteFile honours the umask; the mode under test has to be exact.
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
	}
	if hooksJSON != "" {
		write(".gemini/config/hooks.json", hooksJSON, 0o664)
	}
	if settings != "" {
		encoded, err := json.Marshal(dotf)
		if err != nil {
			t.Fatal(err)
		}
		write(".gemini/settings.json", replaceAll(settings, `"DOTF`, string(encoded[:len(encoded)-1])), 0o600)
	}

	bin := t.TempDir()
	name := "agy"
	if runtime.GOOS == "windows" {
		name = "agy.exe"
	}
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return home, raw
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s is not valid JSON after bind: %v", path, err)
	}
	return doc
}

func bindAgy(t *testing.T, home, raw string, extra ...string) (string, error) {
	t.Helper()
	args := append([]string{"harness", "bind", "--harness", "agy", "--repo-root", repoRootForTest(t),
		"--home", home, "--dotf-path", raw}, extra...)
	stdout, _, err := captureRealStreams(t, args...)
	return stdout, err
}

// TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry is the defect the
// audit found, end to end on the shape of the measured files.
//
// The first binding wrote the agy gate into ~/.gemini/settings.json because that
// file declares BeforeTool. It is Gemini CLI's file. agy reads
// ~/.gemini/config/hooks.json, so the hook never ran under agy, and had it run,
// the payload would not have parsed.
func TestBindEmitsTheAgyGateIntoHooksJSONAndRetiresTheOldEntry(t *testing.T) {
	home, raw := agyBindFixture(t, liveShapedAgyHooks, liveShapedGeminiSettings)
	dotf := hookBinaryToken(raw, runtime.GOOS)
	hooksPath := filepath.Join(home, ".gemini", "config", "hooks.json")
	settingsPath := filepath.Join(home, ".gemini", "settings.json")
	orcaBefore, _ := json.Marshal(readJSONFile(t, hooksPath)["orca-status"])

	stdout, err := bindAgy(t, home, raw)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if !containsSub(stdout, "bind agy: .gemini/config/hooks.json") {
		t.Errorf("the emission was not reported: %q", stdout)
	}
	if !containsSub(stdout, "retire agy: .gemini/settings.json") {
		t.Errorf("the retirement was not reported: %q", stdout)
	}

	// hooks.json: ours beside Orca's, Orca's untouched.
	doc := readJSONFile(t, hooksPath)
	orcaAfter, _ := json.Marshal(doc["orca-status"])
	if string(orcaBefore) != string(orcaAfter) {
		t.Errorf("Orca's group was altered:\nbefore %s\nafter  %s", orcaBefore, orcaAfter)
	}
	ours, _ := doc["dotfiles-harness"].(map[string]any)
	groups, _ := ours["PreToolUse"].([]any)
	if len(groups) != 1 {
		t.Fatalf("the gate was not emitted under our name: %v", doc)
	}
	handler := groups[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)
	if handler["command"] != dotf+" harness gate --harness agy" {
		t.Errorf("command = %v, want the gate on the binary bind was told about", handler["command"])
	}
	if handler["timeout"] != float64(5) {
		t.Errorf("timeout = %v, want 5: an unbounded hook stalls every tool call", handler["timeout"])
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(hooksPath); err != nil || fi.Mode().Perm() != 0o664 {
			t.Errorf("a file shared with another tool keeps its mode, got %v (err %v)", fi.Mode().Perm(), err)
		}
	}

	// settings.json: the old gate is gone, everything else of Gemini CLI's stays.
	settings := readJSONFile(t, settingsPath)
	if cmds := hookCommands(t, settings, "BeforeTool"); len(cmds) != 1 || cmds[0] != "sh /o/gemini-hook.sh" {
		t.Errorf("BeforeTool should hold only Orca's hook after retirement, got %v", cmds)
	}
	if cmds := hookCommands(t, settings, "AfterTool"); len(cmds) != 1 {
		t.Errorf("an event we never touched was altered: %v", cmds)
	}
	if settings["theme"] != "dark" {
		t.Errorf("an unrelated setting was lost: %v", settings)
	}
}

func TestBindAgyIsIdempotent(t *testing.T) {
	home, raw := agyBindFixture(t, liveShapedAgyHooks, liveShapedGeminiSettings)
	if _, err := bindAgy(t, home, raw); err != nil {
		t.Fatalf("first bind: %v", err)
	}
	read := func() (string, string) {
		h, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "hooks.json"))
		s, _ := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))
		return string(h), string(s)
	}
	h1, s1 := read()

	stdout, err := bindAgy(t, home, raw)
	if err != nil {
		t.Fatalf("second bind: %v", err)
	}
	h2, s2 := read()
	if h1 != h2 || s1 != s2 {
		t.Errorf("a re-run rewrote a file")
	}
	if !containsSub(stdout, "already current") {
		t.Errorf("a no-op run did not say so: %q", stdout)
	}
	if containsSub(stdout, "retire") {
		t.Errorf("nothing is left to retire on a re-run: %q", stdout)
	}
}

func TestBindAgyBootstrapsAnAbsentHooksFileAndCreatesNothingToRetire(t *testing.T) {
	home, raw := agyBindFixture(t, "", "")
	if _, err := bindAgy(t, home, raw); err != nil {
		t.Fatalf("bind: %v", err)
	}
	doc := readJSONFile(t, filepath.Join(home, ".gemini", "config", "hooks.json"))
	if len(doc) != 1 || doc["dotfiles-harness"] == nil {
		t.Errorf("a new file should hold only our hook, got %v", doc)
	}
	// Retiring must never bring a file into being.
	if _, err := os.Stat(filepath.Join(home, ".gemini", "settings.json")); !os.IsNotExist(err) {
		t.Errorf("settings.json must not be created just to retire something from it (err %v)", err)
	}
}

// The new home is written before the old entry is retired. If it cannot be
// written, the old entry must still be there: a gate briefly doubled is the
// smaller harm than one that is briefly absent.
func TestBindAgyDoesNotRetireWhenTheNewHomeCannotBeWritten(t *testing.T) {
	home, raw := agyBindFixture(t, "{ this is not json", liveShapedGeminiSettings)
	settingsPath := filepath.Join(home, ".gemini", "settings.json")
	before, _ := os.ReadFile(settingsPath)

	if _, err := bindAgy(t, home, raw); err == nil {
		t.Fatal("an unparseable hooks.json must be refused, not overwritten")
	}
	after, _ := os.ReadFile(settingsPath)
	if string(before) != string(after) {
		t.Error("the old entry was retired although the new home was never written")
	}
	if b, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "hooks.json")); string(b) != "{ this is not json" {
		t.Errorf("a file someone is editing was overwritten: %q", b)
	}
}

func TestBindAgyDryRunWritesNothing(t *testing.T) {
	home, raw := agyBindFixture(t, liveShapedAgyHooks, liveShapedGeminiSettings)
	hooksBefore, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "hooks.json"))
	settingsBefore, _ := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))

	stdout, err := bindAgy(t, home, raw, "--dry-run")
	if err != nil {
		t.Fatalf("bind --dry-run: %v", err)
	}
	if !containsSub(stdout, "would update agy") || !containsSub(stdout, "would retire agy") {
		t.Errorf("a dry run must say what it would do: %q", stdout)
	}
	hooksAfter, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "hooks.json"))
	settingsAfter, _ := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))
	if string(hooksBefore) != string(hooksAfter) || string(settingsBefore) != string(settingsAfter) {
		t.Error("a dry run wrote a file")
	}
}

// An unknown format is a refusal. The previous default arm handed it to claude's
// merge, which is how a format this code did not know would have been written into
// a file of another shape.
func TestBindOneRefusesAFormatItDoesNotKnow(t *testing.T) {
	home := t.TempDir()
	target := harness.BindTarget{
		Agent: "future", File: ".future/hooks.json", Format: "hooks-yaml", Matcher: true,
		EmitHooks: []harness.EmitHook{{ID: "gate", Event: "PreToolUse", Command: "harness gate", Timeout: 5}},
	}
	if _, _, err := bindOne(target, home, "/opt/dotf", false); err == nil || !containsSub(err.Error(), "unsupported bind format") {
		t.Fatalf("want an unsupported-format refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".future", "hooks.json")); !os.IsNotExist(err) {
		t.Errorf("a refused format must not create a file (err %v)", err)
	}
}
