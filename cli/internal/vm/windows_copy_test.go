package vm

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWindowsCopyBothWays(t *testing.T) {
	s, g := heldWindows(t, nil)
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	if err := os.WriteFile(in, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rec := s.Copy(context.Background(), in, `proof:C:\t\a b`, io.Discard); rec.Status != StatusOK || rec.Exit != nil {
		t.Fatalf("to the guest: %+v", rec)
	}
	if string(g.files[`C:\t\a b`]) != "payload" {
		t.Errorf("guest file = %q", g.files[`C:\t\a b`])
	}
	out := filepath.Join(dir, "out")
	if rec := s.Copy(context.Background(), `proof:C:\t\a b`, out, io.Discard); rec.Status != StatusOK {
		t.Fatalf("from the guest: %+v", rec)
	}
	if b, _ := os.ReadFile(out); string(b) != "payload" {
		t.Errorf("host file = %q", b)
	}
}

func TestWindowsCopyKeepsTheTargetsMode(t *testing.T) {
	s, g := heldWindows(t, nil)
	g.files[`C:\t\a`] = []byte("secret")
	out := filepath.Join(t.TempDir(), "out")
	if err := os.WriteFile(out, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if rec := s.Copy(context.Background(), `proof:C:\t\a`, out, io.Discard); rec.Status != StatusOK {
		t.Fatalf("Copy = %+v", rec)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o", st.Mode().Perm())
	}
}

func TestWindowsCopyFailures(t *testing.T) {
	s, g := heldWindows(t, nil)
	out := filepath.Join(t.TempDir(), "out")
	rec := s.Copy(context.Background(), `proof:C:\nothing`, out, io.Discard)
	if rec.Status != StatusFailed {
		t.Errorf("missing guest file: %+v", rec)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a failed copy left a host file")
	}
	dir := t.TempDir()
	kept := filepath.Join(dir, "kept")
	if err := os.WriteFile(kept, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rec := s.Copy(context.Background(), `proof:C:\nothing`, kept, io.Discard); rec.Status != StatusFailed {
		t.Errorf("missing guest file: %+v", rec)
	}
	if b, _ := os.ReadFile(kept); string(b) != "mine" {
		t.Errorf("a failed copy changed the target: %q", b)
	}
	if left, _ := os.ReadDir(dir); len(left) != 1 {
		t.Errorf("a failed copy left %v", left)
	}
	g.files[`C:\x`] = []byte("kept")
	if rec := s.Copy(context.Background(), t.TempDir(), `proof:C:\x`, io.Discard); rec.Status != StatusFailed {
		t.Errorf("a directory: %+v", rec)
	}
	if string(g.files[`C:\x`]) != "kept" {
		t.Error("a refused copy reached the guest file")
	}
	if rec := s.Copy(context.Background(), out, `ghost:C:\x`, io.Discard); rec.Status != StatusNoSession {
		t.Errorf("unknown session: %+v", rec)
	}
}
