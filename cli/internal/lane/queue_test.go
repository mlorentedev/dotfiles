package lane

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeIssues map[int]Issue

func (f fakeIssues) Issue(n int) (Issue, error) {
	i, ok := f[n]
	if !ok {
		return Issue{}, errors.New("issue not found")
	}
	return i, nil
}

// The next item is the first queued issue that is open and nobody holds, and
// asking twice answers the same: nothing in the queue is rotated or written, so
// two sessions cannot disagree and no commit sits between two bugs.
func TestQueue_NextIsTheFirstOpenUnassignedIssueAndIsStable(t *testing.T) {
	src := fakeIssues{
		2501: {Number: 2501, State: "CLOSED"},
		2434: {Number: 2434, State: "OPEN", Assignees: []string{"manu"}},
		2279: {Number: 2279, State: "OPEN", Title: "BUG-117: x"},
		2300: {Number: 2300, State: "OPEN"},
	}
	cfg := Config{Queue: []int{2501, 2434, 2279, 2300}}
	for range 2 {
		got, ok, err := Next(cfg, src)
		if err != nil || !ok || got.Number != 2279 {
			t.Fatalf("Next = %+v, %v, %v; want 2279", got, ok, err)
		}
	}
	if _, ok, err := Next(Config{Queue: []int{2501, 2434}}, src); ok || err != nil {
		t.Errorf("nothing free: ok=%v err=%v; want ok=false and no error", ok, err)
	}
	if _, _, err := Next(Config{Queue: []int{9999}}, src); err == nil || !strings.Contains(err.Error(), "9999") {
		t.Errorf("an unreadable issue is an error naming it, got %v", err)
	}
}

func TestSlug_DropsTheTicketIDAndKeepsAWordBoundedBranchName(t *testing.T) {
	for title, want := range map[string]string{
		"BUG-143: doctor warns that an optional, defaulted HERDR_CONFIG_DIR is unset and --fix suggests a manual profile edit": "doctor-warns-that-an-optional-defaulted-herdr",
		"CLI-119: a fix lane has no successor": "a-fix-lane-has-no-successor",
		"plain title":                          "plain-title",
		"  !!  ":                               "lane",
	} {
		if got := Slug(title); got != want {
			t.Errorf("Slug(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestRenderPrompt_FillsTheTemplateAndRefusesAnUnknownField(t *testing.T) {
	d := PromptData{Number: 2434, Title: "BUG-131: x", Branch: "fix/x", Remaining: []int{2279}}
	got, err := RenderPrompt("Bug #{{.Number}} on {{.Branch}}; then {{range .Remaining}}#{{.}} {{end}}", d)
	if err != nil || got != "Bug #2434 on fix/x; then #2279 " {
		t.Errorf("RenderPrompt = %q, %v", got, err)
	}
	if _, err := RenderPrompt("{{.Nope}}", d); err == nil {
		t.Error("a field the data does not have must fail, not render empty")
	}
}

func TestLoadConfig_DefaultsAndRefusesAnUnknownKey(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, ".dotf", "lane.json"), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"queue":[2434,2279],"template":".dotf/lane-prompt.md"}`)
	cfg, err := LoadConfig(root)
	want := Config{Queue: []int{2434, 2279}, BranchPrefix: "fix", Base: "origin/main", Template: ".dotf/lane-prompt.md"}
	if err != nil || !reflect.DeepEqual(cfg, want) {
		t.Errorf("LoadConfig = %+v, %v; want %+v", cfg, err, want)
	}
	write(`{"queue":[1],"quue":[2]}`)
	if _, err := LoadConfig(root); err == nil {
		t.Error("a misspelt key must fail, not be dropped")
	}
	if _, err := LoadConfig(t.TempDir()); err == nil {
		t.Error("a repo with no .dotf/lane.json has no lane")
	}
}
