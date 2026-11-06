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
	for _, name := range policy.GovernedBasenames {
		if name == "" || filepath.Base(name) != name {
			return fmt.Errorf("governed_basename %q must be one file name", name)
		}
	}
	for _, prefix := range policy.ExcludePrefixes {
		if !repoRelative(prefix) {
			return fmt.Errorf("exclude_prefix %q must stay inside the repository", prefix)
		}
	}
	if err := validateExtensions("known_extension", policy.KnownExtensions); err != nil {
		return err
	}
	if err := validateExtensions("source_extension", policy.SourceExtensions); err != nil {
		return err
	}
	for _, member := range policy.RequiredMembers {
		if !repoRelative(member) {
			return fmt.Errorf("required_member %q must stay inside the repository", member)
		}
	}
	return nil
}

func validateExtensions(label string, extensions []string) error {
	for _, ext := range extensions {
		if len(ext) < 2 || ext[0] != '.' || strings.ContainsAny(ext, `/\`) {
			return fmt.Errorf("%s %q must start with a dot and contain no path separator", label, ext)
		}
	}
	return nil
}

func repoRelative(value string) bool {
	if value == "" {
		return false
	}
	slash := filepath.ToSlash(filepath.Clean(value))
	if filepath.IsAbs(value) || strings.HasPrefix(slash, "/") ||
		(len(slash) >= 2 && slash[1] == ':') {
		return false
	}
	return slash != ".." && !strings.HasPrefix(slash, "../")
}
