package initrepo

import (
	"strings"
	"testing"
)

func TestReadTemplateReturnsEmbeddedContent(t *testing.T) {
	got, err := ReadTemplate("agents-spec-section.md")
	if err != nil {
		t.Fatalf("ReadTemplate: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("ReadTemplate returned empty content")
	}
	// The agents snippet is delimited by BEGIN/END SNIPPET markers; their
	// presence is what init agents will extract. Guards against vendoring the
	// wrong file.
	for _, marker := range []string{"## --- BEGIN SNIPPET ---", "## --- END SNIPPET ---"} {
		if !strings.Contains(string(got), marker) {
			t.Errorf("embedded agents-spec-section.md missing marker %q", marker)
		}
	}
}

func TestReadTemplateUnknownErrors(t *testing.T) {
	if _, err := ReadTemplate("does-not-exist.md"); err == nil {
		t.Error("expected error for an unknown template name")
	}
}
