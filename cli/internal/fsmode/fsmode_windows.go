//go:build windows

package fsmode

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// restrictToOwner replaces the file's DACL with two entries — the calling
// user and LocalSystem, full access each — and marks it protected, so nothing
// inherited from the directory applies any more. That is what 0600 means on
// this OS: "mine alone", with the one account that already reads every
// profile (backup, Defender) kept so those services do not start failing on
// a file we hardened.
//
// The user is the TOKEN's user, never a name looked up from the environment:
// CI's Windows leg runs as a service account, and a name-based lookup would
// harden the wrong principal or none. Administrators are not listed on
// purpose — they hold SeTakeOwnershipPrivilege regardless, and an entry for
// them would be a claim the OS does not enforce.
func restrictToOwner(path string) error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return fmt.Errorf("owner-only ACL for %s: process token: %w", path, err)
	}
	defer func() { _ = token.Close() }()
	user, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("owner-only ACL for %s: token user: %w", path, err)
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return fmt.Errorf("owner-only ACL for %s: LocalSystem SID: %w", path, err)
	}
	entries := []windows.EXPLICIT_ACCESS{
		grantAll(user.User.Sid, windows.TRUSTEE_IS_USER),
		grantAll(system, windows.TRUSTEE_IS_WELL_KNOWN_GROUP),
	}
	dacl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return fmt.Errorf("owner-only ACL for %s: build DACL: %w", path, err)
	}
	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil,
	); err != nil {
		return fmt.Errorf("owner-only ACL for %s: %w", path, err)
	}
	return nil
}

// ownerOnlyApplied reads the DACL's protected flag: a protected DACL is one
// restrictToOwner wrote (nothing inherited applies); an unprotected one is the
// directory's, however its entries happen to read today.
func ownerOnlyApplied(path string) (bool, error) {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return false, fmt.Errorf("read ACL of %s: %w", path, err)
	}
	control, _, err := sd.Control()
	if err != nil {
		return false, fmt.Errorf("read ACL control of %s: %w", path, err)
	}
	return control&windows.SE_DACL_PROTECTED != 0, nil
}

// permDiffers on Windows compares the one bit os.Chmod can express: the owner
// write bit, backed by the read-only attribute. The rest of the POSIX bits
// have no counterpart here and would read as drift forever.
func permDiffers(got, want os.FileMode) bool {
	return got.Perm()&0o200 != want.Perm()&0o200
}

func grantAll(sid *windows.SID, kind windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  kind,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}
