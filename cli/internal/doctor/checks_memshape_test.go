package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// memShapeEnv builds a HOME with a .claude/projects tree and writes one
// MEMORY.md per entry of files (project name → content).
func memShapeEnv(t *testing.T, files map[string]string) (*System, string) {
	t.Helper()
	home := t.TempDir()
	for proj, body := range files {
		dir := filepath.Join(home, ".claude", "projects", proj, "memory")
		mkdirAll(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", proj, err)
		}
	}
	return newSys(map[string]string{"HOME": home}, nil, nil), home
}

const wrappedFixture = "---\nid: \"x-memory\"\ntype: memory\ncontent: |\n" +
	"    # x - Project Memory\n" +
	"      \n" +
	"      ## Session Handoff\n" +
	"      > Updated: 2026-08-09  \n"

const plainFixture = "# x - Project Memory\n\n## Session Handoff\n> Updated: 2026-08-09  \n"

func TestCheckMemoryShape(t *testing.T) {
	t.Run("no projects directory → SKIP", func(t *testing.T) {
		sys := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
		var buf bytes.Buffer
		rep := capture(&buf)
		checkMemoryShape(sys, rep, false)
		if rep.Failures() != 0 || !strings.Contains(buf.String(), "nothing to inspect") {
			t.Errorf("expected SKIP, got\n%s", buf.String())
		}
	})

	t.Run("all plain markdown → PASS", func(t *testing.T) {
		sys, _ := memShapeEnv(t, map[string]string{"-a": plainFixture, "-b": plainFixture})
		var buf bytes.Buffer
		rep := capture(&buf)
		checkMemoryShape(sys, rep, false)
		if rep.Failures() != 0 || !strings.Contains(buf.String(), "no YAML-wrapped") {
			t.Errorf("expected PASS, got\n%s", buf.String())
		}
	})

	t.Run("wrapped without --fix → FAIL, names the files, writes nothing", func(t *testing.T) {
		sys, home := memShapeEnv(t, map[string]string{"-a": wrappedFixture, "-b": plainFixture})
		path := filepath.Join(home, ".claude", "projects", "-a", "memory", "MEMORY.md")
		before, _ := os.ReadFile(path)

		var buf bytes.Buffer
		rep := capture(&buf)
		checkMemoryShape(sys, rep, false)

		if rep.Failures() != 1 {
			t.Errorf("expected 1 failure, got %d\n%s", rep.Failures(), buf.String())
		}
		if !strings.Contains(buf.String(), "dotf doctor --fix") {
			t.Errorf("expected a --fix hint, got\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "wrapped: "+path) {
			t.Errorf("expected the offending path to be named, got\n%s", buf.String())
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			t.Error("verify-only run modified the file")
		}
	})

	t.Run("--fix migrates and leaves plain files alone", func(t *testing.T) {
		sys, home := memShapeEnv(t, map[string]string{"-a": wrappedFixture, "-b": plainFixture})
		var buf bytes.Buffer
		rep := capture(&buf)
		checkMemoryShape(sys, rep, true)

		got, err := os.ReadFile(filepath.Join(home, ".claude", "projects", "-a", "memory", "MEMORY.md"))
		if err != nil {
			t.Fatal(err)
		}
		want := "---\nid: \"x-memory\"\ntype: memory\n---\n\n" + plainFixture
		if string(got) != want {
			t.Errorf("migrated content:\n%q\nwant:\n%q", got, want)
		}

		untouched, _ := os.ReadFile(filepath.Join(home, ".claude", "projects", "-b", "memory", "MEMORY.md"))
		if string(untouched) != plainFixture {
			t.Error("--fix modified an already-plain file")
		}
	})

	t.Run("the markdown hard break survives the migration", func(t *testing.T) {
		// The reason no YAML emitter could do this job: two trailing spaces are
		// the hard break the handoff convention needs, and a block scalar cannot
		// round-trip them.
		sys, home := memShapeEnv(t, map[string]string{"-a": wrappedFixture})
		var buf bytes.Buffer
		checkMemoryShape(sys, capture(&buf), true)

		got, _ := os.ReadFile(filepath.Join(home, ".claude", "projects", "-a", "memory", "MEMORY.md"))
		if !strings.Contains(string(got), "> Updated: 2026-08-09  \n") {
			t.Errorf("hard break lost:\n%q", got)
		}
	})

	t.Run("--fix is idempotent", func(t *testing.T) {
		sys, home := memShapeEnv(t, map[string]string{"-a": wrappedFixture})
		path := filepath.Join(home, ".claude", "projects", "-a", "memory", "MEMORY.md")

		var first bytes.Buffer
		checkMemoryShape(sys, capture(&first), true)
		once, _ := os.ReadFile(path)

		var second bytes.Buffer
		rep := capture(&second)
		checkMemoryShape(sys, rep, true)
		twice, _ := os.ReadFile(path)

		if !bytes.Equal(once, twice) {
			t.Error("second --fix run changed the file again")
		}
		if !strings.Contains(second.String(), "no YAML-wrapped") {
			t.Errorf("second run should report clean, got\n%s", second.String())
		}
	})

	t.Run("an unrecognised shape is warned about, never rewritten", func(t *testing.T) {
		// A block opener followed by a column-0 key: the block ends early and
		// this is a shape we have never seen. Refusing beats guessing on a file
		// holding real session continuity.
		odd := "---\nid: x\ncontent: |\n    # Title\nother: value\n"
		sys, home := memShapeEnv(t, map[string]string{"-a": odd})
		var buf bytes.Buffer
		rep := capture(&buf)
		checkMemoryShape(sys, rep, true)

		got, _ := os.ReadFile(filepath.Join(home, ".claude", "projects", "-a", "memory", "MEMORY.md"))
		if string(got) != odd {
			t.Error("an unrecognised shape was rewritten")
		}
		if !strings.Contains(buf.String(), "shape not recognised") {
			t.Errorf("expected a WARN naming the refusal, got\n%s", buf.String())
		}
	})
}
