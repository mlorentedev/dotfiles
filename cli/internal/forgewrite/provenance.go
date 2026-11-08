package forgewrite

import (
	"fmt"
	"regexp"
	"strings"
)

var forbiddenProvenanceTokens = regexp.MustCompile(`(?i)\b(claude|copilot|agy|antigravity|opencode|codex|gemini|meeseek|meeseeks|builder|reviewer|architect|planner|shipper|curator)\b|(?:^|[^a-z0-9-])(pi)(?:$|[^a-z0-9-])`)

// FormatEnvironmentProvenance formats the cross-OS forge provenance footer line
// (HARNESS-186, ADR-060):
//
//	Environment: <machine-id> (<goos>/<goarch>) · <lane>
//
// It carries only machine, OS/arch, and lane topology — never AI model, agent,
// persona, or meeseek worker names (m2, m3).
func FormatEnvironmentProvenance(machineID, goos, goarch, lane string) string {
	machineID = sanitizeMachineID(strings.TrimSpace(machineID))
	if machineID == "" {
		machineID = "unknown-host"
	}
	goos = strings.TrimSpace(goos)
	if goos == "" {
		goos = "unknown-os"
	}
	goarch = strings.TrimSpace(goarch)
	if goarch == "" {
		goarch = "unknown-arch"
	}
	lane = strings.TrimSpace(lane)
	if lane == "" {
		lane = "main"
	}
	return fmt.Sprintf("Environment: %s (%s/%s) · %s", machineID, goos, goarch, lane)
}

func sanitizeMachineID(machineID string) string {
	if machineID == "" {
		return ""
	}
	if forbiddenProvenanceTokens.MatchString(machineID) {
		return "host-node"
	}
	return machineID
}

// ValidateProvenanceNoAttribution verifies that a provenance line contains zero
// AI/agent/persona/meeseek identifiers (enforcing the global no-attribution rule).
func ValidateProvenanceNoAttribution(s string) error {
	if m := forbiddenProvenanceTokens.FindString(s); m != "" {
		return fmt.Errorf("provenance string %q contains forbidden agent/persona/meeseek attribution token %q", s, strings.TrimSpace(m))
	}
	return nil
}

// AppendEnvironmentProvenance idempotently appends the Environment provenance
// line to a PR description or review triage comment body.
func AppendEnvironmentProvenance(body, machineID, goos, goarch, lane string) string {
	line := FormatEnvironmentProvenance(machineID, goos, goarch, lane)
	if strings.Contains(body, line) {
		return body
	}
	trimmed := strings.TrimRight(body, "\n")
	if trimmed == "" {
		return line + "\n"
	}
	return trimmed + "\n\n" + line + "\n"
}
