package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePiConfig plants a deployed pi config under a fake $HOME.
func writePiConfig(t *testing.T, home, body string) {
	t.Helper()
	p := filepath.Join(home, piConfigRel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func piSys(home string) *System {
	return &System{
		Getenv:   func(k string) string { return map[string]string{"HOME": home}[k] },
		LookPath: func(n string) (string, error) { return "/usr/bin/" + n, nil },
	}
}

func piConfig(apiKey string) string {
	return `{"providers":{"nan":{"baseUrl":"https://api.nan.builders/v1","apiKey":"` + apiKey + `"}}}`
}

func TestAgentConfig_SeverityByShape(t *testing.T) {
	const secret = "sk-LIVE-CREDENTIAL-CANARY"
	for _, tt := range []struct {
		name     string
		apiKey   string
		wantTag  string
		wantSaid string
	}{
		{
			// The defect the setup script called "resolved at runtime". pi sends
			// this verbatim as the bearer token; the server can only 401.
			name: "dotf placeholder pi cannot resolve", apiKey: "{env:NAN_API_KEY}",
			wantTag: "[FAIL]", wantSaid: "not pi's",
		},
		{
			// The successful deploy path, and the worse defect: a credential at
			// rest in a config someone opens to debug the first one.
			name: "materialised literal", apiKey: secret,
			wantTag: "[FAIL]", wantSaid: "literal credential",
		},
		{
			name: "pi braced syntax", apiKey: "${NAN_API_KEY}", wantTag: "[ OK ]",
		},
		{
			// Both forms are pi syntax. Accepting only the braced one would fail
			// a correct config, which is how a guard gets disabled.
			name: "pi bare syntax", apiKey: "$NAN_API_KEY", wantTag: "[ OK ]",
		},
		{
			// pi also executes a leading `!` as a command. Not the chosen shape
			// here, but it resolves, so it must not be reported as broken.
			name: "pi command syntax", apiKey: "!dotf secrets show NAN_API_KEY", wantTag: "[ OK ]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			writePiConfig(t, home, piConfig(tt.apiKey))
			var buf bytes.Buffer
			rep := capture(&buf)

			checkAgentConfigSecrets(piSys(home), rep)

			out := buf.String()
			if !strings.Contains(out, tt.wantTag) {
				t.Errorf("want %s, got:\n%s", tt.wantTag, out)
			}
			if tt.wantSaid != "" && !strings.Contains(out, tt.wantSaid) {
				t.Errorf("the failure must say why (%q), got:\n%s", tt.wantSaid, out)
			}
			// The rule the guard itself must obey: reporting a credential on disk
			// must never involve printing it.
			if strings.Contains(out, secret) {
				t.Errorf("the check printed the credential it was reporting:\n%s", out)
			}
		})
	}
}

// A machine without pi has nothing broken. A red section nobody can act on is
// how operators learn to ignore the whole report.
func TestAgentConfig_AbsentConfigSkips(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)

	checkAgentConfigSecrets(piSys(t.TempDir()), rep)

	if rep.Failures() != 0 {
		t.Fatalf("an absent pi config must not FAIL:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "[SKIP]") {
		t.Errorf("want SKIP for an absent config, got:\n%s", buf.String())
	}
}

// Malformed JSON is its own state: the secrets could not be checked, which is
// not the same as their being fine.
func TestAgentConfig_UnparseableIsWarnedNotPassed(t *testing.T) {
	home := t.TempDir()
	writePiConfig(t, home, `{"providers": <<<not json>>>}`)
	var buf bytes.Buffer
	rep := capture(&buf)

	checkAgentConfigSecrets(piSys(home), rep)

	out := buf.String()
	if strings.Contains(out, "[ OK ]") {
		t.Errorf("an unparseable config must not report its secrets as fine:\n%s", out)
	}
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("want WARN for an unparseable config, got:\n%s", out)
	}
}
