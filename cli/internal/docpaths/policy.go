package docpaths

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const PolicyVersion = 1

type Policy struct {
	Comment           []string `json:"$comment"`
	Version           int      `json:"version"`
	GovernedBasenames []string `json:"governed_basenames"`
	ExcludePrefixes   []string `json:"exclude_prefixes"`
	KnownExtensions   []string `json:"known_extensions"`
	RequiredMembers   []string `json:"required_members"`
	SourceExtensions  []string `json:"source_extensions"`
}

func LoadPolicy(path string) (Policy, error) {
	f, err := os.Open(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read doc-path policy: %w", err)
	}
	defer func() { _ = f.Close() }()

	var policy Policy
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("parse doc-path policy: %w", err)
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
		return fmt.Errorf("parse doc-path policy: %w", err)
	}
	return fmt.Errorf("parse doc-path policy: trailing JSON value")
}

func validatePolicy(policy Policy) error {
	if policy.Version != PolicyVersion {
		return fmt.Errorf("unsupported version %d (want %d)", policy.Version, PolicyVersion)
	}
	if len(policy.GovernedBasenames) == 0 {
		return fmt.Errorf("governed_basenames must not be empty")
	}
	for _, prefix := range policy.ExcludePrefixes {
		clean := filepath.ToSlash(filepath.Clean(prefix))
		if filepath.IsAbs(prefix) || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("exclude_prefix %q must stay inside the repository", prefix)
		}
	}
	return nil
}
