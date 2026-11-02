// Package spechome resolves which GitHub repositories a spec command asks
// about. It is its own leaf package because spec cannot import initrepo:
// initrepo reaches vault, which reaches mem, which imports spec.
package spechome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
)

// bitacoraRepoFile lets a repository declare, committed, the GitHub
// repository that holds its issues: one owner/name. It is the sibling of
// .wip-limit (ADR-043) and, like it, a malformed file is an error (#1906).
const bitacoraRepoFile = ".bitacora-repo"

// Homes are the two GitHub repositories a spec command asks about. They differ
// when a repository's issues live elsewhere: a kubelab spec gated by a
// knowledge issue, or a Gitea-hosted repository whose bitácora is on GitHub.
type Homes struct {
	// Issues completes bare #N links and hosts the work-gate issue.
	Issues string
	// Archive holds specs/archive/ on GitHub: the github.com origin, or ""
	// when the specs live on another forge that gh cannot ask.
	Archive string
}

// Inputs are what Resolve reads. The caller supplies the origin URL
// so a check that reads git through its own seam stays hermetic.
type Inputs struct {
	Root      string
	Flag      string // the command's explicit override
	FlagName  string // how the refusal names that flag
	Env       string // DOTF_BITACORA_REPO
	OriginURL string
}

// Resolve is the one answer `spec init`, `spec audit` and the doctor
// check share. The issue home is the flag, then specs/.bitacora-repo, then
// DOTF_BITACORA_REPO, then a github.com origin. The committed file beats the
// variable because the variable is machine-wide: a value exported for one
// repository must not redirect another that declared its own. Anything else
// fails closed: a non-GitHub origin is never turned into a slug, because gh
// would ask github.com about it.
func Resolve(in Inputs) (Homes, error) {
	var h Homes
	if s, err := initrepo.ParseOriginRepo(in.OriginURL); err == nil {
		h.Archive = s
	}
	file, err := readBitacoraRepo(in.Root)
	if err != nil {
		return Homes{}, err
	}
	var from string
	switch {
	case in.Flag != "":
		h.Issues, from = in.Flag, in.FlagName
	case file != "":
		h.Issues, from = file, filepath.Join("specs", bitacoraRepoFile)
	case in.Env != "":
		h.Issues, from = in.Env, "DOTF_BITACORA_REPO"
	case h.Archive != "":
		h.Issues = h.Archive
	default:
		return Homes{}, noIssueHome(in)
	}
	if !initrepo.ValidRepoSlug(h.Issues) {
		return Homes{}, fmt.Errorf("%s: %q is not owner/name", from, h.Issues)
	}
	return h, nil
}

// readBitacoraRepo returns the declared owner/name, or "" without the file.
func readBitacoraRepo(root string) (string, error) {
	rel := filepath.Join("specs", bitacoraRepoFile)
	b, err := os.ReadFile(filepath.Join(root, rel))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if !initrepo.ValidRepoSlug(s) {
		return "", fmt.Errorf("%s must hold one GitHub owner/name, got %q", rel, s)
	}
	return s, nil
}

// noIssueHome names the origin by forge and host only: a self-hosted remote
// often embeds a token, and this message reaches terminals and transcripts.
func noIssueHome(in Inputs) error {
	forge := initrepo.ClassifyURL(in.Root, in.OriginURL)
	origin := "no origin remote in " + in.Root
	if forge != initrepo.ForgeAbsent {
		origin = fmt.Sprintf("the origin is a %s remote at %s, and only a github.com origin names one", forge, initrepo.OriginHost(in.OriginURL))
	}
	return fmt.Errorf("cannot tell which GitHub repository holds this repository's issues: %s.\n"+
		"Declare it: pass %s owner/name, or commit %s holding owner/name",
		origin, in.FlagName, filepath.Join("specs", bitacoraRepoFile))
}
