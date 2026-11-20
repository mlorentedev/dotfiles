// Package pushlease force-updates the checked-out branch on its remote with an
// explicit lease. It builds git's arguments itself and runs git without a
// shell, so the agent gets the lease push without any deny rule being relaxed
// and without shell text being parsed (HARNESS-203, #3612).
package pushlease

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

// Refusal is a request refused before git push ran: the remote is untouched.
type Refusal struct {
	Reason string
}

func (r *Refusal) Error() string { return r.Reason }

func refuse(format string, args ...any) *Refusal {
	return &Refusal{Reason: fmt.Sprintf(format, args...)}
}

// Prepare resolves the request against the repository.
func Prepare(req Request) (Plan, error) {
	branch, err := gitOutput(req.Dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return Plan{}, fmt.Errorf("read the checked-out branch: %w", err)
	}
	head, err := gitOutput(req.Dir, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return Plan{}, fmt.Errorf("read HEAD: %w", err)
	}
	remote := req.Remote
	if remote == "" {
		remote = "origin"
	}
	expect := strings.ToLower(req.Expect)
	// git checks the lease only for a ref it is about to change: a remote that
	// already equals HEAD is "up to date" and exits 0 whatever the lease says.
	// Comparing first makes a wrong --expect a refusal in that case too; the
	// lease still covers a push landing between this read and ours.
	current, err := remoteBranchSHA(req.Dir, remote, branch)
	if err != nil {
		return Plan{}, err
	}
	if current != expect {
		if current == "" {
			return Plan{}, refuse("stale lease: %s has no branch %s, --expect is %s", remote, branch, expect)
		}
		return Plan{}, refuse("stale lease: %s/%s is %s, --expect is %s", remote, branch, current, expect)
	}
	return Plan{Dir: req.Dir, Branch: branch, Remote: remote, Expect: expect, New: head}, nil
}

// remoteBranchSHA asks the remote itself for the branch, never the
// remote-tracking ref. It returns "" when the remote has no such branch.
func remoteBranchSHA(dir, remote, branch string) (string, error) {
	ref := "refs/heads/" + branch
	out, err := gitOutput(dir, "ls-remote", remote, ref)
	if err != nil {
		return "", fmt.Errorf("read %s on %s: %w", ref, remote, err)
	}
	// ls-remote matches patterns by their tail, so take the exact ref only.
	for line := range strings.SplitSeq(out, "\n") {
		sha, name, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if ok && name == ref {
			return strings.ToLower(sha), nil
		}
	}
	return "", nil
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
	cmd.Env = gitenv.WithoutRepository(os.Environ())
	cmd.Stdout = errOut
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push %s to %s: %w", p.Branch, p.Remote, err)
	}
	_, err := fmt.Fprintf(out, "%s %s..%s\n", p.Branch, p.Expect, p.New)
	return err
}

// gitOutput runs git in dir with the repository variables removed, so an
// inherited GIT_DIR cannot choose which repository is read.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec // fixed git subcommands, no shell
	cmd.Env = gitenv.WithoutRepository(os.Environ())
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
