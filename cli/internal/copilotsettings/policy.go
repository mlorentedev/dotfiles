package copilotsettings

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyRel is the checkout-owned retired Copilot settings policy.
const (
	PolicyRel     = "ai/copilot/retired-settings.json"
	policyVersion = 1
)

var pathSegment = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Policy declares local Copilot setting paths that current runtimes reject.
type Policy struct {
	Comment      []string `json:"$comment"`
	Version      int      `json:"version"`
	RetiredPaths []string `json:"retired_paths"`
}

// LoadPolicy strictly loads the checkout's versioned retired-settings policy.
func LoadPolicy(repo string) (Policy, error) {
	path := filepath.Join(repo, filepath.FromSlash(PolicyRel))
	f, err := os.Open(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read Copilot retired-settings policy: %w", err)
	}
	defer func() { _ = f.Close() }()

	var policy Policy
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("parse Copilot retired-settings policy: %w", err)
	}
	if err := rejectTrailingJSON(dec); err != nil {
		return Policy{}, err
	}
	if err := validatePolicy(policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func rejectTrailingJSON(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("parse Copilot retired-settings policy: %w", err)
	}
	return fmt.Errorf("parse Copilot retired-settings policy: trailing JSON value")
}

func validatePolicy(policy Policy) error {
	if policy.Version != policyVersion {
		return fmt.Errorf("unsupported version %d (want %d)", policy.Version, policyVersion)
	}
	if len(policy.RetiredPaths) == 0 {
		return fmt.Errorf("retired_paths must not be empty")
	}
	seen := make(map[string]bool, len(policy.RetiredPaths))
	for _, path := range policy.RetiredPaths {
		if !validPath(path) {
			return fmt.Errorf("retired path %q must be dotted object keys", path)
		}
		if seen[path] {
			return fmt.Errorf("duplicate retired path %q", path)
		}
		for other := range seen {
			if strings.HasPrefix(path, other+".") || strings.HasPrefix(other, path+".") {
				return fmt.Errorf("overlapping retired paths %q and %q", other, path)
			}
		}
		seen[path] = true
	}
	return nil
}

func validPath(path string) bool {
	if path == "" {
		return false
	}
	start := 0
	for i := 0; i <= len(path); i++ {
		if i != len(path) && path[i] != '.' {
			continue
		}
		if !pathSegment.MatchString(path[start:i]) {
			return false
		}
		start = i + 1
	}
	return true
}
