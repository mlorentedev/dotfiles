package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/forgewrite"
)

// forgeWriteTimeout bounds one write, so a hung forge degrades to an error.
const forgeWriteTimeout = 30 * time.Second

// forgeWriteStdin is where `--body-file -` reads; a seam for tests.
var forgeWriteStdin io.Reader = os.Stdin

// forgeWriteOrigin returns the cwd repository's origin URL; a seam for tests.
var forgeWriteOrigin = func() (string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("ERROR: no origin remote here. WHY: dotf forge writes to the forge the cwd's origin names. FIX: run it from inside the repository")
	}
	return strings.TrimSpace(string(out)), nil
}

// forgeWriteBackend builds the backend for a repository; a seam for tests.
var forgeWriteBackend = func(r forgewrite.Repo) (forgewrite.Backend, error) {
	hosts, err := forgewrite.LoadHosts(env.ResolveForgeHostsPath())
	if err != nil {
		return nil, err
	}
	kind, host, err := forgewrite.Select(r, hosts)
	if err != nil {
		return nil, err
	}
	if kind == forgewrite.KindGitHub {
		return forgewrite.GitHub{Run: runGH}, nil
	}
	api := host.API
	if api == "" {
		api = "https://" + r.Host
	}
	return forgewrite.Gitea{BaseURL: api, Token: func() (string, error) { return forgeToken(host.Secret) }}, nil
}

func runGH(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, "gh", args...)
	c.Stdin = strings.NewReader(string(stdin))
	out, err := c.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
	}
	return out, err
}

// forgeToken reads a registry secret in-process. It never runs `dotf secrets
// show`, which prints, and never returns the value to anything but the backend.
func forgeToken(id string) (string, error) {
	reg, err := loadRegistry()
	if err != nil {
		return "", err
	}
	entry, err := reg.ShowEntry(id)
	if err != nil {
		return "", err
	}
	raw, err := secretLoader().RawResolve(entry)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func newForgeIssueCmd() *cobra.Command {
	issue := &cobra.Command{
		Use:          "issue",
		Short:        "Write to an issue or pull request on the forge origin names",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	issue.AddCommand(newForgeIssueCommentCmd())
	return issue
}

func newForgeIssueCommentCmd() *cobra.Command {
	var bodyFile string
	c := &cobra.Command{
		Use:   "comment <number> --body-file <file|->",
		Short: "Comment on an issue or pull request",
		Long: "comment posts the body of --body-file (or stdin with -) on issue or pull request\n" +
			"<number> of the repository the cwd's origin names. A pull request is an issue on\n" +
			"both supported forges. The body is never an argument, so no prose reaches a\n" +
			"permission rule. There is no merge and no close verb.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil || number < 1 {
				return fmt.Errorf("ERROR: %q is not an issue number. WHY: the argument is the number of the issue or pull request. FIX: pass a positive integer", args[0])
			}
			body, err := readForgeBody(bodyFile)
			if err != nil {
				return err
			}
			origin, err := forgeWriteOrigin()
			if err != nil {
				return err
			}
			repo, err := forgewrite.ParseOrigin(origin)
			if err != nil {
				return err
			}
			backend, err := forgeWriteBackend(repo)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(c.Context(), forgeWriteTimeout)
			defer cancel()
			url, err := backend.CommentIssue(ctx, repo, number, body)
			if err != nil {
				return err
			}
			if url == "" {
				url = "posted (the forge returned no comment URL)"
			}
			_, _ = fmt.Fprintln(c.OutOrStdout(), url)
			return nil
		},
	}
	c.Flags().StringVar(&bodyFile, "body-file", "", "file holding the comment body, or - for stdin (required)")
	return c
}

func readForgeBody(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("ERROR: no comment body. WHY: the body is only read from a file so it never rides on the command line. FIX: pass --body-file <file>, or --body-file - for stdin")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(forgeWriteStdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read the comment body: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("ERROR: the comment body is empty. WHY: an empty comment is never what was meant. FIX: write the text to the file first")
	}
	return string(data), nil
}
