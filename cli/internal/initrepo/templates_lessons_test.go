package initrepo

import (
	"bytes"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lessons"
)

// The index `dotf init` scaffolds is exactly what `dotf lessons fmt` renders
// for no lessons, so every repository starts in the shape fmt maintains.
func TestLessonsIndexTemplateIsWhatFmtRenders(t *testing.T) {
	tmpl, err := templatesFS.ReadFile("templates/lessons-index.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := lessons.RenderIndex(tmpl, nil); !bytes.Equal(got, tmpl) {
		t.Errorf("fmt would rewrite the scaffolded index:\n%s", got)
	}
}
