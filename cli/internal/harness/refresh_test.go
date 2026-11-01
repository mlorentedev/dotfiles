package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSectionOf_EndsAtTheSameLevelAndKeepsDeeperHeadings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "p.md")
	body := "# Top\n\n## 6. Attribution Policy\n- one\n### Detail\n- two\n\n\n## 7. Next\n- other\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := sectionOf(p, "6-attribution-policy")

	if err != nil {
		t.Fatal(err)
	}
	if want := "- one\n### Detail\n- two\n"; got != want {
		t.Errorf("section = %q, want %q (deeper heading kept, trailing blanks collapsed)", got, want)
	}
}

func TestSectionOf_AMissingOrEmptySectionIsAnError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(p, []byte("## Empty\n\n\n## Other\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"empty", "absent"} {
		if _, err := sectionOf(p, slug); err == nil || !strings.Contains(err.Error(), "not found (or empty)") {
			t.Errorf("slug %q: err = %v, want not found (or empty)", slug, err)
		}
	}
}

func TestHeadingSlug_MatchesTheAwkSlug(t *testing.T) {
	for in, want := range map[string]string{
		"6. Attribution Policy":      "6-attribution-policy",
		"12. IaC — Zero  Manual Ops": "12-iac-zero-manual-ops",
		"Definition of Done":         "definition-of-done",
		"Ünïcode Heading":            "ncode-heading",
		"trailing  ":                 "trailing-",
	} {
		if got := headingSlug(in); got != want {
			t.Errorf("headingSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStampProvenance_InsertsAfterTheOpeningFenceOnly(t *testing.T) {
	got := string(stampProvenance([]byte("---\nname: x\n---\nbody\n---\nend"), "00_meta/skills/x/SKILL.md", "abc"))
	want := "---\ngenerated: true\ngenerated_from: 00_meta/skills/x/SKILL.md\ngenerated_sha: abc\nname: x\n---\nbody\n---\nend\n"
	if got != want {
		t.Errorf("stamped =\n%s\nwant\n%s", got, want)
	}
}

// #2884 case 3: an END line with a tail passes the marker count but never
// closes the region. The shell rewrote the file without every line after it.
func TestRefreshedTarget_AnEndLineWithATailIsRefusedNotTruncated(t *testing.T) {
	repo := t.TempDir()
	src := "<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED --> tail\nthe user's line\n"
	if err := os.WriteFile(filepath.Join(repo, "T.md"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := refreshedTarget(repo, "T.md", []string{"r"}, map[string]string{"r": "- rule\n"}, "v")

	if err == nil {
		t.Fatalf("an END line with a tail was accepted; the render was:\n%s", got)
	}
	if !strings.Contains(err.Error(), "never closes") {
		t.Errorf("err = %v, want it to say the region never closes", err)
	}
}

func TestRefreshedTarget_ATargetOverItsLineCapIsRefused(t *testing.T) {
	repo := t.TempDir()
	file := "ai/claude/CLAUDE.md"
	if err := os.MkdirAll(filepath.Join(repo, "ai", "claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := "<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n"
	if err := os.WriteFile(filepath.Join(repo, file), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("- rule\n", 99)

	if _, err := refreshedTarget(repo, file, []string{"r"}, map[string]string{"r": long}, "v"); err == nil {
		t.Fatal("101 lines in a file capped at 100 was accepted")
	}
	if _, err := refreshedTarget(repo, file, []string{"r"}, map[string]string{"r": long[:len(long)-7]}, "v"); err != nil {
		t.Fatalf("100 lines in a file capped at 100: %v", err)
	}
}

// A vault checked out with CRLF (git's autocrlf on Windows) still yields LF
// records, the one line ending every committed record carries.
func TestSectionOf_ACRLFPatternYieldsAnLFRecord(t *testing.T) {
	p := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(p, []byte("# Rule\r\n- one\r\n# Next\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := sectionOf(p, "rule")

	if err != nil || got != "- one\n" {
		t.Errorf("section = %q, %v; want %q", got, err, "- one\n")
	}
}
