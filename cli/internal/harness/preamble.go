package harness

import (
	"fmt"
	"os"
	"strings"
)

// PersonaPreamble is the half of HARNESS-120 that makes the specialization real
// rather than nominal.
//
// Before it, `Request.Role` was threaded from --role through Options into
// Request at dispatch.go:136 and read by exactly one thing: the dry-run
// backend's echo string (dryrun.go:22). Subprocess sends `--model` and the task
// (backends.go:42); Hive sends `--model`, `--timeout` and `--prompt`
// (backends.go:70-78). Neither sent the persona. So a dispatch carrying
// `--role reviewer` ran a GENERIC agent that was merely logged as a reviewer,
// and the mandate, method and boundaries the six records carry stopped at the
// process boundary.
//
// A field that is plumbed but never read is invisible to the compiler and to
// every test that asserts on the record, because the record reports what was
// REQUESTED, not what was sent. That is why the test for this asserts against
// the request the backend RECEIVES.
//
// It is a preamble and not a protocol, deliberately. Both backends already
// accept task text, so the record rides that channel; adding a per-harness
// system-prompt argv would make `harnessFor` a director rather than a map,
// which ADR-035 §4 rejects.
func PersonaPreamble(p *Persona, task string) (string, error) {
	if p == nil {
		return "", fmt.Errorf("no persona to compose a preamble for")
	}
	if strings.TrimSpace(task) == "" {
		return "", fmt.Errorf("persona %q has no task to run: a dispatch with no task has nothing to do", p.Name)
	}

	body, err := recordBody(p)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	// The framing establishes the record as an instruction rather than as a
	// document to comment on. Without it a model handed a markdown file and a
	// question has been observed answering ABOUT the file.
	fmt.Fprintf(&b, "You are operating as the `%s` persona.\n", p.Name)
	if d := strings.TrimSpace(p.Description); d != "" {
		fmt.Fprintf(&b, "\n%s\n", d)
	}
	b.WriteString("\nEverything above the TASK delimiter is your operating instruction " +
		"for this dispatch. Follow it as written, including its boundaries — especially " +
		"the work it tells you NOT to do.\n\n")
	b.WriteString(body)
	// The task goes last. The record is context and the task is the
	// instruction; a task read before the record arrives too late to be shaped
	// by it.
	b.WriteString("\n\n" + taskDelimiter + "\n\n")
	b.WriteString(strings.TrimSpace(task))
	b.WriteString("\n")

	return b.String(), nil
}

// taskDelimiter separates the persona from the work. Deliberately unlikely to
// occur in either: a record is prose and a task is usually prose or a diff.
const taskDelimiter = "===== TASK ====="

// recordBody returns the persona record's markdown, without its frontmatter.
//
// The frontmatter is machine metadata — `id`, `owner`, `generated_sha`, and the
// `skills:` block the GATE reads. Sending it spends the far side's context on
// keys that mean nothing there and invites a model to answer about the record.
// The one frontmatter field worth sending, `description`, is already parsed
// into the struct and is added by the caller from there.
//
// An unreadable record is an error and never an empty body. Falling back to the
// bare task would dispatch exactly the generic agent this function exists to
// replace, and it would do so behind a successful exit — the silent-degrade
// direction this repository refuses everywhere else.
func recordBody(p *Persona) (string, error) {
	raw, err := os.ReadFile(p.Path) // #nosec G304 -- Path is set by LoadPersona from the manifest's record_dir
	if err != nil {
		return "", fmt.Errorf("read persona %q to send it with the task: %w", p.Name, err)
	}
	_, body, err := splitRecord(raw)
	if err != nil {
		return "", fmt.Errorf("persona %q: %w", p.Name, err)
	}
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf(
			"persona %q has an empty record body, so there is no instruction to send; "+
				"dispatching the task alone would silently produce a generic agent", p.Name)
	}
	return body, nil
}
