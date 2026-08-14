package spec

import (
	"strings"
	"testing"
)

// argvIndex returns the position of flag in argv, or -1.
func argvIndex(argv []string, flag string) int {
	for i, a := range argv {
		if a == flag {
			return i
		}
	}
	return -1
}

// argvValue returns the argument following flag, or "".
func argvValue(argv []string, flag string) string {
	if i := argvIndex(argv, flag); i >= 0 && i+1 < len(argv) {
		return argv[i+1]
	}
	return ""
}

// The pin is the whole point. BUG-074's third round ran on the intended model
// only because ~/.pi/agent/settings.json on one machine happened to default to
// it — pi's own documented default provider is `google`, and that file is
// unversioned per-machine state. A launcher that omits either flag reproduces
// that coincidence on the next machine.
func TestReviewerCommandPinsProviderAndModelExplicitly(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "nan/deepseek-v4-flash", Runner: "pi",
		Provider: "nan", Model: "deepseek-v4-flash",
	}, "review this spec")
	if err != nil {
		t.Fatalf("building the pi command: %v", err)
	}

	if got := argvValue(argv, "--provider"); got != "nan" {
		t.Errorf("--provider must be explicit and equal the pool entry, got %q", got)
	}
	if got := argvValue(argv, "--model"); got != "deepseek-v4-flash" {
		t.Errorf("--model must be explicit and equal the pool entry, got %q", got)
	}
	if argvIndex(argv, "--print") < 0 {
		t.Error("the reviewer must run non-interactively")
	}
	// The prompt is positional: neither runner has a --prompt-file flag, which
	// an earlier draft of the builder invented.
	if argv[len(argv)-1] != "review this spec" {
		t.Errorf("prompt must be the trailing positional arg, got %q", argv[len(argv)-1])
	}
}

// Secrets reach the child process and never the ambient shell (ADR-028). The
// interactive pi/agy shell functions wrap the binaries the same way, so the
// launcher takes the same path a human does rather than inventing a second one.
func TestReviewerCommandRunsThroughTheSecretsFacade(t *testing.T) {
	for _, e := range []ReviewerEntry{
		{ID: "nan/x", Runner: "pi", Provider: "nan", Model: "x"},
		{ID: "agy/y", Runner: "agy", Model: "y"},
	} {
		argv, err := ReviewerCommand(e, "p")
		if err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		if len(argv) < 4 || argv[0] != "dotf" || argv[1] != "secrets" || argv[2] != "run" || argv[3] != "--" {
			t.Errorf("%s must be invoked through `dotf secrets run --`, got %v", e.ID, argv[:min(4, len(argv))])
		}
	}
}

// agy's --print-timeout defaults to 5m. BUG-074's third round took roughly 25
// minutes, so the fallback dies on defaults — the concrete form of "configured
// is not exercised".
func TestReviewerCommandRaisesTheAgyPrintTimeout(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "agy/gemini-3.1-pro-high", Runner: "agy", Model: "gemini-3.1-pro-high",
	}, "p")
	if err != nil {
		t.Fatalf("building the agy command: %v", err)
	}
	got := argvValue(argv, "--print-timeout")
	if got == "" {
		t.Fatal("agy must get an explicit --print-timeout; its 5m default is shorter than a real review")
	}
	if got == "5m0s" || got == "5m" {
		t.Fatalf("--print-timeout must exceed agy's own default, got %q", got)
	}
}

// A machine-readable stream is the only record of HOW a reviewer reasoned. The
// verdict alone is not auditable by anyone who was not watching.
func TestReviewerCommandRequestsAMachineReadableStream(t *testing.T) {
	pi, err := ReviewerCommand(ReviewerEntry{ID: "nan/x", Runner: "pi", Provider: "nan", Model: "x"}, "p")
	if err != nil {
		t.Fatal(err)
	}
	if argvValue(pi, "--mode") != "json" {
		t.Errorf("pi must emit json, got %q", argvValue(pi, "--mode"))
	}

	agy, err := ReviewerCommand(ReviewerEntry{ID: "agy/y", Runner: "agy", Model: "y"}, "p")
	if err != nil {
		t.Fatal(err)
	}
	if argvValue(agy, "--output-format") != "stream-json" {
		t.Errorf("agy must emit stream-json, got %q", argvValue(agy, "--output-format"))
	}
}

// Refusing beats guessing: a pool entry missing the flags the launcher needs is
// a configuration error, and falling back to a runner default is precisely the
// failure this whole mechanism exists to prevent.
func TestReviewerCommandRefusesRatherThanFallBackToADefault(t *testing.T) {
	cases := map[string]ReviewerEntry{
		"no model at all":     {ID: "nan/x", Runner: "pi", Provider: "nan"},
		"pi without provider": {ID: "nan/x", Runner: "pi", Model: "x"},
		"unknown runner":      {ID: "z/x", Runner: "codex", Model: "x"},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ReviewerCommand(e, "p"); err == nil {
				t.Fatal("an unusable pool entry must error, not silently use a default")
			}
		})
	}
}

// The session name is derived, not chosen, so a human can attach without being
// told it — and so a second launch for the same spec collides visibly instead of
// quietly starting a rival reviewer.
func TestTmuxSessionNameIsDerivedFromTheSpec(t *testing.T) {
	if got := TmuxSession("HARNESS-071-reviewer-pool"); got != "review-HARNESS-071-reviewer-pool" {
		t.Errorf("unexpected session name %q", got)
	}
}

func TestTranscriptLandsBesideTheReview(t *testing.T) {
	got := TranscriptPath("/repo", "AI-001-x")
	if !strings.HasSuffix(got, "specs/AI-001-x/"+TranscriptFile) {
		t.Errorf("transcript must sit in the spec folder, got %q", got)
	}
}

// The launcher's primary is the pool's first entry — the same ordering the gate
// treats as a flat allow-list, so one file answers both questions.
func TestPoolEntriesLoadInFileOrderWithLauncherFields(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, `{"pool":[
	  {"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"},
	  {"id":"agy/gemini-3.1-pro-high","runner":"agy","model":"gemini-3.1-pro-high","role":"fallback"}
	]}`)

	entries, err := LoadReviewerPoolEntries(root)
	if err != nil {
		t.Fatalf("loading the pool: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "nan/deepseek-v4-flash" || entries[0].Runner != "pi" || entries[0].Provider != "nan" {
		t.Errorf("first entry lost its launcher fields: %+v", entries[0])
	}
	if entries[1].Model != "gemini-3.1-pro-high" {
		t.Errorf("second entry lost its model: %+v", entries[1])
	}
}

// No pool means no opinion — for the launcher as much as for the gate.
func TestPoolEntriesAreNilWhenNoPoolExists(t *testing.T) {
	entries, err := LoadReviewerPoolEntries(t.TempDir())
	if err != nil {
		t.Fatalf("an absent pool is not an error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %+v", entries)
	}
}

// The default is the pool's first entry, and an explicit choice reaches any
// member — which is what makes the fallback exercisable on purpose rather than
// only during an outage. A fallback never chosen deliberately is never tested.
func TestResolveReviewerDefaultsToPrimaryAndHonoursAnExplicitChoice(t *testing.T) {
	entries := []ReviewerEntry{
		{ID: "nan/deepseek-v4-flash", Role: "primary"},
		{ID: "agy/gemini-3.1-pro-high", Role: "fallback"},
	}

	got, err := ResolveReviewer(entries, "")
	if err != nil || got.ID != "nan/deepseek-v4-flash" {
		t.Fatalf("empty want must select the first entry, got %+v (%v)", got, err)
	}
	got, err = ResolveReviewer(entries, "agy/gemini-3.1-pro-high")
	if err != nil || got.Role != "fallback" {
		t.Fatalf("explicit want must select that entry, got %+v (%v)", got, err)
	}
}

// Refusing at the launcher as well as at the gate is defence in depth: the gate
// catches a review that already ran on the wrong model, the launcher stops it
// from running at all — cheaper, and it names what is available.
func TestResolveReviewerRefusesAModelOutsideThePool(t *testing.T) {
	_, err := ResolveReviewer([]ReviewerEntry{{ID: "nan/x"}}, "claude-opus-5")
	if err == nil {
		t.Fatal("a model outside the pool must not be launchable")
	}
	if !strings.Contains(err.Error(), "nan/x") {
		t.Errorf("the error must list what IS available, got: %v", err)
	}
}

// With no pool there is nothing to run, and guessing a model would defeat the
// whole mechanism.
func TestResolveReviewerRefusesWhenThereIsNoPool(t *testing.T) {
	if _, err := ResolveReviewer(nil, ""); err == nil {
		t.Fatal("an absent pool must not resolve to a guessed default")
	}
}

// Both wrappers hand a command STRING to something that re-parses it — tmux to
// its shell, sh to itself. The reviewer prompt contains quotes, backticks and
// newlines, so an unquoted argument would be executed rather than passed.
func TestWrappersQuoteAnArgumentThatWouldOtherwiseExecute(t *testing.T) {
	nasty := "text with 'quotes' and `backticks` and $(id)"
	for name, argv := range map[string][]string{
		"tee":  TeeWrap([]string{"echo", nasty}, "/tmp/t.jsonl"),
		"tmux": TmuxWrap("s", "/repo", []string{"echo", nasty}, "/tmp/t.jsonl"),
	} {
		t.Run(name, func(t *testing.T) {
			joined := argv[len(argv)-1]
			if strings.Contains(joined, "$(id)") && !strings.Contains(joined, `'`) {
				t.Fatalf("command substitution must be quoted, got: %s", joined)
			}
			if !strings.Contains(joined, "| tee ") {
				t.Errorf("the stream must be teed to a transcript, got: %s", joined)
			}
		})
	}
}

// Each runner reads its own deployed render of the same vault-sourced skill, so
// a review never loads a skill from a harness its runner does not use.
func TestReviewerSkillPathIsPerRunner(t *testing.T) {
	pi, agy := ReviewerSkillPath("pi"), ReviewerSkillPath("agy")
	if !strings.Contains(pi, ".pi/agent/skills") {
		t.Errorf("pi must read its own skills dir, got %q", pi)
	}
	if !strings.Contains(agy, ".gemini/skills") {
		t.Errorf("agy must read its own skills dir, got %q", agy)
	}
}

// The prompt must name the canonical id, because `reviewer:` is self-reported
// and the gate matches it exactly — a reviewer that spells its own name
// differently is refused for a string mismatch rather than a policy breach,
// burning a whole review round.
func TestReviewPromptNamesTheExactReviewerIDAndProtectsContractFiles(t *testing.T) {
	p := ReviewPrompt("AI-001-x", "/repo", "nan/deepseek-v4-flash", "/skill/SKILL.md")
	for _, want := range []string{
		"AI-001-x", "/repo", "/skill/SKILL.md",
		"`nan/deepseek-v4-flash`",
		"proposal.md", "tasks.md", "features.json",
		"Do not rubber-stamp",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt must mention %q", want)
		}
	}
}
