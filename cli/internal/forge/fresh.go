package forge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CheckFresh fetches origin and confirms the checkout's declaration is the one
// origin's default branch holds. A checkout that is behind, or carries an
// unmerged edit, would otherwise answer check with a false OK and make apply
// write a declaration nobody merged: on 2026-10-15 a stale main checkout
// reported strict UNCHANGED while main declared it false.
//
// Only remote-tracking refs count as the default branch. A fallback to a local
// main would compare the file with itself and answer fresh forever, so a
// checkout with no reachable origin is refused, not passed.
func CheckFresh(root string) error {
	if out, err := exec.Command("git", "-C", root, "fetch", "--quiet", "origin").CombinedOutput(); err != nil {
		return fmt.Errorf("cannot tell whether %s is current: fetching origin failed: %v %s",
			DeclarationFile, err, strings.TrimSpace(string(out)))
	}
	ref := defaultRemoteRef(root)
	if ref == "" {
		return fmt.Errorf("cannot tell whether %s is current: origin has no default branch (no origin/HEAD, origin/main or origin/master)", DeclarationFile)
	}
	local, err := os.ReadFile(filepath.Join(root, DeclarationFile))
	if err != nil {
		return err
	}
	merged, err := exec.Command("git", "-C", root, "show", ref+":"+DeclarationFile).Output()
	if err != nil || !bytes.Equal(local, merged) {
		return fmt.Errorf("%s differs from %s's: this checkout would act on a declaration that has not landed. Pull, or pass --unmerged to use this checkout's file", DeclarationFile, ref)
	}
	return nil
}

func defaultRemoteRef(root string) string {
	if out, err := exec.Command("git", "-C", root, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD").Output(); err == nil {
		if ref := strings.TrimSpace(string(out)); ref != "" {
			return ref
		}
	}
	for _, ref := range []string{"origin/main", "origin/master"} {
		if exec.Command("git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/remotes/"+ref).Run() == nil {
			return ref
		}
	}
	return ""
}
