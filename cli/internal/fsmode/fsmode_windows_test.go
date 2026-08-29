//go:build windows

package fsmode

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// currentUserSID resolves the calling user the way the implementation does —
// from the token — so the assertion holds under CI's service account too.
func currentUserSID(t *testing.T) string {
	t.Helper()
	var tok windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &tok); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tok.Close() }()
	u, err := tok.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	return u.User.Sid.String()
}

// icaclsEntries returns the ACE lines icacls prints for path — the tool an
// administrator would use to check, so the test verifies the consequence and
// not the call.
func icaclsEntries(t *testing.T, path string) []string {
	t.Helper()
	out, err := exec.Command("icacls", path).CombinedOutput()
	if err != nil {
		t.Fatalf("icacls %s: %v\n%s", path, err, out)
	}
	var entries []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Successfully processed") || strings.HasPrefix(line, "processed") {
			continue
		}
		// The first line carries the path; entries follow, one per line, and
		// the path line's own entry sits after the path.
		if i := strings.Index(line, path); i == 0 {
			line = strings.TrimSpace(line[len(path):])
		}
		if line != "" {
			entries = append(entries, line)
		}
	}
	return entries
}

// AC1: 0600 becomes a protected DACL with exactly the user and SYSTEM.
func TestApply_OwnerOnlyModeSetsAProtectedOwnerOnlyDACL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(p, 0o600); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("the DACL must be protected: inherited entries would otherwise reappear")
	}

	entries := icaclsEntries(t, p)
	if len(entries) != 2 {
		t.Fatalf("want exactly two ACEs (user, SYSTEM), got %d:\n%s", len(entries), strings.Join(entries, "\n"))
	}
	joined := strings.Join(entries, "\n")
	if strings.Contains(joined, "(I)") {
		t.Fatalf("no entry may be inherited:\n%s", joined)
	}
	if !strings.Contains(joined, "NT AUTHORITY\\SYSTEM") && !strings.Contains(joined, "S-1-5-18") {
		t.Fatalf("SYSTEM must keep access:\n%s", joined)
	}
	// icacls prints the user by name; resolve the token SID to its name to
	// compare, falling back to the SID string when the name is unavailable.
	sid := currentUserSID(t)
	psid, err := windows.StringToSid(sid)
	if err != nil {
		t.Fatal(err)
	}
	account, domain, _, err := psid.LookupAccount("")
	if err != nil {
		if !strings.Contains(joined, sid) {
			t.Fatalf("the calling user (%s) must keep access:\n%s", sid, joined)
		}
		return
	}
	if !strings.Contains(joined, domain+"\\"+account) && !strings.Contains(joined, sid) {
		t.Fatalf("the calling user (%s\\%s) must keep access:\n%s", domain, account, joined)
	}
}

// A file written by anything other than Apply carries its directory's DACL.
// Needs must call that a fix waiting to happen for an owner-only mode, and
// nothing at all for a shared one — the question Deploy asks about a file
// whose content is already in sync.
func TestNeeds_SeesAnInheritedDACLAsMissingOwnerOnly(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inherited.txt")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	needs, err := Needs(p, 0o600)
	if err != nil || !needs {
		t.Fatalf("an inherited DACL on a 0600 must need the fix, got needs=%v err=%v", needs, err)
	}
	needs, err = Needs(p, 0o644)
	if err != nil || needs {
		t.Fatalf("a shared mode has nothing to fix on an inherited DACL, got needs=%v err=%v", needs, err)
	}
}

// AC2: a mode with group/other bits leaves the inherited ACL alone.
func TestApply_SharedModeKeepsTheInheritedACL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "public.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := icaclsEntries(t, p)
	if err := Apply(p, 0o644); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	after := icaclsEntries(t, p)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("0644 must not rewrite the ACL\nbefore:\n%s\nafter:\n%s", strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
	sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED != 0 {
		t.Fatal("a shared mode must not protect the DACL")
	}
}
