package claude

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

func TestTruncatedBoundaries(t *testing.T) {
	floor := int64(mem.ClaudeJSONMinBytes)
	cases := []struct {
		name          string
		snapshot, now int64
		want          bool
	}{
		{"healthy file shrank by more than half", 20000, 9999, true},
		{"exactly half is not truncation", 20000, 10000, false},
		{"one byte under half is truncation", 20001, 10000, true},
		{"snapshot at the floor is guarded", floor, 1500, true},
		{"snapshot one byte under the floor is not guarded", floor - 1, 1500, false},
		{"file grew", 20000, 30000, false},
		{"file emptied", 20000, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Truncated(c.snapshot, c.now, floor); got != c.want {
				t.Fatalf("Truncated(%d, %d, %d) = %v, want %v", c.snapshot, c.now, floor, got, c.want)
			}
		})
	}
}

// healthyClaudeJSON is a .claude.json above the floor, with no hooks key.
func healthyClaudeJSON(t *testing.T) []byte {
	t.Helper()
	doc := map[string]any{"oauthAccount": map[string]any{"emailAddress": "x"}, "padding": strings.Repeat("a", 20000)}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestGuardRestoresATruncatedFileByteForByte(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	snapshot := healthyClaudeJSON(t)
	if err := os.WriteFile(path, snapshot, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := Guard(path, mem.ClaudeJSONMinBytes, func() error {
		return os.WriteFile(path, []byte(`{"truncated":true}`), 0o600)
	})
	if err != nil || !restored {
		t.Fatalf("Guard = (%v, %v), want (true, nil)", restored, err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(snapshot) {
		t.Fatalf("restored content differs from the snapshot (%d vs %d bytes)", len(got), len(snapshot))
	}
}

// AC6: the Go path emits no `hooks` key under any input. The only file this
// increment writes is .claude.json, and only as a byte copy of its own
// snapshot; this pins that the restore adds nothing, so a later "helpful"
// normalisation cannot start writing keys into it.
func TestGuardNeverAddsAHooksKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(path, healthyClaudeJSON(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Guard(path, mem.ClaudeJSONMinBytes, func() error {
		return os.WriteFile(path, []byte(`{}`), 0o600)
	}); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	data, _ := os.ReadFile(path)
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["hooks"]; ok {
		t.Fatal("restored .claude.json carries a hooks key")
	}
}

func TestGuardLeavesAHealthyChangeAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(path, healthyClaudeJSON(t), 0o600); err != nil {
		t.Fatal(err)
	}
	grown := append(healthyClaudeJSON(t), ' ')
	restored, err := Guard(path, mem.ClaudeJSONMinBytes, func() error { return os.WriteFile(path, grown, 0o600) })
	if err != nil || restored {
		t.Fatalf("Guard = (%v, %v), want (false, nil)", restored, err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(grown) {
		t.Fatal("a legitimate write by the CLI was reverted")
	}
}

func TestGuardRunsTheActionWhenThereIsNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	ran := false
	restored, err := Guard(path, mem.ClaudeJSONMinBytes, func() error { ran = true; return nil })
	if err != nil || restored || !ran {
		t.Fatalf("Guard = (%v, %v), ran=%v; want (false, nil), ran=true", restored, err, ran)
	}
}

func TestGuardRestoresEvenWhenTheActionFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	snapshot := healthyClaudeJSON(t)
	if err := os.WriteFile(path, snapshot, 0o600); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("install failed")
	restored, err := Guard(path, mem.ClaudeJSONMinBytes, func() error {
		_ = os.WriteFile(path, []byte(`{}`), 0o600)
		return boom
	})
	if !errors.Is(err, boom) || !restored {
		t.Fatalf("Guard = (%v, %v), want (true, %v)", restored, err, boom)
	}
	if got, _ := os.ReadFile(path); string(got) != string(snapshot) {
		t.Fatal("snapshot not restored after a failing action")
	}
}
