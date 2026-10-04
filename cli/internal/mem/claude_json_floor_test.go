package mem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeJSONFloorReadsTheConfiguredThreshold(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "session-start-config.json")
	if err := os.WriteFile(cfg, []byte(`{"thresholds":{"claude_json_min_bytes":4096}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ClaudeJSONFloor(cfg); got != 4096 {
		t.Fatalf("configured floor = %d, want 4096", got)
	}
	if got := ClaudeJSONFloor(filepath.Join(dir, "absent.json")); got != ClaudeJSONMinBytes {
		t.Fatalf("default floor = %d, want %d", got, ClaudeJSONMinBytes)
	}
}
