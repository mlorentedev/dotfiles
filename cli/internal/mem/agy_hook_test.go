package mem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFirstInvocation(t *testing.T) {
	state := t.TempDir()
	first := func(id string) bool {
		t.Helper()
		got, err := FirstInvocation(state, id)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if !first("conv-1") || first("conv-1") || first("conv-1") {
		t.Error("a conversation is first exactly once")
	}
	if !first("conv-2") {
		t.Error("another conversation has its own first invocation")
	}
	if first("") {
		t.Error("an empty id is never first: the hook could not tell turns apart and would repeat on every one")
	}
}

// An id is data from the harness: it names a file, never a path.
func TestFirstInvocationKeepsAnIdInsideTheStateDirectory(t *testing.T) {
	state := t.TempDir()
	if got, err := FirstInvocation(state, "../../escape/..\\x:y"); err != nil || !got {
		t.Fatalf("FirstInvocation = %v, %v", got, err)
	}
	entries, _ := os.ReadDir(filepath.Join(state, agyHookDir))
	if len(entries) != 1 || entries[0].IsDir() {
		t.Fatalf("want one marker file inside %s, got %v", agyHookDir, entries)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(state), "escape")); err == nil {
		t.Error("the id created a path outside the state directory")
	}
}

func TestFirstInvocationPrunesOldMarkersOnly(t *testing.T) {
	state := t.TempDir()
	dir := filepath.Join(state, agyHookDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old, recent := filepath.Join(dir, "old"), filepath.Join(dir, "recent")
	for _, p := range []string{old, recent} {
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-2 * agyMarkerTTL)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	if _, err := FirstInvocation(state, "fresh"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); err == nil {
		t.Error("a marker older than the TTL stayed")
	}
	if _, err := os.Stat(recent); err != nil {
		t.Errorf("a recent marker went: %v", err)
	}
}

// agy requires a JSON object on stdout and reads the text from a nested
// systemMessage step (measured against its embedded hooks documentation).
func TestAgyInjection(t *testing.T) {
	out, err := AgyInjection("lane `x` ACTIVE on `b`\nsecond line")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		InjectSteps []struct {
			SystemMessage struct {
				SystemMessage string `json:"systemMessage"`
			} `json:"systemMessage"`
		} `json:"injectSteps"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got.InjectSteps) != 1 ||
		got.InjectSteps[0].SystemMessage.SystemMessage != "lane `x` ACTIVE on `b`\nsecond line" {
		t.Fatalf("injection = %q (%v), want one nested systemMessage step", out, err)
	}
	if empty, _ := AgyInjection(" \n"); empty != "{}\n" {
		t.Errorf("no text must inject nothing, got %q", empty)
	}
}

func TestParseAgyHook(t *testing.T) {
	got := ParseAgyHook([]byte(`{"conversationId":"c","cwd":"/w","hookEventName":"PreInvocation"}`))
	if got.ConversationID != "c" || got.Cwd != "/w" {
		t.Errorf("got %+v", got)
	}
	if got := ParseAgyHook([]byte("not json")); got != (AgyHookPayload{}) {
		t.Errorf("malformed payload gave %+v, want empty fields", got)
	}
}

// Two long ids that share a prefix, or two ids the sanitiser maps to the same
// text, are two conversations: each is first once, and neither silences the other.
func TestFirstInvocationKeepsDistinctIdsDistinct(t *testing.T) {
	state := t.TempDir()
	prefix := strings.Repeat("a", 100)
	ids := []string{prefix + strings.Repeat("x", 20), prefix + strings.Repeat("y", 20), "a/b", "a_b", "a:b"}
	for _, id := range ids {
		if got, err := FirstInvocation(state, id); err != nil || !got {
			t.Errorf("FirstInvocation(%q) = %v, %v; want first", id, got, err)
		}
	}
	for _, id := range ids {
		if got, _ := FirstInvocation(state, id); got {
			t.Errorf("FirstInvocation(%q) was first twice", id)
		}
	}
	if markerName("conv-1") != "conv-1" {
		t.Errorf("a clean short id must keep its own name, got %q", markerName("conv-1"))
	}
	if n := len(markerName(ids[0])); n > 96 {
		t.Errorf("marker name is %d characters, want at most 96", n)
	}
}
