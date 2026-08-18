package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDocument(t *testing.T) {
	content := `---
id: pattern-socratic-diagnostic-trees
name: socratic-diagnostic-trees
type: pattern
status: active
tags: [architecture, debugging, socratic]
keywords: [troubleshoot, root cause, decision tree]
summary: Structured Socratic diagnostic trees for technical decisions and systematic debugging.
---

# Socratic Diagnostic Trees Pattern

When debugging complex systems, map hypothesis nodes and refute them sequentially.
`
	doc, err := ParseDocument("00_meta/patterns/pattern-socratic-diagnostic-trees.md", []byte(content))
	if err != nil {
		t.Fatalf("ParseDocument error: %v", err)
	}

	if doc.ID != "pattern-socratic-diagnostic-trees" {
		t.Errorf("got ID %q, want %q", doc.ID, "pattern-socratic-diagnostic-trees")
	}
	if doc.Type != TypePattern {
		t.Errorf("got Type %q, want %q", doc.Type, TypePattern)
	}
	if doc.Title != "socratic-diagnostic-trees" {
		t.Errorf("got Title %q, want %q", doc.Title, "socratic-diagnostic-trees")
	}
	if len(doc.Tags) != 3 {
		t.Errorf("got %d tags, want 3", len(doc.Tags))
	}
}

func TestSearch(t *testing.T) {
	tmpDir := t.TempDir()

	patsDir := filepath.Join(tmpDir, "00_meta", "patterns")
	skillsDir := filepath.Join(tmpDir, "00_meta", "skills", "systematic-debugging")
	if err := os.MkdirAll(patsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}

	patFile := filepath.Join(patsDir, "pattern-socratic-trees.md")
	patContent := `---
id: pattern-socratic-trees
name: socratic-trees
type: pattern
tags: [diagnostic, architecture]
keywords: [hypotheses, root cause]
description: Guide to socratic trees
---
# Socratic Trees
Use decision trees for root cause isolation.
`
	if err := os.WriteFile(patFile, []byte(patContent), 0644); err != nil {
		t.Fatal(err)
	}

	skillFile := filepath.Join(skillsDir, "SKILL.md")
	skillContent := `---
id: systematic-debugging-skill
name: systematic-debugging
type: skill
tags: [debugging, troubleshooting]
keywords: [bug, defect, reproduce]
description: Four-phase systematic debugging workflow
---
# Systematic Debugging Skill
Phase 1: Root cause investigation before changing code.
`
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test 1: Search by keyword
	res, err := Search(tmpDir, "debugging", TypeAll, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected search hits for 'debugging', got 0")
	}
	if res[0].ID != "systematic-debugging-skill" {
		t.Errorf("expected top result to be systematic-debugging-skill, got %s", res[0].ID)
	}

	// Test 2: Search with type filter
	resPats, err := Search(tmpDir, "trees", TypePattern, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(resPats) != 1 || resPats[0].Type != TypePattern {
		t.Errorf("expected 1 pattern result, got %v", resPats)
	}

	// Test 3: Search with no match
	resEmpty, err := Search(tmpDir, "nonexistentxyzterm", TypeAll, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(resEmpty) != 0 {
		t.Errorf("expected 0 results, got %d", len(resEmpty))
	}
}
