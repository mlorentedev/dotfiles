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
	c.AddCommand(newLessonsFmtCmd())
	return c
}

func newLessonsFmtCmd() *cobra.Command {
	var (
		check bool
		dir   string
	)
	c := &cobra.Command{
		Use:   "fmt",
		Short: "Normalise every lesson and regenerate the index tables",
		Long: "fmt rewrites each lesson-NNN-<slug>.md under the lessons directory (and its\n" +
			"first-level category directories) into one form: frontmatter with id, type,\n" +
			"status, title and created, an H1 equal to the title, and the date in one\n" +
			"place. The body is left as written. It then regenerates each directory's\n" +
			"_index.md table from the files, between generated markers, keeping the\n" +
			"index's own prose.\n\n" +
			"The files are the source of truth. A merge conflict on an index is resolved\n" +
			"by re-running fmt. A lesson number used twice anywhere is an error.\n\n" +
			"--check writes nothing and exits 1 when any file would change, naming it.\n" +
			"A second fmt on a formatted tree changes nothing.",
		Example:       "  dotf lessons fmt\n  dotf lessons fmt --check",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			if dir == "" {
				root := env.RepoDir()
				if root == "" {
					return fmt.Errorf("lessons fmt: not inside a repository — pass --dir")
				}
				dir = filepath.Join(root, "docs", "lessons")
			}
			changes, err := lessons.Plan(dir)
			if err != nil {
				c.PrintErrln("lessons fmt:", err)
				return err
			}
			w := c.OutOrStdout()
			if check {
				for _, ch := range changes {
					_, _ = fmt.Fprintf(w, "not formatted: %s\n", ch.Path)
				}
				if len(changes) > 0 {
					_, _ = fmt.Fprintln(w, "run: dotf lessons fmt")
					return errLessonsDrift
				}
				_, _ = fmt.Fprintln(w, "[OK] lessons and indexes are in their canonical form")
				return nil
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
