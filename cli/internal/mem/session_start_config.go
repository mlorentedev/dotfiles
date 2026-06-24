package mem

import (
	"encoding/json"
	"os"
)

// adapterConfig is the SDD-004 session-start-config.json contract: numeric
// thresholds + per-injector enable flags. It is the Go port of the shell hook's
// cfg_threshold / cfg_injector_enabled helpers. A missing file, unreadable JSON, or
// absent key falls back to the historical default so a partial install never breaks
// SessionStart.
type adapterConfig struct {
	Thresholds map[string]int `json:"thresholds"`
	Injectors  map[string]struct {
		// Enabled is a pointer so an absent flag ("config silent on this injector")
		// is distinguishable from an explicit false; absent defaults to enabled.
		Enabled *bool `json:"enabled"`
	} `json:"injectors"`
}

// loadAdapterConfig reads session-start-config.json from path. A missing or
// malformed file yields a non-nil zero config whose accessors return the defaults —
// callers never need a nil check.
func loadAdapterConfig(path string) *adapterConfig {
	cfg := &adapterConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return &adapterConfig{} // partial install / corrupt file -> all defaults
	}
	return cfg
}

// threshold returns the configured value for key, or def when the config is absent
// or the key is missing (cfg_threshold).
func (c *adapterConfig) threshold(key string, def int) int {
	if c != nil {
		if v, ok := c.Thresholds[key]; ok {
			return v
		}
	}
	return def
}

// injectorEnabled reports whether the named injector is on. It defaults to true
// (the historical "all on" behaviour) unless the config explicitly sets it false
// (cfg_injector_enabled).
func (c *adapterConfig) injectorEnabled(key string) bool {
	if c == nil {
		return true
	}
	inj, ok := c.Injectors[key]
	if !ok || inj.Enabled == nil {
		return true
	}
	return *inj.Enabled
}
