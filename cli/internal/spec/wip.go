package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultWIPLimit is how many specs may be active before init refuses a new
// one (#770). Measured on 2026-09-30: dotfiles held 32 active specs, 5 of them
// finished and never archived and 11 stalled under 30%, some since June. The
// owner's rule is to finish before starting, and a rule no command asks for
// is not followed (lesson 301), so init asks.
const DefaultWIPLimit = 10

// wipLimitFile lets a repository declare its own limit, one integer.
const wipLimitFile = ".wip-limit"

// ActiveSpecs counts spec folders under specs/ that carry a proposal.md,
// leaving out specs/archive/.
func ActiveSpecs(repoRoot string) (int, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "specs"))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		if _, err := os.Stat(filepath.Join(repoRoot, "specs", e.Name(), "proposal.md")); err == nil {
			n++
		}
	}
	return n, nil
}

// WIPLimit is the repository's declared limit from specs/.wip-limit, or
// DefaultWIPLimit when the file is absent. A file that does not hold a
// positive integer is an error, never a silent default.
func WIPLimit(repoRoot string) (int, error) {
	p := filepath.Join(repoRoot, "specs", wipLimitFile)
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return DefaultWIPLimit, nil
	}
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s must hold one positive integer, got %q", p, strings.TrimSpace(string(b)))
	}
	return n, nil
}

// RecordWIPOverride writes the override into the new spec's frontmatter, so
// the reason travels with the spec and reaches its reviewer and its archive.
func RecordWIPOverride(repoRoot, id, reason string, active, limit int, date string) error {
	p := filepath.Join(repoRoot, "specs", id, "proposal.md")
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		return fmt.Errorf("%s has no frontmatter to record the WIP override in", p)
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return fmt.Errorf("%s has an unterminated frontmatter", p)
	}
	end += 4
	line := fmt.Sprintf("wip_override: %s", strconv.Quote(fmt.Sprintf("%s (%d active, limit %d, %s)", reason, active, limit, date)))
	s = s[:end] + "\n" + line + s[end:]
	return os.WriteFile(p, []byte(s), 0o644)
}
