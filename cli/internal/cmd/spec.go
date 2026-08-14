package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/initrepo"
	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// now is the clock used to stamp created: in scaffolded specs. It is a package
// var so tests can pin it deterministically.
var now = func() time.Time { return time.Now().UTC() }

func newSpecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Spec-driven development scaffolding (ADR-020)",
		Long: "spec scaffolds and manages per-feature SDD spec folders.\n" +
			"The Go twin of scripts/init-spec.sh + scripts/archive-spec.sh; templates\n" +
			"are embedded, so init works without the vault checked out.",
	}
	cmd.AddCommand(newSpecInitCmd())
	cmd.AddCommand(newSpecReviewCmd())
	cmd.AddCommand(newSpecArchiveCmd())
	return cmd
}

// lookPath is a seam so tests can decide whether tmux "exists" without
// depending on the machine running them.
var lookPath = exec.LookPath

// runCommand is a seam for the launch itself, so the command's wiring is
// testable without spawning a reviewer that would take half an hour.
var runCommand = func(dir string, argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// runForeground runs the reviewer in this terminal, streaming its output to both
// the screen and the transcript.
//
// Deliberately NOT `sh -c '… | tee …'`, for two reasons that a shell pipeline
// cannot satisfy:
//
//  1. In POSIX sh the status of `cmd | tee f` is TEE's status, and tee almost
//     always succeeds — so a reviewer that crashes or hits its --print-timeout
//     would exit 0 and be reported as a completed review. dash has no
//     `set -o pipefail` to fix that, and the portable workaround is a
//     file-descriptor dance nobody should have to read.
//  2. `sh` is not on PATH on stock Windows, and this foreground path is exactly
//     the fallback the command documents for machines without tmux — i.e. it
//     would have failed on the platform it names, with a bare
//     `exec: "sh": executable file not found`.
//
// io.MultiWriter needs no shell, preserves the child's real exit status, and
// behaves identically everywhere.
var runForeground = func(dir string, argv []string, transcript string) error {
	f, err := os.Create(transcript)
	if err != nil {
		return fmt.Errorf("creating the transcript: %w", err)
	}

	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = dir
	c.Stdout = io.MultiWriter(os.Stdout, f)
	c.Stderr = io.MultiWriter(os.Stderr, f)
	runErr := c.Run()

	// Close is checked, not deferred-and-ignored: this file is being WRITTEN,
	// and a failed close means buffered output never reached the disk. Silently
	// dropping that would leave a truncated transcript looking complete — the
	// artifact whose entire job is to record what happened.
	//
	// The reviewer's own error wins when both fail: it is the outcome the caller
	// asked about, and a transcript problem is secondary to a failed review.
	if closeErr := f.Close(); closeErr != nil && runErr == nil {
		return fmt.Errorf("the review ran but its transcript could not be written: %w", closeErr)
	}
	return runErr
}

func newSpecReviewCmd() *cobra.Command {
	var (
		reviewer   string
		foreground bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "review <feature-id>",
		Short: "Run an adversarial review on a pooled model",
		Long: `Launch /adversarial-review for specs/<feature-id>/ on a model from
harness/reviewer-pool.json, and write its verdict to the spec folder.

The point is that the model is not the launcher's choice, nor the caller's habit:
it comes from the pool, and dotf spec archive refuses a review signed outside it.
The pool's first entry is the default; --reviewer selects any other member, which
is how the fallback gets exercised deliberately rather than only in an outage.

The model is always passed to the runner explicitly. Relying on a runner's own
default is how a review comes to be pinned by coincidence — pi defaults to the
google provider, and its configured default lives in unversioned per-machine
state.

By default the reviewer runs in a detached tmux session named review-<feature-id>
so the run can be watched while it happens; attach with the command printed on
launch. Without tmux (Windows, or a machine that lacks it) the run goes to the
foreground and says so. A machine-readable transcript is written beside the
review, because the verdict records what a reviewer concluded and the transcript
is the only record of how.`,
		Example:      "  dotf spec review HARNESS-071-reviewer-pool\n  dotf spec review AI-001-ollama-public --reviewer agy/gemini-3.1-pro-high",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := spec.ValidateID(id); err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoRoot, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}
			specDir := filepath.Join(repoRoot, "specs", id)
			if info, statErr := os.Stat(specDir); statErr != nil || !info.IsDir() {
				return fmt.Errorf("spec not found: %s", specDir)
			}

			entries, err := spec.LoadReviewerPoolEntries(repoRoot)
			if err != nil {
				return err
			}
			chosen, err := spec.ResolveReviewer(entries, reviewer)
			if err != nil {
				return err
			}

			skill := spec.ReviewerSkillPath(chosen.Runner)
			prompt := spec.ReviewPrompt(id, repoRoot, chosen.ID, skill)
			argv, err := spec.ReviewerCommand(chosen, prompt)
			if err != nil {
				return err
			}

			transcript := spec.TranscriptPath(repoRoot, id)
			session := spec.TmuxSession(id)
			useTmux := !foreground
			if useTmux {
				if _, lookErr := lookPath("tmux"); lookErr != nil {
					// Not a failure: tmux is Linux-only and dotf is
					// cross-platform. Degrade loudly rather than silently, so
					// nobody waits for a session that was never created.
					cmd.Printf("[INFO] tmux not found — running in the foreground; the session is not detachable\n")
					useTmux = false
				}
			}

			cmd.Printf("Reviewer:   %s (%s, %s)\n", chosen.ID, chosen.Runner, chosen.Role)
			cmd.Printf("Spec:       specs/%s\n", id)
			cmd.Printf("Transcript: %s\n", transcript)

			launch := argv
			if useTmux {
				launch = spec.TmuxWrap(session, repoRoot, argv, transcript)
			}

			if dryRun {
				// ShellJoin, not strings.Join: the tmux form's last element is a
				// whole pipeline, so joining raw elements prints a line that
				// cannot be pasted back into a shell.
				cmd.Printf("\n[DRY RUN] would run:\n  %s\n", spec.ShellJoin(launch))
				return nil
			}

			if useTmux {
				cmd.Printf("Session:    %s\n\n", session)
				if err := runCommand(repoRoot, launch); err != nil {
					return fmt.Errorf("starting the tmux session: %w", err)
				}
				cmd.Printf("[OK] Review running detached. Watch it with:\n\n    tmux attach -t %s\n\n", session)
				cmd.Printf("When it finishes, %s carries the verdict and archive reads it.\n", spec.ReviewFile)
				return nil
			}

			cmd.Printf("\n")
			return runForeground(repoRoot, launch, transcript)
		},
	}

	cmd.Flags().StringVar(&reviewer, "reviewer", "", "pool member to run (default: the pool's first entry)")
	cmd.Flags().BoolVar(&foreground, "foreground", false, "run in this terminal instead of a detached tmux session")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the command that would run, and exit")
	return cmd
}

func newSpecInitCmd() *cobra.Command {
	var (
		issueNum     int
		forceNoGate  bool
		bitacoraRepo string
	)

	cmd := &cobra.Command{
		Use:   "init <feature-id>",
		Short: "Scaffold a per-feature spec folder",
		Long: `Scaffold specs/<feature-id>/{proposal,tasks,verification}.md from the
embedded SDD templates.

Work-gate (ADR-018): the spec must be downstream of an OPEN GitHub issue.
Pass --issue <N>; the issue is verified via 'gh issue view' and its title is
recorded in the proposal's frontmatter (issue: owner/repo#N) and ## Why comment.
The issue's repo defaults to the current repo's origin; override with
--bitacora-repo owner/repo or $DOTF_BITACORA_REPO for a cross-repo work-gate.
Use --force-no-gate to scaffold without an issue (NOT RECOMMENDED).

Mechanical only: fill the proposal interactively afterwards ("/spec fill" in an
agent) or by hand. Do not skip the Why.`,
		Example:      "  dotf spec init AI-001-ollama-public --issue 42",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := spec.ValidateID(id); err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoRoot, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}

			var issueTitle, repoSlug string
			if !forceNoGate {
				if issueNum == 0 {
					return fmt.Errorf("no work-gate given: pass --issue <number>.\n" +
						"Per ADR-018 every spec is downstream of an OPEN GitHub issue on the\n" +
						"bitácora Project. Options: (a) open/find the issue, re-run with --issue;\n" +
						"(b) re-run with --force-no-gate (NOT RECOMMENDED)")
				}
				// Resolve the repo that hosts the work-gate issue (HARNESS-023):
				// explicit --bitacora-repo, then DOTF_BITACORA_REPO, else the
				// current repo's origin slug. Used for BOTH the gate check and the
				// proposal's issue: frontmatter, so a cross-repo gate (e.g. a
				// kubelab spec gated by a knowledge issue) resolves correctly
				// instead of silently checking the current repo.
				repoSlug = bitacoraRepo
				if repoSlug == "" {
					repoSlug = os.Getenv("DOTF_BITACORA_REPO")
				}
				if repoSlug == "" {
					if s, e := initrepo.OriginRepo(repoRoot); e == nil {
						repoSlug = s
					}
				}
				if repoSlug == "" {
					return fmt.Errorf("cannot determine the repo hosting issue #%d: no "+
						"--bitacora-repo, no DOTF_BITACORA_REPO, and no origin remote in %s.\n"+
						"Pass --bitacora-repo owner/repo (e.g. mlorentedev/knowledge) or set "+
						"DOTF_BITACORA_REPO", issueNum, repoRoot)
				}
				if !initrepo.ValidRepoSlug(repoSlug) {
					return fmt.Errorf("invalid bitácora repo %q: want owner/name", repoSlug)
				}
				issueTitle, err = spec.Gate(issueNum, repoSlug)
				if err != nil {
					return err
				}
				cmd.Printf("[INFO] Work-gate OK: %s#%d is open — %s\n", repoSlug, issueNum, issueTitle)
			}

			date := now().Format("2006-01-02")
			warning, err := spec.Scaffold(repoRoot, id, date, repoSlug, issueNum, issueTitle)
			if warning != "" {
				cmd.PrintErrf("[WARN] %s\n", warning)
			}
			if err != nil {
				return err
			}

			cmd.Printf("\n[OK] Created: specs/%s\n", id)
			cmd.Printf("     proposal.md, tasks.md, verification.md\n")
			if issueNum > 0 {
				cmd.Printf("     Work-gate linked: issue #%d\n", issueNum)
			}
			cmd.Printf("\nNext: fill proposal.md interactively (\"/spec fill %s\" in an agent)\n", id)
			cmd.Printf("      or edit by hand. Do not skip the Why.\n")
			return nil
		},
	}

	cmd.Flags().IntVar(&issueNum, "issue", 0, "GitHub issue number that gates this work (must exist and be OPEN)")
	cmd.Flags().StringVar(&bitacoraRepo, "bitacora-repo", "", "owner/repo hosting the work-gate issue (default: current repo's origin, or $DOTF_BITACORA_REPO)")
	cmd.Flags().BoolVar(&forceNoGate, "force-no-gate", false, "skip the open-issue work-gate (NOT RECOMMENDED — the gate is the SSOT)")
	return cmd
}

func newSpecArchiveCmd() *cobra.Command {
	var (
		prURL         string
		abandoned     bool
		forceDrafts   bool
		forceNoReview bool
	)

	cmd := &cobra.Command{
		Use:   "archive <feature-id>",
		Short: "Archive (or abandon) a per-feature spec folder",
		Long: `Move specs/<feature-id>/ into specs/archive/ (or specs/archive/_abandoned/
under --abandoned) and rewrite the proposal status to archived/abandoned.

Mechanical only — the Go twin of scripts/archive-spec.sh. A pre-flight refuses to
archive while unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags remain (override
with --force-with-drafts). A second pre-flight refuses without a fresh, passing
review.md from /adversarial-review (override with --force-without-review, or
declare "review: waived" with a reason in proposal.md). Vault promotion
(lessons/ADR/pattern) and any backlog tick stay interactive via "/spec archive"
in an agent.`,
		Example:      "  dotf spec archive AI-001-ollama-public --pr https://github.com/owner/repo/pull/42",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			repoRoot, err := spec.RepoRoot(cwd)
			if err != nil {
				return err
			}

			target, err := spec.Archive(repoRoot, id, spec.ArchiveOptions{
				Abandoned:          abandoned,
				ForceWithDrafts:    forceDrafts,
				ForceWithoutReview: forceNoReview,
				PRURL:              prURL,
				Date:               now().Format("2006-01-02"),
			})
			if err != nil {
				return err
			}

			rel, relErr := filepath.Rel(repoRoot, target)
			if relErr != nil {
				rel = target
			}
			status := "archived"
			if abandoned {
				status = "abandoned"
			}
			cmd.Printf("\n[OK] Archived: specs/%s -> %s\n", id, rel)
			cmd.Printf("     status: %s\n", status)
			if prURL != "" {
				cmd.Printf("     PR: %s\n", prURL)
			}
			cmd.Printf("\nVault promotion (lessons/ADR/pattern) and backlog tick must be done\n")
			cmd.Printf("separately (via \"/spec archive\" in an agent, or by hand).\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&prURL, "pr", "", "record this PR URL in proposal.md (informational)")
	cmd.Flags().BoolVar(&abandoned, "abandoned", false, "route to specs/archive/_abandoned/ and set status abandoned")
	cmd.Flags().BoolVar(&forceDrafts, "force-with-drafts", false, "archive even with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags")
	cmd.Flags().BoolVar(&forceNoReview, "force-without-review", false, "archive even without a fresh, passing review.md")
	return cmd
}
