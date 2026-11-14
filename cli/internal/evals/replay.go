package evals

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// caseIDPattern is the schema's id pattern. A case id names a directory of the
// run, so the checkout holds it to the pattern even for a case nobody validated.
var caseIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Replay is one case's checkout: a detached worktree at its reviewed commit,
// owned by the run and removed when it ends.
type Replay struct {
	Dir  string
	repo string
}

// CheckoutReplay checks out c's reviewed commit, detached, at
// <runDir>/<case>-<epoch>. A commit the repository lacks is fetched from
// origin by its sha, then through the case's fetch_ref, without creating a
// ref. A case whose commit cannot be resolved fails naming the case: it is a
// failure of the run, never a score.
func CheckoutReplay(repo string, c Case, runDir string, epoch int) (Replay, error) {
	if !caseIDPattern.MatchString(c.ID) {
		return Replay{}, fmt.Errorf("case id %q is not a case name (%s), and it names the replay directory", c.ID, caseIDPattern)
	}
	if !isSHA(c.ReviewedSHA) {
		return Replay{}, fmt.Errorf("case %s: reviewed_sha %q is not a full sha", c.ID, c.ReviewedSHA)
	}
	if err := outsideRepo(repo, runDir); err != nil {
		return Replay{}, fmt.Errorf("case %s: %w", c.ID, err)
	}
	if err := resolveCommit(repo, c); err != nil {
		return Replay{}, err
	}
	dir := filepath.Join(runDir, fmt.Sprintf("%s-%d", c.ID, epoch))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return Replay{}, fmt.Errorf("case %s: %w", c.ID, err)
	}
	if out, err := gitIn(repo, "worktree", "add", "--quiet", "--detach", dir, c.ReviewedSHA); err != nil {
		_ = os.RemoveAll(dir)
		_, _ = gitIn(repo, "worktree", "prune")
		return Replay{}, fmt.Errorf("case %s: check out %s: %w\n%s", c.ID, c.ReviewedSHA, err, out)
	}
	return Replay{Dir: dir, repo: repo}, nil
}

// Remove deletes the replay worktree and its registration in the repository.
func (r Replay) Remove() error {
	out, err := gitIn(r.repo, "worktree", "remove", "--force", r.Dir)
	if _, pruneErr := gitIn(r.repo, "worktree", "prune"); err == nil && pruneErr != nil {
		err = pruneErr
	}
	if err != nil {
		return fmt.Errorf("remove replay %s: %w\n%s", r.Dir, err, out)
	}
	return nil
}

// resolveCommit makes c's reviewed commit present in repo, fetching it when it
// is absent. A fetch names no destination, so it creates no ref and writes no
// FETCH_HEAD: the replay worktree is what keeps the commit.
func resolveCommit(repo string, c Case) error {
	if hasCommit(repo, c.ReviewedSHA) {
		return nil
	}
	tried := []string{c.ReviewedSHA}
	if c.FetchRef != "" {
		tried = append(tried, c.FetchRef)
	}
	var errs []error
	for _, src := range tried {
		out, err := gitIn(repo, "fetch", "--quiet", "--no-tags", "--no-write-fetch-head", "origin", src)
		if err != nil {
			errs = append(errs, fmt.Errorf("fetch origin %s: %w: %s", src, err, strings.TrimSpace(out)))
			continue
		}
		if hasCommit(repo, c.ReviewedSHA) {
			return nil
		}
		errs = append(errs, fmt.Errorf("fetch origin %s: the commit is not on it", src))
	}
	return fmt.Errorf("case %s: reviewed_sha %s cannot be resolved in %s, so the case is not replayed and not scored\n%w",
		c.ID, c.ReviewedSHA, repo, errors.Join(errs...))
}

func hasCommit(repo, sha string) bool {
	_, err := gitIn(repo, "cat-file", "-e", sha+"^{commit}")
	return err == nil
}

// outsideRepo refuses a run directory inside repo's tree: a replay worktree
// there would be seen by the repository it replays.
func outsideRepo(repo, runDir string) error {
	if !filepath.IsAbs(runDir) {
		return fmt.Errorf("the run directory %q must be absolute", runDir)
	}
	top, err := gitIn(repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("%s is not a git repository: %w", repo, err)
	}
	realTop, realRun := spec.ResolveExisting(strings.TrimSpace(top)), spec.ResolveExisting(runDir)
	if rel, err := filepath.Rel(realTop, realRun); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("the run directory %s is inside %s; a replay runs outside every repository tree", runDir, top)
	}
	return nil
}

func isSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

func gitIn(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return string(out), err
}
