package nanprobe

import (
	"fmt"
	"io"
	"strings"
)

// Failing reports whether any result needs a person: a refusal (a file binds a
// model the key cannot call) or a model that stayed unavailable on both passes.
// Quota is excluded, because it resets on its own and an issue about it would
// be noise that teaches the reader to ignore the canary.
func Failing(results []Result) bool {
	for _, r := range results {
		if r.Class == Refused || r.Class == Unavailable {
			return true
		}
	}
	return false
}

// remedy is what the reader should do about one class.
func remedy(c Class) string {
	switch c {
	case Refused:
		return "this key cannot call it: replace it in every file listed"
	case Quota:
		return "nothing to fix: it passes, or the quota resets"
	case Unavailable:
		return "unanswered on two passes: check https://nan.builders/docs/models, and replace it if it persists"
	default:
		return ""
	}
}

// WriteMarkdown renders the results as the body of the canary's issue and of
// the job summary. Rows needing action come first.
func WriteMarkdown(w io.Writer, results []Result, b Bindings, notes []string) error {
	var sb strings.Builder
	sb.WriteString("### NaN model canary\n\n")
	sb.WriteString("| Model | Result | What to do | Bound in |\n|---|---|---|---|\n")
	for _, pass := range []bool{true, false} {
		for _, r := range results {
			needsAction := r.Class == Refused || r.Class == Unavailable
			if needsAction != pass {
				continue
			}
			result := string(r.Class)
			if r.Detail != "" {
				result += " (" + r.Detail + ")"
			}
			files := make([]string, 0, len(b.Files[r.Model]))
			for _, f := range b.Files[r.Model] {
				files = append(files, "`"+f+"`")
			}
			fmt.Fprintf(&sb, "| `%s` | %s | %s | %s |\n", r.Model, result, remedy(r.Class), strings.Join(files, ", "))
		}
	}
	if len(notes) > 0 {
		sb.WriteString("\nNot read:\n\n")
		for _, n := range notes {
			sb.WriteString("- " + n + "\n")
		}
	}
	_, err := io.WriteString(w, sb.String())
	return err
}
