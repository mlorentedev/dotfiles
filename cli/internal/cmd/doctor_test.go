package cmd

import (
	"strings"
	"testing"
)

func TestDoctorCmdHelp(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutSub string
	}{
		{"help lists the doctor command", []string{"--help"}, "doctor"},
		{"doctor --help describes the sweep", []string{"doctor", "--help"}, "diagnostic"},
		{"doctor exposes --fix", []string{"doctor", "--help"}, "--fix"},
		{"doctor exposes --verbose", []string{"doctor", "--help"}, "--verbose"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := execute(t, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			combined := stdout + stderr
			if !strings.Contains(combined, tt.wantOutSub) {
				t.Errorf("output %q does not contain %q", combined, tt.wantOutSub)
			}
		})
	}
}

func TestDoctorCmdRejectsArgs(t *testing.T) {
	// doctor takes no positional args; a stray one is an error (silenced from
	// stdout/stderr, but still surfaced as a non-nil error → exit 1 in main).
	if _, _, err := execute(t, "doctor", "bogus"); err == nil {
		t.Error("expected an error for an unexpected positional arg")
	}
}
