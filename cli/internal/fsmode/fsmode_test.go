package fsmode

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// OwnerOnly is the one rule the Windows branch keys on: any bit for group or
// other means the inherited ACL stays.
func TestOwnerOnly(t *testing.T) {
	cases := []struct {
		mode os.FileMode
		want bool
	}{
		{0o600, true}, {0o400, true}, {0o700, true},
		{0o644, false}, {0o640, false}, {0o604, false}, {0o755, false}, {0o666, false},
	}
	for _, tc := range cases {
		if got := OwnerOnly(tc.mode); got != tc.want {
			t.Errorf("OwnerOnly(%#o) = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

// Needs is the read-only mirror of Apply: true while Apply would change
// something, false right after it ran — on every OS, whatever "something"
// means there (bits on POSIX, the read-only attribute and the DACL on Windows).
func TestNeeds_MirrorsApply(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	needs, err := Needs(p, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Fatal("a fresh 0644 file must need the owner-only mode")
	}
	if err := Apply(p, 0o600); err != nil {
		t.Fatal(err)
	}
	needs, err = Needs(p, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if needs {
		t.Fatal("right after Apply nothing must be left to apply")
	}
	if _, err := Needs(filepath.Join(t.TempDir(), "absent"), 0o600); err == nil {
		t.Fatal("an absent file is an error, not a no")
	}
}

// Apply is os.Chmod everywhere: the POSIX bits land on POSIX, and the
// read-only attribute lands on Windows for a mode without the owner write bit.
func TestApply_IsChmodOnEveryOS(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(p, 0o400); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if fi.Mode().Perm()&0o200 != 0 {
			t.Fatalf("0400 must clear the write bit (read-only attribute), got %v", fi.Mode())
		}
		return
	}
	if fi.Mode().Perm() != 0o400 {
		t.Fatalf("perm = %v, want 0400", fi.Mode().Perm())
	}
}
