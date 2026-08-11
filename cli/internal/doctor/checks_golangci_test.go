package doctor

import (
	"bytes"
	"strings"
	"testing"
)

// TestGolangciVersionExtraction pins the parser against the real output of both
// majors. v1 and v2 differ by a leading `v`, and BUG-071 was a v1-vs-v2 drift —
// a parser that only understood the version it was written against would report
// the other as unrecognised and hide the exact case it exists to catch.
func TestGolangciVersionExtraction(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
	}{
		{
			name: "v2 output (no leading v)",
			out:  `golangci-lint has version 2.12.2 built with go1.26.0 from (unknown, modified: ?) on (unknown)`,
			want: "2.12.2",
		},
		{
			name: "v1 output (leading v)",
			out:  `golangci-lint has version v1.62.2 built with go1.26.0 from (unknown, modified: ?) on (unknown)`,
			want: "1.62.2",
		},
		{
			name: "unrecognised output yields no version rather than a wrong one",
			out:  "some other tool entirely",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := newSys(nil, []string{"golangci-lint"},
				map[string]string{"golangci-lint --version": tt.out})
			if got := golangciVersion(sys); got != tt.want {
				t.Errorf("golangciVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckGolangciLint(t *testing.T) {
	const v2 = `golangci-lint has version 2.12.2 built with go1.26.0 from (unknown) on (unknown)`
	const v1 = `golangci-lint has version v1.62.2 built with go1.26.0 from (unknown) on (unknown)`

	tests := []struct {
		name       string
		onPath     []string
		cmdOut     map[string]string
		pin        string
		wantSubstr string
		wantFail   bool
	}{
		{
			name:       "installed version matches the pin",
			onPath:     []string{"golangci-lint"},
			cmdOut:     map[string]string{"golangci-lint --version": v2},
			pin:        "2.12.2",
			wantSubstr: "matches versions.conf",
		},
		{
			// The BUG-071 case: this is what the machine looked like while
			// `golangci-lint run` reported 0 issues and CI failed.
			name:       "two majors behind the pin warns rather than passing silently",
			onPath:     []string{"golangci-lint"},
			cmdOut:     map[string]string{"golangci-lint --version": v1},
			pin:        "2.12.2",
			wantSubstr: "golangci-lint version drift: installed=1.62.2 pinned=2.12.2",
		},
		{
			name:       "absent tool skips and names the pinned install command",
			pin:        "2.12.2",
			wantSubstr: "@v2.12.2",
		},
		{
			name:       "absent tool with no pin still skips cleanly",
			wantSubstr: "not installed",
		},
		{
			name:       "unparseable version output is reported, not treated as a match",
			onPath:     []string{"golangci-lint"},
			cmdOut:     map[string]string{"golangci-lint --version": "mystery build"},
			pin:        "2.12.2",
			wantSubstr: "not recognised",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			rep := capture(&buf)
			cfg := &Config{Versions: map[string]string{}}
			if tt.pin != "" {
				cfg.Versions["GOLANGCI_LINT_VERSION"] = tt.pin
			}

			checkGolangciLint(newSys(nil, tt.onPath, tt.cmdOut), cfg, rep)

			got := buf.String()
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("report missing %q; got:\n%s", tt.wantSubstr, got)
			}
			// Drift must never fail the run: the tool works, it just cannot
			// speak for CI. Same posture as every other pinned-tool check.
			if !tt.wantFail && rep.ExitCode() != 0 {
				t.Errorf("check must not FAIL (exit=%d); got:\n%s", rep.ExitCode(), got)
			}
		})
	}
}
