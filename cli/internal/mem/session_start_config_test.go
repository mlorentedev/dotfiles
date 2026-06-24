package mem

import (
	"path/filepath"
	"testing"
)

func TestLoadAdapterConfig(t *testing.T) {
	cfgJSON := `{
	  "thresholds": { "memory_md_max_lines": 150, "crystallize_max_days": 14 },
	  "injectors": {
	    "doctor_drift": { "enabled": true },
	    "memory_temperature": { "enabled": false }
	  }
	}`
	dir := t.TempDir()
	path := filepath.Join(dir, "session-start-config.json")
	mustWrite(t, path, cfgJSON)
	cfg := loadAdapterConfig(path)

	t.Run("threshold reads a configured value", func(t *testing.T) {
		if got := cfg.threshold("memory_md_max_lines", 999); got != 150 {
			t.Errorf("got %d, want 150", got)
		}
	})

	t.Run("threshold falls back to default for a missing key", func(t *testing.T) {
		if got := cfg.threshold("claude_json_min_bytes", 10240); got != 10240 {
			t.Errorf("got %d, want default 10240", got)
		}
	})

	t.Run("injectorEnabled honours an explicit false", func(t *testing.T) {
		if cfg.injectorEnabled("memory_temperature") {
			t.Error("memory_temperature should be disabled")
		}
	})

	t.Run("injectorEnabled defaults true for an absent injector", func(t *testing.T) {
		if !cfg.injectorEnabled("claude_json_size") {
			t.Error("an unlisted injector should default to enabled")
		}
	})
}

func TestLoadAdapterConfigMissingFileUsesDefaults(t *testing.T) {
	cfg := loadAdapterConfig(filepath.Join(t.TempDir(), "absent.json"))
	if got := cfg.threshold("memory_md_max_lines", 150); got != 150 {
		t.Errorf("missing config should yield default, got %d", got)
	}
	if !cfg.injectorEnabled("doctor_drift") {
		t.Error("missing config should default injectors to enabled")
	}
}
