package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/lessons"
)

// errLessonsDrift is returned by `lessons fmt --check` when a file is not in
// its canonical form; the paths are printed first.
var errLessonsDrift = errors.New("lessons fmt: files are not in their canonical form")

func newLessonsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "lessons",
		Short:        "Keep a repository's lessons in one format with a generated index",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(newLessonsFmtCmd(), newLessonsIndexCmd())
	return c
}

// lessonsDir resolves the --dir flag: the flag, else <repo>/docs/lessons.
func lessonsDir(verb, dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	root := env.RepoDir()
	if root == "" {
		return "", fmt.Errorf("lessons %s: not inside a repository — pass --dir", verb)
	}
	return filepath.Join(root, "docs", "lessons"), nil
}

func newLessonsIndexCmd() *cobra.Command {
	var dir string
	c := &cobra.Command{
		Use:   "index",
		Short: "Print the lessons index, rendered from the lesson files",
		Long: "index renders the index of the lessons under the lessons directory to stdout:\n" +
			"one table of number, title and date for the lessons in it, then one section per\n" +
			"category directory. It writes no file. The index is not committed: two PRs\n" +
			"that each add a lesson would conflict on it, and under a merge queue a\n" +
			"conflict ejects the PR.",
		Example:       "  dotf lessons index\n  dotf lessons index | grep -i worktree",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			d, err := lessonsDir("index", dir)
			if err != nil {
				return err
			}
			out, err := lessons.Index(d)
			if err != nil {
				c.PrintErrln("lessons index:", err)
				return err
			}
			_, err = c.OutOrStdout().Write(out)
			return err
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "lessons directory (default: <repo>/docs/lessons)")
	return c
}

func newLessonsFmtCmd() *cobra.Command {
	var (
		check bool
		dir   string
	)
	c := &cobra.Command{
		Use:   "fmt",
		Short: "Normalise every lesson file into one format",
		Long: "fmt rewrites each lesson-NNN-<slug>.md under the lessons directory (and its\n" +
			"first-level category directories) into one form: frontmatter with id, type,\n" +
			"status, title and created, an H1 equal to the title, and the date in one\n" +
			"place. The body is left as written.\n\n" +
			"The files are the source of truth. A lesson number used twice anywhere is an\n" +
			"error, and so is a wikilink that names no lesson. fmt neither reads nor\n" +
			"writes an _index.md: the index is rendered on read by `dotf lessons index`.\n\n" +
			"--check writes nothing and exits 1 when any file would change, naming it, or\n" +
			"when an _index.md is still committed under the lessons directory (a table\n" +
			"nothing regenerates goes stale in silence). fmt leaves such a file alone.\n" +
			"A second fmt on a formatted tree changes nothing.",
		Example:       "  dotf lessons fmt\n  dotf lessons fmt --check",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			d, err := lessonsDir("fmt", dir)
			if err != nil {
				return err
			}
			changes, err := lessons.Plan(d)
			if err != nil {
				c.PrintErrln("lessons fmt:", err)
				return err
			}
			stray, err := lessons.StrayIndexes(d)
			if err != nil {
				return err
			}
			w := c.OutOrStdout()
			if check {
				for _, ch := range changes {
					_, _ = fmt.Fprintf(w, "not formatted: %s\n", ch.Path)
				}
				for _, p := range stray {
					_, _ = fmt.Fprintf(w, "committed index: %s\n", p)
				}
				if len(changes) > 0 {
					_, _ = fmt.Fprintln(w, "run: dotf lessons fmt")
				}
				if len(stray) > 0 {
					_, _ = fmt.Fprintln(w, "run: git rm the index; render it with `dotf lessons index`")
				}
				if len(changes) > 0 || len(stray) > 0 {
					return errLessonsDrift
				}
				_, _ = fmt.Fprintln(w, "[OK] lessons are in their canonical form")
				return nil
			}
			for _, p := range stray {
				_, _ = fmt.Fprintf(w, "left alone: %s (a committed index goes stale; git rm it and use `dotf lessons index`)\n", p)
			}
			if err := lessons.Apply(changes); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "formatted %d file(s)\n", len(changes))
			return nil
		},
	}
	c.Flags().BoolVar(&check, "check", false, "write nothing; exit 1 when any file is not in its canonical form")
	c.Flags().StringVar(&dir, "dir", "", "lessons directory (default: <repo>/docs/lessons)")
	return c
}
