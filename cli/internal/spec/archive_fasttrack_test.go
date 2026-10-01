package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveFastTrack(t *testing.T) {
	tests := []struct {
		name      string
		abandoned bool
		expected  string
	}{
		{
			name:      "archived status",
			abandoned: false,
			expected:  "status: archived",
		},
		{
			name:      "abandoned status",
			abandoned: true,
			expected:  "status: abandoned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			id := "FTD-001"
			specDir := filepath.Join(repoRoot, "specs", id)
			if err := os.MkdirAll(specDir, 0755); err != nil {
				t.Fatal(err)
			}

			specContent := `---
status: active
---
# FTD-001

## Intent
Intent

## Scope
Scope

## Checklist
- [x] done

## Evidence
Evidence

## Promotion candidates
- [ ] Lesson? no: not needed
- [ ] ADR? no: not needed
- [ ] Pattern? no: not needed
`
			if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(specContent), 0644); err != nil {
				t.Fatal(err)
			}

			opts := ArchiveOptions{
				Abandoned: tt.abandoned,
				VaultRoot: func() (string, error) { return "/fake/vault", nil },
			}

			target, err := Archive(repoRoot, id, opts)
			if err != nil {
				t.Fatalf("Archive failed for Fast-Track spec: %v", err)
			}

			expectedTarget := filepath.Join(repoRoot, "specs", "archive", id)
			if tt.abandoned {
				expectedTarget = filepath.Join(repoRoot, "specs", "archive", "_abandoned", id)
			}
			if target != expectedTarget {
				t.Errorf("expected target %s, got %s", expectedTarget, target)
			}

			data, err := os.ReadFile(filepath.Join(target, "spec.md"))
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(string(data), tt.expected) {
				t.Errorf("expected %q, got:\n%s", tt.expected, string(data))
			}
		})
	}
}
