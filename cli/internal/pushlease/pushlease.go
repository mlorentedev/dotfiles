// Package pushlease force-updates the checked-out branch on its remote with an
// explicit lease. It builds git's arguments itself and runs git without a
// shell, so the agent gets the lease push without any deny rule being relaxed
// and without shell text being parsed (HARNESS-203, #3612).
package pushlease

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/gitenv"
)

// Request is what the caller asks for.
type Request struct {
	// Dir is the working tree whose checked-out branch is pushed.
	Dir string
	// Remote is the remote to push to; empty resolves it from the branch.
	Remote string
	// Expect is the remote sha the caller read. It is never derived here: a
	// lease taken from the remote-tracking ref holds after any fetch, which
	// makes it a plain force.
	Expect string
}

// Plan is a request that passed every refusal, with what it resolved to.
type Plan struct {
	Dir    string
	Branch string
	Remote string
	Expect string
	// New is the local HEAD, read once, so the push and the line printed
	// after it name the same commit.
	New string
}

// Refusal codes: stable tags for callers and tests; Reason is the prose.
const (
	CodeDetachedHead      = "detached-head"
	CodeProtectedBranch   = "protected-branch"
	CodeBadExpect         = "bad-expect"
	CodeUnknownRemote     = "unknown-remote"
	CodeRemoteUnreadable  = "remote-unreadable"
	CodeDefaultBranch     = "default-branch"
	CodeDefaultUnresolved = "default-unresolved"
	CodeStaleLease        = "stale-lease"
	CodeSplitRemote       = "split-remote"
)

// Refusal is a request refused before git push ran: the remote is untouched.
type Refusal struct {
	Code   string
	Reason string
}

func (r *Refusal) Error() string { return r.Reason }

func refuse(code, format string, args ...any) *Refusal {
	return &Refusal{Code: code, Reason: fmt.Sprintf(format, args...)}
}

// fullSHA is a whole object name, SHA-1 or SHA-256. An abbreviation or a ref
// name would let git resolve the lease itself.
var fullSHA = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// Prepare resolves the request and refuses it, before any push, when it is not
// a lease push of a feature branch to its own name. Local checks run first, so
// a request refused on its face never reaches the network.
func Prepare(req Request) (Plan, error) {
	branch, err := checkedOutBranch(req.Dir)
	if err != nil {
		return Plan{}, err
	}
	if strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master") {
		return Plan{}, refuse(CodeProtectedBranch, "%s is never force-updated", branch)
	}
	expect := strings.ToLower(req.Expect)
	if !fullSHA.MatchString(expect) {
		return Plan{}, refuse(CodeBadExpect, "--expect must be the full sha the remote branch holds (40 or 64 hex), got %q", req.Expect)
	}
	remote, err := resolveRemote(req.Dir, req.Remote, branch)
	if err != nil {
		return Plan{}, err
	}
	if err := checkRemote(req.Dir, remote, branch, expect); err != nil {
		return Plan{}, err
	}
	head, err := gitOutput(req.Dir, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return Plan{}, fmt.Errorf("read HEAD: %w", err)
	}
	return Plan{Dir: req.Dir, Branch: branch, Remote: remote, Expect: expect, New: head}, nil
}

// checkedOutBranch is the branch HEAD names. symbolic-ref exits 1, and only 1,
// when HEAD is detached.
func checkedOutBranch(dir string) (string, error) {
	ref, err := gitOutput(dir, "symbolic-ref", "--quiet", "HEAD")
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return "", refuse(CodeDetachedHead, "HEAD is detached: check out the branch to push")
	}
	if err != nil {
		return "", fmt.Errorf("read the checked-out branch: %w", err)
	}
	branch, ok := strings.CutPrefix(ref, "refs/heads/")
	if !ok || branch == "" {
		return "", refuse(CodeDetachedHead, "HEAD names %s, not a branch", ref)
	}
	return branch, nil
}

// resolveRemote is --remote, else the branch's push remote, else its upstream
// remote, else origin, the order git push itself uses. The result must be a
// configured remote: a URL or a path is refused.
func resolveRemote(dir, explicit, branch string) (string, error) {
	remote := explicit
	for _, key := range []string{"branch." + branch + ".pushRemote", "remote.pushDefault", "branch." + branch + ".remote"} {
		if remote != "" {
			break
		}
		remote, _ = gitOutput(dir, "config", "--get", key)
	}
	if remote == "" {
		remote = "origin"
	}
	if strings.HasPrefix(remote, "-") {
		return "", refuse(CodeUnknownRemote, "%q is not a remote name", remote)
	}
	fetch, err := gitOutput(dir, "remote", "get-url", "--all", remote)
	if err != nil {
		return "", refuse(CodeUnknownRemote, "%q is not a configured remote: %v", remote, err)
	}
	// The checks read one repository and the push must write only that one.
	// ls-remote reads a remote's first URL while git pushes to every URL it
	// has, so a second remote.<r>.url is a split too, though its fetch and push
	// lists are equal. A push URL that differs, by pushurl or by
	// url.<base>.pushInsteadOf (get-url expands both), is the other way the
	// checks describe one repository while the push writes another. Reasons
	// name the remote, never a URL, which may carry credentials.
	if strings.Contains(fetch, "\n") {
		return "", refuse(CodeSplitRemote, "%s has more than one URL: its default branch and lease are read from the first, and the push would write to every one", remote)
	}
	if push, err := gitOutput(dir, "remote", "get-url", "--push", "--all", remote); err != nil || push != fetch {
		return "", refuse(CodeSplitRemote, "%s pushes to a different URL than it fetches from, so its default branch and lease cannot be checked against the push target", remote)
	}
	return remote, nil
}

// checkRemote reads the remote once and refuses the push when the branch is
// its default or the lease is stale.
func checkRemote(dir, remote, branch, expect string) error {
	view, err := readRemote(dir, remote, branch)
	if err != nil {
		return err
	}
	if err := checkDefault(dir, remote, branch, view.defaultBranch); err != nil {
		return err
	}
	// git checks the lease only for a ref it is about to change: a remote that
	// already equals HEAD is "up to date" and exits 0 whatever the lease says.
	// Comparing first makes a wrong --expect a refusal in that case too; the
	// lease still covers a push landing between this read and ours.
	if view.branchSHA == expect {
		return nil
	}
	current := view.branchSHA
	if current == "" {
		current = "absent"
	}
	return refuse(CodeStaleLease, "stale lease: %s/%s is %s, --expect is %s", remote, branch, current, expect)
}

// remoteView is what the remote itself reports, never the remote-tracking refs.
type remoteView struct {
	defaultBranch string // "" when the remote reports no HEAD branch
	branchSHA     string // "" when the remote has no such branch
}

// readRemote asks the remote, in one call, for its HEAD and for the branch.
func readRemote(dir, remote, branch string) (remoteView, error) {
	ref := "refs/heads/" + branch
	out, err := gitOutput(dir, "ls-remote", "--symref", remote, "HEAD", ref)
	if err != nil {
		return remoteView{}, refuse(CodeRemoteUnreadable, "cannot read %s: %v", remote, err)
	}
	var view remoteView
	// ls-remote matches patterns by their tail, so only exact names count.
	for line := range strings.SplitSeq(out, "\n") {
		left, name, ok := strings.Cut(strings.TrimSpace(line), "\t")
		switch {
		case !ok:
		case name == "HEAD" && strings.HasPrefix(left, "ref: refs/heads/"):
			view.defaultBranch = strings.TrimPrefix(left, "ref: refs/heads/")
		case name == ref:
			view.branchSHA = strings.ToLower(left)
		}
	}
	return view, nil
}

// checkDefault refuses the branch when it is the remote's default by either
// account: what the remote reports now, or what refs/remotes/<remote>/HEAD
// recorded. Either alone may be stale or missing; when both are, the default
// is unknown and that is a refusal too.
func checkDefault(dir, remote, branch, reported string) error {
	recorded := ""
	if ref, err := gitOutput(dir, "symbolic-ref", "--quiet", "refs/remotes/"+remote+"/HEAD"); err == nil {
		recorded = strings.TrimPrefix(ref, "refs/remotes/"+remote+"/")
	}
	if reported == "" && recorded == "" {
		return refuse(CodeDefaultUnresolved, "cannot tell the default branch of %s: it reports none and %s/HEAD is unset", remote, remote)
	}
	for _, def := range []string{reported, recorded} {
		if def != "" && strings.EqualFold(def, branch) {
			return refuse(CodeDefaultBranch, "%s is the default branch of %s", branch, remote)
		}
	}
	return nil
}

// pushArgs is the whole git command line: one refspec, the branch to its own
// name, and one lease on that same ref.
func (p Plan) pushArgs() []string {
	ref := "refs/heads/" + p.Branch
	return []string{"-C", p.Dir, "push", "--force-with-lease=" + ref + ":" + p.Expect, p.Remote, p.New + ":" + ref}
}

// Push runs the planned push. git's own output goes to errOut; on success out
// gets one line, "<branch> <old>..<new>".
func Push(p Plan, out, errOut io.Writer) error {
	cmd := exec.Command("git", p.pushArgs()...) //nolint:gosec // argv built by pushArgs from a Plan, no shell
	cmd.Env = gitEnv()
	cmd.Stdout = errOut
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push %s to %s: %w", p.Branch, p.Remote, err)
	}
	// The stale-lease check has already proved the remote held Expect, so an
	// Expect equal to New is a push git found up to date: say so, or the line
	// reads as evidence of a push that did not happen.
	suffix := ""
	if p.Expect == p.New {
		suffix = " (already there, nothing pushed)"
	}
	_, err := fmt.Fprintf(out, "%s %s..%s%s\n", p.Branch, p.Expect, p.New, suffix)
	return err
}

// gitEnv is the environment of every git child: the repository variables
// removed, so an inherited GIT_DIR cannot choose which repository is read, and
// no terminal prompt, so a remote asking for credentials fails instead of
// hanging an agent's turn (precedent: worktree.list).
func gitEnv() []string {
	return append(gitenv.WithoutRepository(os.Environ()), "GIT_TERMINAL_PROMPT=0")
}

// gitOutput runs git in dir with the repository variables removed, so an
// inherited GIT_DIR cannot choose which repository is read.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec // fixed git subcommands, no shell
	cmd.Env = gitEnv()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
