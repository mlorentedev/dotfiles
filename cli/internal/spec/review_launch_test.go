package spec

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	}, "review this spec", 0, "/repo")
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
		argv, err := ReviewerCommand(e, "p", 0, "/repo")
		if err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		if len(argv) < 4 || argv[0] != "dotf" || argv[1] != "secrets" || argv[2] != "run" || argv[3] != "--" {
			t.Errorf("%s must be invoked through `dotf secrets run --`, got %v", e.ID, argv[:min(4, len(argv))])
		}
	}
}

// When a pool member declares SecretID, the launcher scopes the injection with
// `--only <secret_id>` to shield the reviewer from unrelated broken secrets (BUG-089).
func TestReviewerCommandScopesSecretsWhenSecretIDDeclared(t *testing.T) {
	cases := []struct {
		entry    ReviewerEntry
		wantOnly string
	}{
		{
			entry:    ReviewerEntry{ID: "nan/deepseek-v4-flash", Runner: "pi", Provider: "nan", Model: "deepseek-v4-flash", SecretID: "NAN_API_KEY"},
			wantOnly: "NAN_API_KEY",
		},
		{
			entry:    ReviewerEntry{ID: "agy/gemini-3.1-pro-high", Runner: "agy", Model: "gemini-3.1-pro-high", SecretID: "GEMINI_API_KEY"},
			wantOnly: "GEMINI_API_KEY",
		},
	}
	for _, c := range cases {
		argv, err := ReviewerCommand(c.entry, "p", 0, "/repo")
		if err != nil {
			t.Fatalf("%s: %v", c.entry.ID, err)
		}
		if len(argv) < 6 || argv[0] != "dotf" || argv[1] != "secrets" || argv[2] != "run" || argv[3] != "--only" || argv[4] != c.wantOnly || argv[5] != "--" {
			t.Errorf("%s must use `dotf secrets run --only %s --`, got %v", c.entry.ID, c.wantOnly, argv[:min(6, len(argv))])
		}
	}
}

// agy's --print-timeout defaults to 5m. BUG-074's third round took roughly 25
// minutes, so the fallback dies on defaults — the concrete form of "configured
// is not exercised".
func TestReviewerCommandRaisesTheAgyPrintTimeout(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "agy/gemini-3.1-pro-high", Runner: "agy", Model: "gemini-3.1-pro-high",
	}, "p", 0, "/repo")
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
	pi, err := ReviewerCommand(ReviewerEntry{ID: "nan/x", Runner: "pi", Provider: "nan", Model: "x"}, "p", 0, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if argvValue(pi, "--mode") != "json" {
		t.Errorf("pi must emit json, got %q", argvValue(pi, "--mode"))
	}

	agy, err := ReviewerCommand(ReviewerEntry{ID: "agy/y", Runner: "agy", Model: "y"}, "p", 0, "/repo")
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
			if _, err := ReviewerCommand(e, "p", 0, "/repo"); err == nil {
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
	// Built with filepath.Join rather than a slash-joined literal: dotf is
	// cross-platform and Join uses \\ on Windows, so a hardcoded "specs/…"
	// asserts the separator instead of the structure and fails there.
	want := filepath.Join("specs", "AI-001-x", TranscriptFile)
	if !strings.HasSuffix(got, want) {
		t.Errorf("transcript must sit in the spec folder\n want suffix: %q\n         got: %q", want, got)
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

// The default is a random draw over the whole pool (HARNESS-093), so every
// member — the provider-diverse fallback included — gets exercised in the
// ordinary course of reviews rather than only during an outage; an explicit
// choice still reaches any member, which is how a run is reproduced.
func TestDrawReviewerReachesEveryMemberAndResolveHonoursAnExplicitChoice(t *testing.T) {
	entries := []ReviewerEntry{
		{ID: "nan/deepseek-v4-flash", Role: "primary"},
		{ID: "agy/gemini-3.1-pro-high", Role: "fallback"},
	}

	for i, want := range []string{"nan/deepseek-v4-flash", "agy/gemini-3.1-pro-high"} {
		got, err := DrawReviewer(entries, func(n int) int { return i })
		if err != nil || got.ID != want {
			t.Fatalf("draw %d must select %s, got %+v (%v)", i, want, got, err)
		}
	}
	if _, err := DrawReviewer(entries, func(n int) int { return n }); err == nil {
		t.Fatal("an out-of-range draw must be an error, not a panic or a silent first entry")
	}
	if _, err := ResolveReviewer(entries, ""); err == nil {
		t.Fatal("an empty name is not a request for the first entry any more")
	}
	got, err := ResolveReviewer(entries, "agy/gemini-3.1-pro-high")
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
	if _, err := DrawReviewer(nil, func(int) int { return 0 }); err == nil {
		t.Fatal("an absent pool must not resolve to a guessed default")
	}
}

// tmux re-parses its command through a shell, so the reviewer prompt — which
// carries quotes, backticks and newlines — must arrive quoted or it executes.
//
// The first version of this test could not fail: it asserted
// `contains("$(id)") && !contains("'")`, and ShellJoin quotes EVERY argument, so
// the second half was always false. It passed whether or not the dangerous token
// was quoted. Worse, the mutation battery reported it as guarding the behaviour,
// because deleting shellQuote broke an unrelated test (stubGh) and the
// whole-package run could not say which test killed the mutant.
//
// So this asserts the exact quoted form instead: the argument must appear
// verbatim inside single quotes, with its own quotes escaped the POSIX way.
func TestWrappersQuoteAnArgumentThatWouldOtherwiseExecute(t *testing.T) {
	nasty := "text with 'quotes' and `backticks` and $(id)"
	// Single quotes cannot nest, so a literal ' becomes '\'' — close, escape, reopen.
	want := `'text with '\''quotes'\'' and ` + "`backticks`" + ` and $(id)'`

	argv := TmuxWrap("s", "/repo", []string{"echo", nasty}, "/tmp/t.jsonl", nil)
	joined := argv[len(argv)-1]

	if !strings.Contains(joined, want) {
		t.Fatalf("argument must be single-quoted verbatim.\n want: %s\n  got: %s", want, joined)
	}
	if !strings.Contains(joined, "| tee '/tmp/t.jsonl'") {
		t.Errorf("the stream must be teed to a quoted transcript path, got: %s", joined)
	}
}

// Each runner reads its own deployed render of the same vault-sourced skill, so
// a review never loads a skill from a harness its runner does not use.
func TestReviewerSkillPathIsPerRunner(t *testing.T) {
	pi, agy := ReviewerSkillPath("pi"), ReviewerSkillPath("agy")
	// Same reason as the transcript test: assert the path structure, not the
	// separator the host happens to use.
	if !strings.Contains(pi, filepath.Join(".pi", "agent", "skills")) {
		t.Errorf("pi must read its own skills dir, got %q", pi)
	}
	if !strings.Contains(agy, filepath.Join(".gemini", "skills")) {
		t.Errorf("agy must read its own skills dir, got %q", agy)
	}
}

// The prompt must name the canonical id, because `reviewer:` is self-reported
// and the gate matches it exactly — a reviewer that spells its own name
// differently is refused for a string mismatch rather than a policy breach,
// burning a whole review round.
func TestReviewPromptNamesTheExactReviewerIDAndProtectsContractFiles(t *testing.T) {
	p := ReviewPrompt("AI-001-x", "/repo", "nan/deepseek-v4-flash", "pi", "/skill/SKILL.md", "basebasebase")
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

// The two runners take their prompt differently, and getting agy's wrong does
// not fail loudly — it produces a reviewer that greets you.
//
// agy's --print CONSUMES a value, so `agy --print --model X … "<prompt>"` makes
// --print swallow "--model": the model goes unset, the prompt is orphaned, and
// agy answers with a session greeting at exit 0. This was reproduced live: a
// launched review wrote "I am currently running on Gemini 3.1 Pro. How can I
// help you today?" into its transcript and nothing else.
func TestReviewerCommandGivesAgyThePromptAsThePrintValue(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "agy/gemini-3.1-pro-high", Runner: "agy", Model: "gemini-3.1-pro-high",
	}, "REVIEW-PROMPT", 0, "/repo")
	if err != nil {
		t.Fatalf("building the agy command: %v", err)
	}

	if got := argvValue(argv, "--print"); got != "REVIEW-PROMPT" {
		t.Fatalf("--print must carry the prompt as its value, got %q", got)
	}
	// Nothing may sit between --print and the prompt, or that flag is eaten as
	// the prompt instead.
	if i := argvIndex(argv, "--print"); i != len(argv)-2 {
		t.Fatalf("--print must be the last flag, at %d of %d: %v", i, len(argv), argv)
	}
	// The model must survive, which is exactly what the broken ordering lost.
	if got := argvValue(argv, "--model"); got != "gemini-3.1-pro-high" {
		t.Fatalf("--model must survive the ordering, got %q", got)
	}
}

// pi is the other convention, kept apart deliberately so a future edit that
// "unifies" them breaks a test instead of a reviewer.
func TestReviewerCommandGivesPiThePromptAsATrailingPositional(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "nan/x", Runner: "pi", Provider: "nan", Model: "x",
	}, "REVIEW-PROMPT", 0, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if argv[len(argv)-1] != "REVIEW-PROMPT" {
		t.Fatalf("pi takes the prompt positionally, got %q", argv[len(argv)-1])
	}
	if argvValue(argv, "--print") == "REVIEW-PROMPT" {
		t.Fatal("pi's --print is a boolean; giving it the prompt would consume the wrong token")
	}
}

// A deadline exists to tell "slow" apart from "hung". Too short and a real
// review is killed mid-flight; too long and a stuck run is indistinguishable
// from a working one for as long as it lasts.
//
// This bounds the default from both sides. The floor is the ~25 minutes
// BUG-074's third round actually took; the ceiling is the mistake this replaced,
// a 90m default under which a hung reviewer held for an hour and a half before
// anyone could tell.
func TestDefaultReviewerTimeoutIsBoundedFromBothSides(t *testing.T) {
	if DefaultReviewerTimeout < 26*time.Minute {
		t.Errorf("too short: a real review took ~25m, got %s", DefaultReviewerTimeout)
	}
	if DefaultReviewerTimeout > 45*time.Minute {
		t.Errorf("too long: a stuck reviewer must be noticed, not waited on, got %s", DefaultReviewerTimeout)
	}
}

// The caller can raise it for a spec that genuinely warrants longer, and a zero
// value means "unset" rather than "no deadline" — the one reading that would
// silently restore the unbounded behaviour.
func TestReviewerCommandHonoursAnExplicitTimeoutAndTreatsZeroAsUnset(t *testing.T) {
	e := ReviewerEntry{ID: "agy/y", Runner: "agy", Model: "y"}

	custom, err := ReviewerCommand(e, "p", 90*time.Minute, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if got := argvValue(custom, "--print-timeout"); got != "1h30m0s" {
		t.Errorf("an explicit timeout must reach the runner, got %q", got)
	}

	zero, err := ReviewerCommand(e, "p", 0, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if got := argvValue(zero, "--print-timeout"); got != DefaultReviewerTimeout.String() {
		t.Errorf("zero must mean the default, not an absent deadline, got %q", got)
	}
}

// agy prompts for approval on every tool call. A detached reviewer has no human
// to answer, so each call is auto-DENIED — and the failure does not look like
// one: the observed run read a few files, was refused `git rev-parse HEAD`, and
// stopped after 14 seconds while reporting `status: SUCCESS` with an empty
// response and no review.md.
//
// Running the suite and mutating the code IS the review, so a reviewer that
// cannot execute cannot review. pi needs no equivalent, which is precisely why a
// configured fallback is not a proven one.
func TestReviewerCommandLetsAgyActWithoutAHumanToApprove(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "agy/gemini-3.1-pro-high", Runner: "agy", Model: "gemini-3.1-pro-high",
	}, "p", 0, "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if argvIndex(argv, "--dangerously-skip-permissions") < 0 {
		t.Fatal("a detached agy reviewer cannot answer permission prompts, so every tool call is denied")
	}
}

// Without --add-dir, agy runs its shell commands in its OWN install directory:
// `pwd` answers ~/.gemini/antigravity-cli and `git rev-parse HEAD` fails with
// "not a git repository". A reviewer that cannot reach the tree cannot run the
// suite or mutate anything — it reviews by reading and grades generously, which
// is precisely what the first Gemini review did (all A's, one SPECULATIVE Minor,
// no independent verification).
//
// This is the difference between a fallback that produces an artifact and one
// that produces a review.
func TestReviewerCommandGivesAgyReachIntoTheRepo(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "agy/gemini-3.1-pro-high", Runner: "agy", Model: "gemini-3.1-pro-high",
	}, "p", 0, "/repo/root")
	if err != nil {
		t.Fatal(err)
	}
	if got := argvValue(argv, "--add-dir"); got != "/repo/root" {
		t.Fatalf("agy must be given the repo as a workspace or it cannot execute in it, got %q", got)
	}
	// The auto-approval above is bounded rather than bare. Verified not to cost
	// the review anything: git, `go test` and file writes all work under it.
	if argvIndex(argv, "--sandbox") < 0 {
		t.Error("unattended auto-approval should run inside the sandbox")
	}
	// --print must still be last, or it consumes a flag as the prompt.
	if i := argvIndex(argv, "--print"); i != len(argv)-2 {
		t.Fatalf("--print must remain the final flag, at %d of %d", i, len(argv))
	}
}

// pi needs neither flag: it runs in the caller's directory and asks for no
// approval. The arms differ, and pretending otherwise is what produced two
// silent failures in this one already.
func TestReviewerCommandDoesNotGivePiAgySpecificFlags(t *testing.T) {
	argv, err := ReviewerCommand(ReviewerEntry{
		ID: "nan/x", Runner: "pi", Provider: "nan", Model: "x",
	}, "p", 0, "/repo/root")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--add-dir", "--sandbox", "--dangerously-skip-permissions"} {
		if argvIndex(argv, flag) >= 0 {
			t.Errorf("pi must not receive agy's %s", flag)
		}
	}
}

// The reviewer must be TOLD who it is, not left to work it out from the
// doctrine it reads.
//
// Measured 2026-08-29 (#1383), AI-042 round 4: the draw was
// nan/deepseek-v4-flash, the transcript carries 68 assistant messages stamped
// with that provider and model, and the run ended with "I'm an Anthropic model,
// and this pool exists specifically to exclude Anthropic" and a refusal to write
// the verdict. It had read the pool's own BUG-074 note and reasoned its way to an
// identity the launcher had known since the draw. 40 minutes, 248 bash calls, no
// review.md.
func TestReviewPromptTellsTheReviewerWhoItIs(t *testing.T) {
	p := ReviewPrompt("AI-042-x", "/repo", "nan/deepseek-v4-flash", "pi", "/skill/SKILL.md", "basebasebase")
	for _, want := range []string{
		"You are running as `nan/deepseek-v4-flash`",
		"(runner `pi`)",
		"do not infer it from the doctrine",
		"you are the model that was drawn",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt must state the identity: missing %q", want)
		}
	}

	// Before the mechanical constraints, not buried among them: the model that
	// refused had read most of the brief before it decided who it was.
	if strings.Index(p, "You are running as") > strings.Index(p, "Mechanical constraints:") {
		t.Error("the identity must come before the mechanical constraints")
	}
}
