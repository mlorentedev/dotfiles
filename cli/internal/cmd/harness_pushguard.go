package cmd

import (
	"fmt"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// pushGuardGitTimeout bounds each git question. The hook itself is bounded at 5s
// by the binding; two questions must fit well inside it.
const pushGuardGitTimeout = 2 * time.Second

// newHarnessPushGuardCmd is the PreToolUse hook that refuses a forced git push
// the permission globs cannot judge (HARNESS-201).
//
// IT EXITS 0 OR 2 AND NOTHING ELSE, like `harness gate`: Claude treats any other
// status as a non-blocking error, so a crash would silently turn into "allowed"
// while looking like a failure nobody reads. Every path resolves to an explicit
// decision. An unreadable payload, a tool that is not a shell and a command that
// is not a force all answer 0: the guard never blocks what it does not understand
// to be a force, which keeps its normal state quiet.
//
// IT NEVER EMITS AN ALLOW. A verified lease push is merely not refused; whether it
// then prompts is the permission system's decision, not this hook's.
func newHarnessPushGuardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push-guard",
		Short: "Refuse a forced git push other than --force-with-lease=<branch>:<sha> on your own branch",
		Long: `push-guard reads a Claude PreToolUse payload on stdin and answers with an exit
code: 0 to let the call proceed to the normal permission flow, 2 to refuse it with
the reason on stderr.

It parses the shell command and refuses a push that forces (--force, -f, +<refspec>,
--mirror, a bare or sha-less --force-with-lease) and any push that forces onto the
remote's default branch or onto a branch other than the one checked out. The one
force it lets through is:

  git push <remote> --force-with-lease=<branch>:<full sha> <refspec>

on the checked-out branch, when that branch is not the remote's default branch. It
also refuses 'npm run land:push', which is the owner's landing mode.

Pushes that do not force are never touched.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPushGuard(cmd, newGitPushEnv)
		},
	}
}

// pushGuardPayload is the slice of a PreToolUse payload the guard reads. The
// field names are Claude's documented ones; `cwd` is in the base set every hook
// receives (lesson 264).
type pushGuardPayload struct {
	ToolName  string `json:"tool_name"`
	Cwd       string `json:"cwd"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func runPushGuard(cmd *cobra.Command, mkEnv func(cwd string) harness.PushEnv) error {
	raw, _ := io.ReadAll(cmd.InOrStdin())
	var p pushGuardPayload
	if json.Unmarshal(raw, &p) != nil {
		return nil
	}
	var shell harness.Shell
	switch strings.ToLower(p.ToolName) {
	case "bash":
		shell = harness.ShellPOSIX
	case "powershell":
		shell = harness.ShellPowerShell
	default:
		return nil
	}
	if strings.TrimSpace(p.ToolInput.Command) == "" {
		return nil
	}
	v := harness.GuardCommand(p.ToolInput.Command, shell, mkEnv(p.Cwd))
	if !v.Refused {
		return nil
	}
	msg := "push-guard: " + v.Reason
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), msg)
	return withExitCode(gateExitBlock, errors.New(msg))
}

// newGitPushEnv answers the guard's two questions by asking git.
func newGitPushEnv(cwd string) harness.PushEnv {
	return harness.PushEnv{
		Dir: cwd,
		CurrentBranch: func(dir string) string {
			return gitLine(dir, "symbolic-ref", "--short", "-q", "HEAD")
		},
		DefaultBranch: func(dir, remote string) string {
			if !safeRemoteName(remote) {
				return ""
			}
			ref := gitLine(dir, "symbolic-ref", "--short", "-q", "refs/remotes/"+remote+"/HEAD")
			return strings.TrimPrefix(ref, remote+"/")
		},
	}
}

// safeRemoteName accepts a remote's configured name, not an option or a URL.
var safeRemote = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func safeRemoteName(s string) bool { return safeRemote.MatchString(s) && !strings.Contains(s, "..") }

// gitLine runs `git -C dir args...` and returns its trimmed first line, "" on
// any failure: an unanswerable question refuses a lease push, never allows one.
func gitLine(dir string, args ...string) string {
	if dir == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), pushGuardGitTimeout)
	defer cancel()
	full := append([]string{"-C", nativeDir(runtime.GOOS, dir)}, args...)
	out, err := exec.CommandContext(ctx, "git", full...).Output() // #nosec G204 -- fixed git subcommands; dir and remote are validated or passed as one argv element
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(line)
}

var msysDrive = regexp.MustCompile(`^/(?:cygdrive/)?([A-Za-z])(/.*)?$`)

// nativeDir turns the directory spelling a POSIX shell on Windows uses
// (`/c/Users/u`, `/cygdrive/c/...`) into one a native git accepts. Subagents send
// `cd /c/Users/... && git push ...` and a native binary cannot resolve that path,
// which would refuse every lease push on a Windows box while Linux CI stayed green.
// Off Windows a path is a path and passes unchanged.
func nativeDir(goos, p string) string {
	if goos != "windows" {
		return p
	}
	p = strings.ReplaceAll(p, `\`, "/")
	if m := msysDrive.FindStringSubmatch(p); m != nil {
		rest := m[2]
		if rest == "" {
			rest = "/"
		}
		return strings.ToUpper(m[1]) + ":" + rest
	}
	return p
}
