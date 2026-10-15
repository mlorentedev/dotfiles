package forge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// fetchTimeout bounds CheckFresh's fetch, so a stalled origin degrades to a
// refusal instead of a hung check or apply.
const fetchTimeout = 60 * time.Second

// CheckFresh fetches origin and confirms the checkout's declaration is the one
// origin's default branch holds. A checkout that is behind, or carries an
// unmerged edit, would otherwise answer check with a false OK and make apply
// write a declaration nobody merged: on 2026-10-15 a stale main checkout
// reported strict UNCHANGED while main declared it false.
//
// It refuses when that cannot be told, too: a failed fetch, or no default
// branch on origin.
func CheckFresh(root string) error {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "git", "-C", root, "fetch", "--quiet", "origin").CombinedOutput(); err != nil {
		return fmt.Errorf("cannot tell whether %s is current: fetching origin failed: %v %s",
			DeclarationFile, err, strings.TrimSpace(string(out)))
	}
	ref, same, err := LandedDeclaration(root)
	if err != nil {
		return fmt.Errorf("cannot tell whether %s is current: %w", DeclarationFile, err)
	}
	if !same {
		return fmt.Errorf("%s differs from %s's: this checkout would act on a declaration that has not landed", DeclarationFile, ref)
	}
	return nil
}

// LandedDeclaration compares the checkout's declaration with the one origin's
// default branch holds, as last fetched; it does not fetch. ref names that
// branch; a declaration absent from it is not the same.
//
// Only remote-tracking refs count as the default branch. A fallback to a local
// main would compare the file with itself and answer "same" forever, so a
// checkout with no origin default is an error, not a match.
func LandedDeclaration(root string) (ref string, same bool, err error) {
	ref = defaultRemoteRef(root)
	if ref == "" {
		return "", false, fmt.Errorf("origin has no default branch (no origin/HEAD, origin/main or origin/master)")
	}
	local, err := os.ReadFile(filepath.Join(root, DeclarationFile))
	if err != nil {
		return ref, false, err
	}
	landed, err := exec.Command("git", "-C", root, "show", ref+":"+DeclarationFile).Output()
	return ref, err == nil && bytes.Equal(local, landed), nil
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
