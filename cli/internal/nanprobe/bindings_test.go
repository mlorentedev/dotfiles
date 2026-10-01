package nanprobe

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCollectJoinsTheMapAndEveryPinSite(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(repo, ".pr_agent.toml"), "[config]\nmodel = \"openai/mimo-v2.6-flash\"\n")
	writeFile(t, filepath.Join(home, ".pi/agent/settings.json"), `{"defaultModel":"deepseek-v4-flash"}`)

	m := map[string]any{
		"tiers":  map[string]any{"mid": map[string]any{"nan": "deepseek-v4-flash", "claude": "sonnet"}},
		"chains": map[string]any{"review": []any{"nan:mimo-v2.6-flash", "claude:opus"}},
		"services": map[string]any{
			"rerank":  map[string]any{"pool": "nan", "model": "rerank"},
			"advisor": map[string]any{"pool": "claude", "model": "fable"},
		},
	}
	pins := &harness.ModelPins{Sites: []harness.PinSite{
		{File: ".pr_agent.toml", Scope: "repo", Pins: []harness.Pin{
			{ID: "primary", Kind: "toml-key", Locator: "model", Prefix: "openai/", Pool: "nan"}}},
		{File: "$HOME/.pi/agent/settings.json", Scope: "deployed", Pins: []harness.Pin{
			{ID: "pi", Kind: "json-path", Locator: "defaultModel", Pool: "nan"}}},
		{File: "$HOME/.absent/settings.json", Scope: "deployed", Pins: []harness.Pin{
			{ID: "absent", Kind: "json-path", Locator: "model", Pool: "nan"}}},
	}}

	b, notes, err := Collect(m, pins, repo, home, "nan")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tg := range b.Targets() {
		got = append(got, tg.Model+"="+string(tg.API))
	}
	if strings.Join(got, ",") != "deepseek-v4-flash=chat,mimo-v2.6-flash=chat,rerank=rerank" {
		t.Errorf("targets %q: only the nan pool, from the map and the pins, each on its own API", got)
	}
	if got := strings.Join(b.Files["mimo-v2.6-flash"], ","); got != "harness/model-map.json,.pr_agent.toml" {
		t.Errorf("mimo bound in %q: the report must name every file to edit", got)
	}
	if got := strings.Join(b.Files["deepseek-v4-flash"], ","); got != "harness/model-map.json,$HOME/.pi/agent/settings.json" {
		t.Errorf("deepseek bound in %q", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], ".absent") {
		t.Errorf("an absent deployed site is reported as not read, not silently skipped: %v", notes)
	}
}

func TestCollectRefusesALocatorThatMatchesNothing(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, ".pr_agent.toml"), "[config]\nother = \"x\"\n")
	pins := &harness.ModelPins{Sites: []harness.PinSite{
		{File: ".pr_agent.toml", Scope: "repo", Pins: []harness.Pin{
			{ID: "primary", Kind: "toml-key", Locator: "model", Prefix: "openai/", Pool: "nan"}}},
	}}
	// Zero values would read as zero dead models: a false all-clear.
	if _, _, err := Collect(map[string]any{}, pins, repo, t.TempDir(), "nan"); err == nil {
		t.Fatal("a rotted locator must fail the canary, not shrink what it probes")
	}
}

func TestReportPutsRowsNeedingActionFirstAndQuotaDoesNotFail(t *testing.T) {
	results := []Result{
		{Model: "fine", Status: 200, Class: Answered},
		{Model: "spent", Status: 402, Class: Quota, Detail: "HTTP 402"},
		{Model: "gone", Status: 401, Class: Refused, Detail: "HTTP 401"},
	}
	b := Bindings{Files: map[string][]string{"gone": {".pr_agent.toml"}}}

	if !Failing(results) {
		t.Error("a refusal needs a person")
	}
	if Failing(results[:2]) {
		t.Error("quota resets on its own and must not open an issue")
	}

	var buf bytes.Buffer
	if err := WriteMarkdown(&buf, results, b, []string{"x is not present here"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Index(out, "`gone`") > strings.Index(out, "`fine`") {
		t.Errorf("the refused row must come first:\n%s", out)
	}
	if !strings.Contains(out, "replace it in every file listed") || !strings.Contains(out, "`.pr_agent.toml`") {
		t.Errorf("a refused row names the remedy and the files:\n%s", out)
	}
	if !strings.Contains(out, "Not read:") {
		t.Errorf("notes are rendered:\n%s", out)
	}
}
