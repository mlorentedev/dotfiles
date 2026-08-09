package memshape

import (
	"errors"
	"strings"
	"testing"
)

// The fixtures here are synthetic on purpose. The real corpus — 23 before/after
// pairs from the 2026-05-26 wrap — lives in the private knowledge vault and is
// exercised by TestUnwrapAgainstVaultGroundTruth, which skips when the vault is
// absent. dotfiles is a public repository, so committing those files would
// publish infrastructure notes and session history. These fixtures reproduce the
// SHAPES; the ground-truth test proves the shapes are the real ones.

func TestIsWrapped(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"plain markdown", "# Title\n\n## Section\n", false},
		{"frontmatter but no block scalar", "---\nid: x\ntype: memory\n---\n\n# Title\n", false},
		{"wrapped under content", "---\nid: x\ncontent: |\n    # Title\n", true},
		{"wrapped under another key", "---\nid: x\nbody: |\n    # Title\n", true},
		{"strip indicator", "---\nid: x\ncontent: |-\n    # Title\n", true},
		{"keep indicator", "---\nid: x\ncontent: |+\n    # Title\n", true},
		{"explicit indent indicator", "---\nid: x\ncontent: |2\n  # Title\n", true},
		{"no leading ---", "id: x\ncontent: |\n    # Title\n", false},
		{"empty", "", false},
		// A body line that merely looks like an opener must not trip detection
		// from column 0 — but a real opener is at column 0, so an indented
		// lookalike inside a body is not a false positive here.
		{"pipe inside prose", "---\nid: x\n---\n\nSee `foo: |` for details\n", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsWrapped(tc.src); got != tc.want {
				t.Errorf("IsWrapped = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			// hive's shape: every body line at the same depth. residual == 0, so
			// this is the regression case against over-stripping.
			name: "uniform indent",
			src: "---\nid: \"x-memory\"\ntype: memory\ncontent: |\n" +
				"    # x - Project Memory\n" +
				"\n" +
				"    ## Section\n" +
				"    - item\n",
			want: "---\nid: \"x-memory\"\ntype: memory\n---\n\n" +
				"# x - Project Memory\n" +
				"\n" +
				"## Section\n" +
				"- item\n",
		},
		{
			// The 2026-05-26 wrapper's shape, 16 of 17 files: first body line one
			// level shallower than the rest. Stopping at the YAML-correct indent
			// (4) would leave every later line at column 2, where crystallize's
			// column-0 markers still would not match.
			name: "first line shallower than the rest",
			src: "---\nid: \"x-memory\"\ncontent: |\n" +
				"    # x - Project Memory\n" +
				"      \n" +
				"      ## Section\n" +
				"      - item\n",
			want: "---\nid: \"x-memory\"\n---\n\n" +
				"# x - Project Memory\n" +
				"\n" +
				"## Section\n" +
				"- item\n",
		},
		{
			// Two trailing spaces are the markdown hard break the handoff
			// convention mandates. They are content and must survive untouched —
			// this is exactly what a yaml.v3 re-emit destroys.
			name: "markdown hard breaks survive",
			src: "---\nid: x\ncontent: |\n" +
				"    ## Session Handoff\n" +
				"      > Updated: 2026-08-09  \n" +
				"      **Last task:** something  \n",
			want: "---\nid: x\n---\n\n" +
				"## Session Handoff\n" +
				"> Updated: 2026-08-09  \n" +
				"**Last task:** something  \n",
		},
		{
			// A genuinely indented markdown construct must not be flattened: one
			// line at the block indent forces residual to 0.
			name: "nested content keeps its relative indent",
			src: "---\nid: x\ncontent: |\n" +
				"    # Title\n" +
				"    - top\n" +
				"      - nested\n",
			want: "---\nid: x\n---\n\n" +
				"# Title\n" +
				"- top\n" +
				"  - nested\n",
		},
		{
			name: "single-line body",
			src:  "---\nid: x\ncontent: |\n    # Only\n",
			want: "---\nid: x\n---\n\n# Only\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Unwrap(tc.src)
			if err != nil {
				t.Fatalf("Unwrap: %v", err)
			}
			if got != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", visible(got), visible(tc.want))
			}
		})
	}
}

func TestUnwrapRefusesWhatItDoesNotUnderstand(t *testing.T) {
	t.Run("plain markdown is not wrapped", func(t *testing.T) {
		if _, err := Unwrap("# Title\n\n## Section\n"); !errors.Is(err, ErrNotWrapped) {
			t.Errorf("want ErrNotWrapped, got %v", err)
		}
	})

	t.Run("frontmatter-only file is not wrapped", func(t *testing.T) {
		// The shape #864 migrates TO. It must not be re-processed.
		if _, err := Unwrap("---\nid: x\n---\n\n# Title\n"); !errors.Is(err, ErrNotWrapped) {
			t.Errorf("want ErrNotWrapped, got %v", err)
		}
	})

	t.Run("a key after the block ends it — refuse rather than guess", func(t *testing.T) {
		src := "---\nid: x\ncontent: |\n    # Title\nother: value\n"
		if _, err := Unwrap(src); err == nil || errors.Is(err, ErrNotWrapped) {
			t.Errorf("want a refusal for an unrecognised shape, got %v", err)
		}
	})

	t.Run("empty block", func(t *testing.T) {
		if _, err := Unwrap("---\nid: x\ncontent: |\n"); err == nil {
			t.Error("want an error for a block with no content")
		}
	})
}

// TestUnwrapIsIdempotent pins the property doctor --fix relies on: a second run
// must be a no-op, not a second transform.
func TestUnwrapIsIdempotent(t *testing.T) {
	src := "---\nid: x\ncontent: |\n    # Title\n      \n      ## Section\n"

	once, err := Unwrap(src)
	if err != nil {
		t.Fatalf("first Unwrap: %v", err)
	}
	if IsWrapped(once) {
		t.Fatal("output still reports as wrapped — doctor --fix would loop")
	}
	if _, err := Unwrap(once); !errors.Is(err, ErrNotWrapped) {
		t.Errorf("second Unwrap should decline, got %v", err)
	}
}

func visible(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "·"), "\n", "⏎\n")
}
