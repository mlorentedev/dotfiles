package harness

import (
	"strings"
	"testing"
)

// TestPersonaPreambleCarriesTheRecord is AC6 at the composition layer.
//
// The measurement behind it: Request.Role is set at dispatch.go:136 and read
// only by dryrun.go:22. Subprocess sends --model and the task; Hive sends
// --model, --timeout and --prompt. So before this, `--role reviewer` dispatched
// a generic agent that was merely LOGGED as a reviewer, and the mandate,
// method and boundaries in the record never crossed the process boundary.
func TestPersonaPreambleCarriesTheRecord(t *testing.T) {
	reviewer := personaNamed(t, "reviewer")

	got, err := PersonaPreamble(reviewer, "check this diff")
	if err != nil {
		t.Fatalf("PersonaPreamble: %v", err)
	}

	// Sentences from the record's own prose. Asserted because the whole point
	// is that this specific text reaches the far side — a preamble carrying
	// only the persona's NAME would pass a weaker test and change nothing.
	//
	// The description is asserted too, and it comes from the PARSED struct
	// rather than the raw frontmatter: it is the persona's own one-line
	// statement of when it applies, which is worth sending, while `owner:` and
	// `generated_sha:` are not.
	for _, want := range []string{
		"reviewer",
		"Reviews and refutes",     // description, via Persona.Description
		"Try to refute the claim", // ## Mandate
		"a reviewer who fixes what they find has stopped being independent", // ## Boundaries
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preamble omits %q from the record", want)
		}
	}

	if !strings.Contains(got, "check this diff") {
		t.Error("preamble omits the task it was composed with")
	}

	// The task goes LAST. The record is context and the task is the
	// instruction; a model that reads the record after the task has had the
	// operating instruction arrive too late to shape how it reads the work.
	if strings.Index(got, "check this diff") < strings.Index(got, "Try to refute the claim") {
		t.Error("the task precedes the record; the persona must be established first")
	}

	// Frontmatter is machine metadata — id, owner, generated_sha, the skills
	// block the GATE reads. Sending it spends the far side's context on keys
	// that mean nothing to it and invites a model to answer ABOUT the record.
	for _, leaked := range []string{"generated_from:", "owner:", "type: agent"} {
		if strings.Contains(got, leaked) {
			t.Errorf("preamble leaks frontmatter key %q; the body is what instructs, "+
				"the frontmatter is what the loader reads", leaked)
		}
	}
}

// TestPersonaPreambleDistinguishesPersonas is the assertion that would fail on
// a preamble that was built and dropped, or on one that names a role without
// carrying it. Two personas must produce materially different instructions.
func TestPersonaPreambleDistinguishesPersonas(t *testing.T) {
	task := "do the thing"

	rev, err := PersonaPreamble(personaNamed(t, "reviewer"), task)
	if err != nil {
		t.Fatalf("reviewer: %v", err)
	}
	bld, err := PersonaPreamble(personaNamed(t, "builder"), task)
	if err != nil {
		t.Fatalf("builder: %v", err)
	}

	if rev == bld {
		t.Fatal("two personas composed identical instructions")
	}
	// Not merely different: each must carry ITS OWN boundary, which is the
	// sentence that makes the specialization real rather than nominal. The two
	// boundaries are near-opposites — the reviewer must not edit, the builder
	// must not review itself — so a preamble that mixed them would be worse
	// than one that carried neither.
	const reviewerBoundary = "you do not edit"
	const builderBoundary = "you do not redecide the architecture"

	if !strings.Contains(rev, reviewerBoundary) {
		t.Error("reviewer's preamble does not carry its refusal to edit")
	}
	if strings.Contains(rev, builderBoundary) {
		t.Error("reviewer's preamble carries builder's boundary")
	}
	if !strings.Contains(bld, builderBoundary) {
		t.Error("builder's preamble does not carry its scope boundary")
	}
	if strings.Contains(bld, reviewerBoundary) {
		t.Error("builder's preamble carries reviewer's boundary")
	}
}

// TestSplitRecordReturnsBothHalves pins the seam the loader and the preamble now
// share. Before HARNESS-120 the frontmatter had one reader and the body had
// none; giving the body a second parser would have been two ideas of where a
// record ends, disagreeing silently.
//
// The CRLF case is here because no shipped record exercises it on Linux and the
// Windows leg of CI compiles the same tree: a body split that kept the `---` of
// a CRLF fence would prepend three dashes to every dispatched instruction, on
// that platform only.
func TestSplitRecordReturnsBothHalves(t *testing.T) {
	for _, tc := range []struct {
		name, raw, wantFront, wantBody string
	}{
		{"lf", "---\nname: x\n---\n\n# Mandate\nDo the thing.\n", "\nname: x", "# Mandate\nDo the thing."},
		{"crlf", "---\r\nname: x\r\n---\r\n\r\n# Mandate\r\nDo it.\r\n", "\r\nname: x\r", "# Mandate\r\nDo it."},
		{"bom", "\ufeff---\nname: x\n---\n\nbody\n", "\nname: x", "body"},
		{"empty body", "---\nname: x\n---\n", "\nname: x", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			front, body, err := splitRecord([]byte(tc.raw))
			if err != nil {
				t.Fatalf("splitRecord: %v", err)
			}
			if string(front) != tc.wantFront {
				t.Errorf("front = %q, want %q", front, tc.wantFront)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}

	// A malformed record yields an error and never an empty body: an empty body
	// would dispatch the bare task, which is the generic agent this change
	// exists to replace, behind a successful exit.
	for _, raw := range []string{"# no fence\n", "---\nname: x\n"} {
		if _, _, err := splitRecord([]byte(raw)); err == nil {
			t.Errorf("split %q without an error", raw)
		}
	}
}

// TestPersonaPreambleRefusesAnEmptyRecord keeps the failure loud. A record that
// cannot be read is not one with nothing to say: dispatching the task alone
// would silently produce the generic agent this whole change exists to replace,
// and the caller would see a successful dispatch.
func TestPersonaPreambleRefusesAnEmptyRecord(t *testing.T) {
	if _, err := PersonaPreamble(&Persona{Name: "ghost", Path: "/nonexistent/AGENT.md"}, "t"); err == nil {
		t.Fatal("composed a preamble from a record that does not exist")
	}
	if _, err := PersonaPreamble(nil, "t"); err == nil {
		t.Fatal("composed a preamble from no persona at all")
	}
	if _, err := PersonaPreamble(personaNamed(t, "reviewer"), "  "); err == nil {
		t.Fatal("composed a preamble around an empty task")
	}
}
