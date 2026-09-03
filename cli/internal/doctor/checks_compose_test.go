package doctor

import (
	"bytes"
	"strings"
	"testing"
)

// TestCheckDockerCompose pins the one tool setup's check_dependencies named that
// doctor covered nowhere. Compose v2 ships as a `docker` CLI plugin, not a
// binary, so a PATH test alone answers wrongly on a current install — measured
// on msi 2026-09-02, where the v1 binary and plugin v2.39.1 are both present.
// The repo provisions compose in no installer, versions.conf entry or contract
// binary, so absence is a SKIP; a FAIL would red every box that never wanted it
// (the BUG-052 reasoning that put terraform in optionalTools).
func TestCheckDockerCompose(t *testing.T) {
	cases := []struct {
		name         string
		onPath       []string
		cmdOut       map[string]string
		wantFailures int
		wantSubstr   string
	}{
		{
			name:       "v2 plugin present → pass naming the version",
			onPath:     []string{"docker"},
			cmdOut:     map[string]string{"docker compose version": "Docker Compose version v2.39.1"},
			wantSubstr: "v2.39.1",
		},
		{
			// A box still on the standalone binary: compose works, so this is
			// not a failure, but the plugin is the supported form.
			name:       "only the v1 binary → pass, flagged legacy",
			onPath:     []string{"docker", "docker-compose"},
			wantSubstr: "legacy",
		},
		{
			name:       "docker present, no compose either way → skip",
			onPath:     []string{"docker"},
			wantSubstr: "not installed",
		},
		{
			// Without docker there is nothing for a plugin to hang off; the
			// core-tools section already reports docker itself.
			name:       "no docker → skip without duplicating the core-tools verdict",
			wantSubstr: "docker",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(nil, tc.onPath, tc.cmdOut)
			var buf bytes.Buffer
			rep := capture(&buf)
			checkDockerCompose(sys, rep)

			if rep.Failures() != tc.wantFailures {
				t.Fatalf("failures = %d, want %d\n%s", rep.Failures(), tc.wantFailures, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantSubstr) {
				t.Fatalf("output missing %q\n%s", tc.wantSubstr, buf.String())
			}
		})
	}
}
