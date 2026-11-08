package forgewrite

import (
	"strings"
	"testing"
)

func TestForgePRProvenanceNoAttribution(t *testing.T) {
	got := FormatEnvironmentProvenance("EGW-LEN029", "windows", "amd64", "feat-lane-primary")
	want := "Environment: EGW-LEN029 (windows/amd64) · feat-lane-primary"
	if got != want {
		t.Fatalf("FormatEnvironmentProvenance = %q, want %q", got, want)
	}

	if err := ValidateProvenanceNoAttribution(got); err != nil {
		t.Fatalf("expected canonical provenance %q to pass ValidateProvenanceNoAttribution, got %v", got, err)
	}

	// AppendEnvironmentProvenance idempotently appends the line to PR/comment bodies.
	body := "## Summary\n\nImplemented feature."
	withProv := AppendEnvironmentProvenance(body, "EGW-LEN029", "windows", "amd64", "feat-lane-primary")
	if !strings.Contains(withProv, want) {
		t.Fatalf("AppendEnvironmentProvenance missing %q in:\n%s", want, withProv)
	}
	// Calling a second time must not duplicate the line.
	withProv2 := AppendEnvironmentProvenance(withProv, "EGW-LEN029", "windows", "amd64", "feat-lane-primary")
	if strings.Count(withProv2, "Environment:") != 1 {
		t.Fatalf("expected idempotent AppendEnvironmentProvenance, got:\n%s", withProv2)
	}

	// Hostnames like raspberry-pi or builder must not fail ValidateProvenanceNoAttribution.
	for _, host := range []string{"raspberry-pi", "pi", "builder"} {
		line := FormatEnvironmentProvenance(host, "linux", "arm64", "feat-lane-primary")
		if err := ValidateProvenanceNoAttribution(line); err != nil {
			t.Fatalf("expected host %q provenance %q to pass ValidateProvenanceNoAttribution, got %v", host, line, err)
		}
	}

	// Forbidden AI/agent/persona/meeseek tokens must be rejected (m2, m3).
	forbiddenInputs := []string{
		"Environment: EGW-LEN029 (windows/amd64) · claude",
		"Environment: EGW-LEN029 (windows/amd64) · copilot",
		"Environment: EGW-LEN029 (windows/amd64) · agy",
		"Environment: EGW-LEN029 (windows/amd64) · pi",
		"Environment: EGW-LEN029 (windows/amd64) · opencode",
		"Environment: EGW-LEN029 (windows/amd64) · meeseek-reviewer-avocado",
		"Environment: EGW-LEN029 (windows/amd64) · builder",
		"Environment: EGW-LEN029 (windows/amd64) · reviewer",
		"Environment: EGW-LEN029 (windows/amd64) · architect",
		"Environment: EGW-LEN029 (windows/amd64) · planner",
		"Environment: EGW-LEN029 (windows/amd64) · shipper",
		"Environment: EGW-LEN029 (windows/amd64) · curator",
	}
	for _, bad := range forbiddenInputs {
		if err := ValidateProvenanceNoAttribution(bad); err == nil {
			t.Errorf("expected ValidateProvenanceNoAttribution(%q) to fail, got nil", bad)
		}
	}
}
