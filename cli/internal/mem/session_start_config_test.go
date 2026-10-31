package mem

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// The committed config declares every injector flag the adapter reads, so an
// owner can turn each one off where the file says it can be. The bats pins the
// file's keys; this pins them to the flags the code reads (#2915). readFlags
// lists them by hand, so the count of injectorEnabled call sites in the
// package's source is checked against it: a new flag read fails here until it
// is listed, and so declared.
func TestCommittedConfigDeclaresTheInjectorsTheAdapterReads(t *testing.T) {
	readFlags := []string{resumeInjector}
	if n := injectorReadSites(t); n != len(readFlags) {
		t.Fatalf("the package calls injectorEnabled at %d sites but readFlags lists %d; list the new flag and declare it in session-start-config.json", n, len(readFlags))
	}
	cfg := loadAdapterConfig(filepath.Join("..", "..", "..", "session-start-config.json"))
	if len(cfg.Injectors) == 0 {
		t.Fatal("the repo-root session-start-config.json did not load")
	}
	for _, key := range readFlags {
		if inj, ok := cfg.Injectors[key]; !ok || inj.Enabled == nil {
			t.Errorf("session-start-config.json does not declare injectors.%s.enabled", key)
		}
	}
}

// injectorReadSites counts the calls to injectorEnabled in the package's
// non-test source, the definition excluded.
func injectorReadSites(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("list the package source: %v", err)
	}
	call := regexp.MustCompile(`\.injectorEnabled\(`)
	n := 0
	for _, e := range entries {
		f := e.Name()
		if !strings.HasSuffix(f, ".go") || strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f) // #nosec G304 -- this package's own source
		if err != nil {
			t.Fatal(err)
		}
		n += len(call.FindAllIndex(src, -1))
	}
	return n
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
