package initrepo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

// InitOptions configures the full `dotf init` scaffold. Root must be an existing
// absolute directory. VaultPath == "" skips the vault entry (the caller passes ""
// for --skip-vault or an absent vault).
type InitOptions struct {
	Root              string
	Stack             string // go | python | node | ts | none
	Date              string // YYYY-MM-DD
	DryRun            bool   // report what would change without modifying disk
	SkipAgents        bool
	SkipGithub        bool
	VaultPath         string // resolved vault root, or "" to skip
	ClaudeProjectsDir string // ~/.claude/projects, for the memory symlink
}

// InitStep is one orchestration step's outcome, for human-readable reporting.
type InitStep struct {
	Name   string
	Status string // "ok" | "skipped" | "warn"
	Detail string
}

// InitReport is the ordered log of what Init did.
type InitReport struct {
	Root  string
	Steps []InitStep
}

func (r *InitReport) add(name, status, detail string) {
	r.Steps = append(r.Steps, InitStep{Name: name, Status: status, Detail: detail})
}

// Init runs the full scaffold sequence under opts.Root. The two core steps —
// structure and (unless skipped) AGENTS.md — are fatal on error; every
// host-coupled step (CI by stack, stack init, git, pre-commit, vault, github)
// degrades to a "skipped"/"warn" entry in the report and never aborts the run,
// so `dotf init` always leaves a usable repo (ADR-022 C7).
func Init(opts InitOptions) (InitReport, error) {
	report := InitReport{Root: opts.Root}

	// 1. Structure + static files (core — fatal).
	sc, err := ScaffoldOpts(opts.Root, opts.DryRun)
	if err != nil {
		return report, fmt.Errorf("scaffold structure: %w", err)
	}
	report.add("structure", "ok", fmt.Sprintf("%d files, %d skipped", len(sc.Created), len(sc.Skipped)))

	// 2. AGENTS.md + SDD (core — fatal unless skipped).
	if opts.SkipAgents {
		report.add("agents", "skipped", "--skip-agents")
	} else {
		res, err := BootstrapAgentsOpts(opts.Root, false, opts.DryRun)
		if err != nil {
			return report, fmt.Errorf("bootstrap AGENTS.md: %w", err)
		}
		report.add("agents", "ok", "AGENTS.md "+res.Action)
	}

	// 3. CI by stack.
	if action, err := WriteCIOpts(opts.Root, opts.Stack, opts.DryRun); err != nil {
		report.add("ci", "warn", err.Error())
	} else {
		report.add("ci", okOrSkip(action != "created"), "ci.yml "+action)
	}

	// 4. Stack init (lightweight, offline).
	if opts.DryRun {
		report.add("stack", "ok", "stack="+opts.Stack+" (dry-run)")
	} else {
		stack, _ := StackInit(opts.Root, opts.Stack)
		report.add("stack", okOrSkip(len(stack.Actions) == 0), stackDetail(stack))
	}

	// 5. git init.
	if opts.DryRun {
		if _, err := os.Stat(filepath.Join(opts.Root, ".git")); err == nil {
			report.add("git", "skipped", "git exists")
		} else {
			report.add("git", "ok", "git would initialize")
		}
	} else {
		if action, err := GitInit(opts.Root); err != nil {
			report.add("git", "warn", err.Error())
		} else {
			report.add("git", okOrSkip(action != "initialized"), "git "+action)
		}
	}

	// 6. pre-commit install.
	if opts.DryRun {
		if _, err := exec.LookPath("pre-commit"); err == nil {
			report.add("pre-commit", "ok", "pre-commit would install")
		} else {
			report.add("pre-commit", "skipped", "pre-commit not found")
		}
	} else {
		if action, err := PreCommitInstall(opts.Root); err != nil {
			report.add("pre-commit", "warn", err.Error())
		} else {
			report.add("pre-commit", okOrSkip(action != "installed"), "pre-commit "+action)
		}
	}

	// 7. Vault entry (auto-skip when no vault).
	if opts.DryRun {
		if opts.VaultPath != "" {
			report.add("vault", "ok", "vault entry would be created")
		} else {
			report.add("vault", "skipped", "no vault path")
		}
	} else {
		vres, err := vault.WriteProjectEntry(vault.ProjectEntryOptions{
			VaultPath:         opts.VaultPath,
			RepoRoot:          opts.Root,
			Stack:             opts.Stack,
			Date:              opts.Date,
			ClaudeProjectsDir: opts.ClaudeProjectsDir,
		})
		if err != nil {
			report.add("vault", "warn", err.Error())
		} else if vres.Action == "skipped" {
			report.add("vault", "skipped", vres.Reason)
		} else {
			report.add("vault", "ok", vres.EntryDir)
		}
	}

	// 8. GitHub repo defaults (auto-skip without a remote / gh).
	if opts.SkipGithub {
		report.add("github", "skipped", "--skip-github")
	} else {
		status, detail := initGithubWithDryRun(opts.Root, opts.DryRun)
		report.add("github", status, detail)
	}

	return report, nil
}

func initGithubWithDryRun(root string, dryRun bool) (status, detail string) {
	repo, err := OriginRepo(root)
	if err != nil {
		return "skipped", "no origin remote yet (run `dotf init github` after adding one)"
	}
	res, err := ApplyDeleteBranchOnMerge(repo, dryRun)
	if err != nil {
		return "warn", err.Error()
	}
	if res.Action == "skipped" {
		return "skipped", res.Reason
	}
	return "ok", "delete_branch_on_merge " + res.Action + " on " + repo
}

func okOrSkip(skipped bool) string {
	if skipped {
		return "skipped"
	}
	return "ok"
}

func stackDetail(s StackResult) string {
	switch {
	case len(s.Actions) > 0:
		return "did: " + strings.Join(s.Actions, ", ")
	case len(s.Skipped) > 0:
		return "skipped: " + strings.Join(s.Skipped, ", ")
	default:
		return "stack=" + s.Stack
	}
}
