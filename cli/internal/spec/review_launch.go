package spec

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Deadlines for a reviewer subprocess. An adversarial review reads a spec, runs
// the test suite, and mutation-tests the change — BUG-074's third round took
// roughly 25 minutes of wall clock. agy's --print-timeout defaults to 5m, so the
// fallback would die on defaults; this is passed explicitly for that reason.
const reviewerTimeout = 90 * time.Minute

// TranscriptFile is where a launched review's machine-readable event stream is
// written, beside the review.md it produces.
//
// The verdict says WHAT a reviewer concluded; the transcript is the only record
// of HOW. That difference is what makes a review auditable by someone who was
// not watching — which, for a gate whose value is independence, is the point.
const TranscriptFile = "review-transcript.jsonl"

// ReviewerEntry is one pool member. `id` is the canonical string the reviewer
// records in review.md and the gate matches on; `provider`/`model` are the
// launcher's flags, deliberately explicit rather than parsed out of `id`.
type ReviewerEntry struct {
	ID       string `json:"id"`
	Runner   string `json:"runner"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Role     string `json:"role"`
}

// ReviewerCommand builds the argv that runs one pooled reviewer non-interactively.
//
// Every runner is invoked through `dotf secrets run --`, which is the only
// sanctioned way to hand a credential to a process (ADR-028): secrets reach the
// child and never the ambient shell. The interactive `pi`/`agy` shell functions
// wrap the binaries the same way, so this is the same path a human takes.
//
// The model is ALWAYS passed explicitly. Relying on a runner's configured
// default is how BUG-074's third round came to be pinned by coincidence: it
// resolved to nan/deepseek-v4-flash only because ~/.pi/agent/settings.json on
// that machine said so, while pi's own default provider is google. A pin that
// depends on unversioned per-machine state is not a pin.
// prompt is passed as a positional argument because that is what both runners
// accept — `pi [options] [messages...]` and agy's `--print` both read the prompt
// from argv, and neither has a --prompt-file flag. Verified against the
// installed binaries rather than assumed; an earlier draft of this function
// invented one.
func ReviewerCommand(e ReviewerEntry, prompt string) ([]string, error) {
	if strings.TrimSpace(e.Model) == "" {
		return nil, fmt.Errorf("pool entry %q has no model to pin — the launcher must not fall back to a runner default", e.ID)
	}

	base := []string{"dotf", "secrets", "run", "--"}

	switch e.Runner {
	case "pi":
		// --provider is mandatory here, not defensive: pi's own default is
		// `google`, so omitting it silently reviews on a different provider than
		// the pool entry names.
		if strings.TrimSpace(e.Provider) == "" {
			return nil, fmt.Errorf("pool entry %q uses runner pi, whose default provider is google — an explicit provider is required", e.ID)
		}
		return append(base,
			"pi", "--print",
			"--provider", e.Provider,
			"--model", e.Model,
			"--mode", "json",
			prompt,
		), nil

	case "agy":
		// agy has no --provider; the model id selects the family, and the
		// effort tier is encoded in the id itself (`…-pro-high` vs `…-pro-low`),
		// so --effort would be redundant. --print-timeout is passed because its
		// 5m default is far shorter than a real review.
		return append(base,
			"agy", "--print",
			"--model", e.Model,
			"--output-format", "stream-json",
			"--print-timeout", reviewerTimeout.String(),
			prompt,
		), nil

	default:
		return nil, fmt.Errorf("pool entry %q names runner %q, which the launcher does not know how to invoke\n"+
			"known runners: pi, agy", e.ID, e.Runner)
	}
}

// TranscriptPath is where the launched reviewer's event stream lands for a spec.
func TranscriptPath(repoRoot, specID string) string {
	return filepath.Join(repoRoot, "specs", specID, TranscriptFile)
}

// TmuxSession is the session name a launched review runs under.
//
// Deterministic and derived from the spec id so a human can attach without being
// told the name, and so a second launch for the same spec collides visibly
// instead of silently starting a rival reviewer.
func TmuxSession(specID string) string { return "review-" + specID }
