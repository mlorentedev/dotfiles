package copilotsettings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	repo := t.TempDir()
	writePolicy(t, repo, `{
		"version": 1,
		"retired_paths": ["telemetry.disabled", "legacy.flag"]
	}`)

	got, err := LoadPolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"telemetry.disabled", "legacy.flag"}
	if got.Version != 1 || !reflect.DeepEqual(got.RetiredPaths, want) {
		t.Fatalf("LoadPolicy() = %+v, want version 1 and %v", got, want)
	}
}

func TestLoadPolicyRejectsInvalidContracts(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"unknown field", `{"version":1,"retired_paths":["telemetry.disabled"],"extra":true}`, "unknown field"},
		{"unsupported version", `{"version":2,"retired_paths":["telemetry.disabled"]}`, "unsupported version"},
		{"empty list", `{"version":1,"retired_paths":[]}`, "must not be empty"},
		{"empty path", `{"version":1,"retired_paths":[""]}`, "retired path"},
		{"duplicate path", `{"version":1,"retired_paths":["telemetry.disabled","telemetry.disabled"]}`, "duplicate"},
		{"overlapping path", `{"version":1,"retired_paths":["telemetry","telemetry.disabled"]}`, "overlapping"},
		{"empty segment", `{"version":1,"retired_paths":["telemetry..disabled"]}`, "retired path"},
		{"leading dot", `{"version":1,"retired_paths":[".telemetry"]}`, "retired path"},
		{"trailing dot", `{"version":1,"retired_paths":["telemetry."]}`, "retired path"},
		{"path separator", `{"version":1,"retired_paths":["telemetry/disabled"]}`, "retired path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writePolicy(t, repo, tc.body)
			_, err := LoadPolicy(repo)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("LoadPolicy() error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestLoadPolicyMissingFailsLoud(t *testing.T) {
	if _, err := LoadPolicy(t.TempDir()); err == nil {
		t.Fatal("LoadPolicy() accepted a missing policy")
	}
}

func TestLoadPolicyCurrentRepository(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	got, err := LoadPolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RetiredPaths, []string{"telemetry.disabled"}) {
		t.Fatalf("current retired paths = %v", got.RetiredPaths)
	}
}

func writePolicy(t *testing.T, repo, body string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(PolicyRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
