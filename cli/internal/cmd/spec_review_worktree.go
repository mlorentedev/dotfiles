package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// reviewWorktreeMargin is how long a review worktree's lease outlives the
// run's deadline. Until the lease passes, sweep leaves the worktree alone, so
// the margin covers the copy-back and removal that follow the deadline.
const reviewWorktreeMargin = 30 * time.Minute

// reviewTree is the disposable worktree a reviewer runs in (HARNESS-142): a
// detached checkout of the reviewed commit, under the worktree root.
type reviewTree struct {
	repoRoot string // the main repository, which the worktree is added to
	name     string
	path     string
}

// planReviewTree resolves where the review worktree will live without making
// it, so --dry-run can print the launch. A seam: the command-level fixtures
// have no real repository, as for resolveReviewBase.
var planReviewTree = func(repoRoot, id string, slot spec.ReviewSlot) (reviewTree, error) {
	main, err := worktree.ResolveMainRepoRoot(repoRoot)
	if err != nil {
		return reviewTree{}, err
	}
	root := env.ResolvePath("WORKTREE_ROOT")
	if root == "" {
		return reviewTree{}, errors.New("WORKTREE_ROOT could not be resolved, so the review has no worktree to run in: run `dotf doctor`")
	}
	nonce := make([]byte, 3)
	if _, err := rand.Read(nonce); err != nil {
		return reviewTree{}, err
	}
	name := fmt.Sprintf("%s-%s-%s", id, slot.Name, hex.EncodeToString(nonce))
	return reviewTree{repoRoot: main, name: name, path: worktree.ResolveWorktreePath(root, filepath.Base(main), name)}, nil
}

// createReviewTree adds the planned worktree, detached at commit, with a
// review lease of ttl.
var createReviewTree = func(t reviewTree, commit string, ttl time.Duration) error {
	info, err := worktree.AddDetached(worktree.DetachedOptions{
		RepoRoot:     t.repoRoot,
		WorktreeRoot: env.ResolvePath("WORKTREE_ROOT"),
		Name:         t.name,
		Commit:       commit,
		TTL:          ttl,
		Creator:      "dotf spec review",
		Kind:         worktree.KindReview,
	})
	if err != nil {
		return fmt.Errorf("creating the review worktree: %w", err)
	}
	if info.Path != t.path {
		return fmt.Errorf("the review worktree was created at %s, not at the planned %s", info.Path, t.path)
	}
	return nil
}

// removeReviewTree force-removes a review worktree and prunes its record.
var removeReviewTree = func(path string) error {
	main, err := worktree.ResolveMainRepoRoot(path)
	if err != nil {
		return err
	}
	runner := &worktree.RealSweepRunner{}
	if err := runner.WorktreeRemoveForce(main, path); err != nil {
		return err
	}
	return runner.WorktreePrune(main)
}

// reviewTTL is the review worktree's lease: the deadline the reviewer runs
// under, plus the margin.
func reviewTTL(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		timeout = spec.DefaultReviewerTimeout
	}
	return timeout + reviewWorktreeMargin
}

// mirrorSpecFolder makes dst hold exactly the regular files under src: it
// copies each one over, then deletes the files dst holds that src does not. It
// carries the caller's uncommitted spec folder into the review worktree (AC6),
// so the reviewer reads the round files and the request the launch just wrote.
// The deletion matters as much as the copy: a verdict committed at HEAD is
// checked out into the worktree, and once the caller rotated it aside, keeping
// it would let a run that ends without a verdict copy the old one back.
func mirrorSpecFolder(src, dst string) error {
	if err := copySpecFolder(src, dst); err != nil {
		return err
	}
	return filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dst, p)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(filepath.Join(src, rel)); errors.Is(err, fs.ErrNotExist) {
			return os.Remove(p)
		} else if err != nil {
			return err
		}
		return nil
	})
}

// copySpecFolder copies the regular files under src into dst, keeping their
// layout and overwriting what dst holds.
func copySpecFolder(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// finishReviewTree runs when `spec deadline` ends, whatever ended it: it
// copies the review file back to the caller's spec folder, records that the
// output is saved, and removes the worktree. Nothing is removed when the copy
// fails, since the worktree then holds the only copy of the verdict; a failed
// removal is reported and left for `dotf worktree sweep`, which takes a review
// worktree whose output was saved once its lease passes.
func finishReviewTree(errw io.Writer, review, tree, callerSpec string) {
	if tree == "" || callerSpec == "" {
		return
	}
	if _, err := os.Stat(review); err == nil {
		if err := copyFile(review, filepath.Join(callerSpec, filepath.Base(review))); err != nil {
			_, _ = fmt.Fprintf(errw, "[review] could not copy %s back to %s, so the review worktree is kept: %v\n", review, callerSpec, err)
			return
		}
	}
	meta, err := worktree.LoadMetadata(tree)
	if err == nil && meta != nil {
		meta.OutputSaved = true
		err = worktree.SaveMetadata(tree, *meta)
	}
	if err != nil {
		_, _ = fmt.Fprintf(errw, "[review] could not record that the output of %s was copied back: %v\n", tree, err)
	}
	// Leave the worktree before removing it: Windows refuses to remove the
	// working directory of a running process, and the launch starts this one
	// in it.
	if cwd, err := os.Getwd(); err == nil && isWithin(cwd, tree) {
		_ = os.Chdir(callerSpec)
	}
	if err := removeReviewTree(tree); err != nil {
		_, _ = fmt.Fprintf(errw, "[review] could not remove the review worktree %s; `dotf worktree sweep` takes it once its lease passes: %v\n", tree, err)
	}
}

// finishLeftoverReviewTree finishes a foreground run's worktree when it is
// still there. `spec deadline` removes it however the run ends, but a run that
// never started never reached deadline. Finishing again keeps the worktree
// when the copy-back fails, as deadline does.
func finishLeftoverReviewTree(errw io.Writer, review, tree, callerSpec string) {
	if _, err := os.Stat(tree); err == nil {
		finishReviewTree(errw, review, tree, callerSpec)
	}
}

// isWithin reports whether p is dir or below it.
func isWithin(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
