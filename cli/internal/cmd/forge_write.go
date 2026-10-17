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
		Short:        "Write to an issue on the forge the origin names",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	issue.AddCommand(newForgeCommentCmd("an issue or pull request"), newForgeIssueCreateCmd(), newForgeIssueEditCmd())
	return issue
}

func newForgePRCmd() *cobra.Command {
	pr := &cobra.Command{
		Use:          "pr",
		Short:        "Write to a pull request on the forge the origin names",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	pr.AddCommand(newForgeCommentCmd("a pull request"))
	return pr
}

// forgeTarget resolves the repository the cwd's origin names and the backend
// that writes to it; an undetectable forge is an error here, never a guess.
func forgeTarget() (forgewrite.Repo, forgewrite.Backend, error) {
	origin, err := forgeWriteOrigin()
	if err != nil {
		return forgewrite.Repo{}, nil, err
	}
	repo, err := forgewrite.ParseOrigin(origin)
	if err != nil {
		return forgewrite.Repo{}, nil, err
	}
	backend, err := forgeWriteBackend(repo)
	return repo, backend, err
}

func parseForgeNumber(arg string) (int, error) {
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("ERROR: %q is not an issue number. WHY: the argument is the number of the issue or pull request. FIX: pass a positive integer", arg)
	}
	return n, nil
}

// printForgeURL prints what the forge returned, or says plainly that the write
// went through without a URL. An empty URL is not a failure: a retry would
// write twice.
func printForgeURL(c *cobra.Command, url string) {
	if url == "" {
		url = "done (the forge returned no URL)"
	}
	_, _ = fmt.Fprintln(c.OutOrStdout(), url)
}

func newForgeCommentCmd(what string) *cobra.Command {
	var bodyFile string
	c := &cobra.Command{
		Use:   "comment <number> --body-file <file|->",
		Short: "Comment on " + what,
		Long: "comment posts the body of --body-file (or stdin with -) on " + what + "\n" +
			"<number> of the repository the cwd's origin names. A pull request is an issue on\n" +
			"both supported forges. The body is never an argument, so no prose reaches a\n" +
			"permission rule. There is no merge and no close verb.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			number, err := parseForgeNumber(args[0])
			if err != nil {
				return err
			}
			body, err := readForgeBody(bodyFile)
			if err != nil {
				return err
			}
			repo, backend, err := forgeTarget()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(c.Context(), forgeWriteTimeout)
			defer cancel()
			url, err := backend.CommentIssue(ctx, repo, number, body)
			if err != nil {
				return err
			}
			printForgeURL(c, url)
			return nil
		},
	}
	c.Flags().StringVar(&bodyFile, "body-file", "", "file holding the comment body, or - for stdin (required)")
	return c
}

func newForgeIssueCreateCmd() *cobra.Command {
	var title, bodyFile string
	var labels []string
	c := &cobra.Command{
		Use:   "create --title <title> --body-file <file|-> [--label <name>]...",
		Short: "Open an issue",
		Long: "create opens an issue in the repository the cwd's origin names. The body is read\n" +
			"only from --body-file (or stdin with -). The title is the caller's: dotf forge does\n" +
			"not compute an AREA-NNN id. Each --label must already exist on the forge. Board\n" +
			"fields are not set here.",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("ERROR: no title. WHY: an issue needs one. FIX: pass --title <text>")
			}
			body, err := readForgeBody(bodyFile)
			if err != nil {
				return err
			}
			repo, backend, err := forgeTarget()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(c.Context(), forgeWriteTimeout)
			defer cancel()
			url, err := backend.CreateIssue(ctx, repo, forgewrite.NewIssue{Title: title, Body: body, Labels: labels})
			if err != nil {
				return err
			}
			printForgeURL(c, url)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "issue title (required)")
	c.Flags().StringVar(&bodyFile, "body-file", "", "file holding the issue body, or - for stdin (required)")
	c.Flags().StringArrayVar(&labels, "label", nil, "label name; repeat for several")
	return c
}

func newForgeIssueEditCmd() *cobra.Command {
	var title, bodyFile string
	c := &cobra.Command{
		Use:   "edit <number> [--title <title>] [--body-file <file|->]",
		Short: "Change the title or body of an issue or pull request",
		Long: "edit replaces the title and/or body of issue or pull request <number>. A field\n" +
			"not named is left alone. It never changes state: there is no close verb.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			number, err := parseForgeNumber(args[0])
			if err != nil {
				return err
			}
			var edit forgewrite.IssueEdit
			if c.Flags().Changed("title") {
				if strings.TrimSpace(title) == "" {
					return fmt.Errorf("ERROR: the title is empty. WHY: an empty title is never what was meant. FIX: pass --title <text>")
				}
				edit.Title = &title
			}
			if c.Flags().Changed("body-file") {
				body, err := readForgeBody(bodyFile)
				if err != nil {
					return err
				}
				edit.Body = &body
			}
			if edit.Title == nil && edit.Body == nil {
				return fmt.Errorf("ERROR: nothing to change. WHY: edit needs a field to replace. FIX: pass --title and/or --body-file")
			}
			repo, backend, err := forgeTarget()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(c.Context(), forgeWriteTimeout)
			defer cancel()
			url, err := backend.EditIssue(ctx, repo, number, edit)
			if err != nil {
				return err
			}
			printForgeURL(c, url)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "new title")
	c.Flags().StringVar(&bodyFile, "body-file", "", "file holding the new body, or - for stdin")
	return c
}

func readForgeBody(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("ERROR: no body. WHY: the body is only read from a file so it never rides on the command line. FIX: pass --body-file <file>, or --body-file - for stdin")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(forgeWriteStdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read the body: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("ERROR: the body is empty. WHY: an empty comment is never what was meant. FIX: write the text to the file first")
	}
	return string(data), nil
}
