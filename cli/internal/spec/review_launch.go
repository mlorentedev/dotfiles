package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultReviewerTimeout bounds a reviewer subprocess.
//
// It has to clear a real review — BUG-074's third round took roughly 25 minutes
// of wall clock, reading the spec, running the suite and mutation-testing the
// change — while staying short enough that a STUCK reviewer is noticed rather
// than waited on. An earlier draft used 90m, which is the wrong end of that
// trade: a hung run held for an hour and a half before anyone could tell it
// apart from a slow one, and slow-versus-hung is precisely the distinction a
// deadline exists to make.
//
// agy's own --print-timeout defaults to 5m, well under a real review, so this is
// always passed explicitly rather than inherited. Override per run with
// `dotf spec review --timeout` when a spec genuinely warrants longer.
const DefaultReviewerTimeout = 30 * time.Minute

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
// The two runners take their prompt DIFFERENTLY, and assuming otherwise is not a
// cosmetic error — it silently produces a reviewer that answers nothing:
//
//	pi   [options] [messages...]   -> prompt is a trailing POSITIONAL
//	agy  --print <prompt>          -> prompt is the VALUE of --print
//
// Because agy's --print consumes a value, writing `agy --print --model X …
// "<prompt>"` makes --print swallow "--model", leaving the model unset and the
// prompt orphaned. agy then starts a session and replies with a greeting: exit 0,
// plausible-looking output, no review. That is exactly how this was nearly
// shipped — a smoke test read "I am running on Gemini 3.1 Pro" as proof the arm
// worked, when it was proof the prompt had been discarded.
//
// Every flag here is verified against the installed binaries. An earlier draft
// also invented a --prompt-file that neither runner has.
func ReviewerCommand(e ReviewerEntry, prompt string, timeout time.Duration, repoRoot string) ([]string, error) {
	if timeout <= 0 {
		timeout = DefaultReviewerTimeout
	}
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
		//
		// --print goes LAST and carries the prompt as its value. Order matters:
		// any flag placed between --print and the prompt is consumed as the
		// prompt instead.
		//
		// --dangerously-skip-permissions is required, not optional. agy prompts
		// for approval on every tool call, and a detached reviewer has no human
		// to answer: each call is auto-DENIED. Observed exactly that — the run
		// read a few files, was refused `git rev-parse HEAD`, and gave up after
		// 14 seconds while reporting `status: SUCCESS` with an empty response.
		// A reviewer that cannot run the test suite cannot review, so the choice
		// is this flag or no fallback arm at all.
		//
		// The name is a fair warning and the posture is real: the reviewer gets
		// unattended shell in this repo. That is inherent to adversarial review
		// — running the suite and mutating the code IS the job, and the pi arm
		// has the same reach without asking. What bounds it is the deadline
		// above, the pool restricting WHICH models get here, and the fact that
		// the reviewer's only sanctioned output is review.md.
		//
		// --add-dir is what makes the review possible at all. Without it agy runs
		// its shell commands in its OWN install directory, not the repo: `pwd`
		// answers ~/.gemini/antigravity-cli and `git rev-parse HEAD` fails with
		// "not a git repository". A reviewer that cannot reach the tree cannot
		// run the suite or mutate anything, so it reviews by reading and grades
		// generously — which is exactly what the first Gemini review did.
		//
		// --sandbox bounds what the auto-approval above can touch. Verified not
		// to cost anything the review needs: under it, git, `go test` and file
		// writes all still work.
		agyArgs := []string{
			"agy",
			"--model", e.Model,
			"--output-format", "stream-json",
			"--print-timeout", timeout.String(),
			"--dangerously-skip-permissions",
			"--sandbox",
		}
		if strings.TrimSpace(repoRoot) != "" {
			agyArgs = append(agyArgs, "--add-dir", repoRoot)
		}
		// --print goes LAST and carries the prompt as its value.
		agyArgs = append(agyArgs, "--print", prompt)
		return append(base, agyArgs...), nil

	default:
		return nil, fmt.Errorf("pool entry %q names runner %q, which the launcher does not know how to invoke\n"+
			"known runners: pi, agy", e.ID, e.Runner)
	}
}

// TranscriptPath is where the launched reviewer's event stream lands for a spec.
func TranscriptPath(repoRoot, specID string) string {
	return filepath.Join(repoRoot, "specs", specID, TranscriptFile)
}

// ResolveReviewer picks which pool member runs.
//
// Default is the pool's FIRST entry — the launcher's primary. An explicit want
// selects any member by id, which is what makes the fallback reachable
// deliberately rather than only when the primary breaks: a fallback that is
// never chosen on purpose is never exercised, and an unexercised fallback is
// decoration.
func ResolveReviewer(entries []ReviewerEntry, want string) (ReviewerEntry, error) {
	if len(entries) == 0 {
		return ReviewerEntry{}, fmt.Errorf("no %s in this repo — the launcher has no model to run and will not guess one", ReviewerPoolFile)
	}
	want = strings.TrimSpace(want)
	if want == "" {
		return entries[0], nil
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.ID == want {
			return e, nil
		}
		ids = append(ids, e.ID)
	}
	return ReviewerEntry{}, fmt.Errorf("reviewer %q is not in %s\navailable: %s",
		want, ReviewerPoolFile, strings.Join(ids, ", "))
}

// ReviewPrompt is the brief handed to a launched reviewer.
//
// Deliberately THIN. The skill file is the contract, and repeating its content
// here would create a second source of truth that drifts. What the prompt adds
// is only what the skill cannot know: which spec, which repo, and the mechanical
// constraints that keep the review from invalidating itself.
//
// The canonical id is stated because `reviewer:` is self-reported and the gate
// matches it exactly — a reviewer that writes its own name a different way gets
// refused for a mismatch rather than for a policy breach, which wastes a whole
// review round on a string.
func ReviewPrompt(specID, repoRoot, reviewerID, skillPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Perform an adversarial review of the spec `%s`.\n\n", specID)
	fmt.Fprintf(&b, "Repo: %s\n\n", repoRoot)
	fmt.Fprintf(&b, "Read and follow the skill at %s exactly — its workflow, its severity x reality\n"+
		"classification, its test-traceability gate, its evaluator rubric, and its output format.\n"+
		"Read that file first; it defines the whole deliverable.\n\n", skillPath)
	b.WriteString("Mechanical constraints:\n")
	fmt.Fprintf(&b, "- Write your verdict to `specs/%s/%s`, overwriting what is there. The YAML\n"+
		"  frontmatter is part of the deliverable; `dotf spec archive` parses it and refuses a\n"+
		"  malformed one.\n", specID, ReviewFile)
	fmt.Fprintf(&b, "- Set `reviewer:` to exactly `%s`. That string is matched against this repo's\n"+
		"  reviewer pool, so any other spelling is refused even when the model is correct.\n", reviewerID)
	b.WriteString("- Set `reviewed_sha` to the output of `git rev-parse HEAD`.\n")
	b.WriteString("- Do NOT modify `proposal.md`, `tasks.md` or `features.json`. Those are the contract\n" +
		"  files a staleness check watches, so editing one would invalidate your own review the\n" +
		"  moment you wrote it. If you believe one needs changing, say so as a finding instead.\n")
	b.WriteString("- Do not commit, push, or comment on the PR. Writing the review is the deliverable.\n")
	b.WriteString("- Verify claims by running things — the build, the linters, the test suite, and\n" +
		"  mutation edits that you revert afterwards — not by reading assertions about them.\n" +
		"  Leave the working tree clean apart from your review.\n\n")
	b.WriteString("You are reviewing a change you did not write. Do not rubber-stamp it. Assume gaps\n" +
		"exist until you have argued against them with evidence.\n")
	return b.String()
}

// TmuxSession is the session name a launched review runs under.
//
// Deterministic and derived from the spec id so a human can attach without being
// told the name, and so a second launch for the same spec collides visibly
// instead of silently starting a rival reviewer.
func TmuxSession(specID string) string { return "review-" + specID }

// ReviewerSkillPath is where a given runner reads its deployed skills from.
//
// Each agent gets its own render of the same vault-sourced skill
// (harness/manifest.json's deploy targets), so the path differs per runner while
// the content does not. Pointing at the runner's OWN copy matters: it is the one
// that agent would load anyway, so the review is not reading a skill from a
// harness it does not use.
func ReviewerSkillPath(runner string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	dir := ".pi/agent/skills"
	if runner == "agy" {
		dir = ".gemini/skills"
	}
	return filepath.Join(home, dir, "adversarial-review", "SKILL.md")
}

// shellQuote renders one argument safe for a POSIX shell.
//
// Needed because both wrappers below hand a COMMAND STRING to something that
// re-parses it — tmux to its own shell, sh to itself — so an unquoted prompt
// containing backticks, quotes or newlines would be executed rather than passed.
// The reviewer prompt contains all three.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ShellJoin renders argv as a single POSIX-shell command string. Exported
// because the dry-run output has to print exactly what would run: joining the
// raw elements with spaces produces a line that is NOT runnable, since the last
// element is itself a whole pipeline.
func ShellJoin(argv []string) string {
	quoted := make([]string, 0, len(argv))
	for _, a := range argv {
		quoted = append(quoted, shellQuote(a))
	}
	return strings.Join(quoted, " ")
}

// TmuxWrap starts argv detached in a named session, teeing to transcript.
//
// Detached-with-a-name rather than attached: the caller gets its terminal back
// and the run is still watchable, which is the whole difference between a review
// that can be audited while it happens and one that reports only at the end.
// `-c dir` pins the working directory so the reviewer resolves the repo the same
// way the launcher did.
func TmuxWrap(session, dir string, argv []string, transcript string) []string {
	return []string{
		"tmux", "new-session", "-d",
		"-s", session,
		"-c", dir,
		ShellJoin(argv) + " | tee " + shellQuote(transcript),
	}
}
