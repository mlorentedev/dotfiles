package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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
		Short:        "Keep a repository's lessons in one format and render their index",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(newLessonsFmtCmd(), newLessonsIndexCmd(), newLessonsRenumberCmd())
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
				c.PrintErrln("lessons fmt:", err)
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

func newLessonsRenumberCmd() *cobra.Command {
	var (
		next bool
		to   int
		dir  string
	)
	c := &cobra.Command{
		Use:   "renumber <file> (--next | --to N)",
		Short: "Move a lesson to another number and rewrite its id and the links to it",
		Long: "renumber moves one lesson to a new number: --next takes one more than the\n" +
			"highest number under the lessons directory (never a gap: a retired number may\n" +
			"still be cited elsewhere), --to N takes N. A number another lesson holds is\n" +
			"refused. The lesson's id follows the file name, and every wikilink in the tree\n" +
			"that names it, as [[lesson-N]] or [[lesson-N-slug]], is rewritten.\n\n" +
			"It works in a tree where two lessons share a number: the state a branch is in\n" +
			"after main is merged and `dotf pr land` refused its lesson number. A link that\n" +
			"names the old number and so both lessons is left alone and printed, because it\n" +
			"still resolves and a wrong guess would pass `lessons fmt --check`.\n\n" +
			"A tracked file moves with git mv; the rewritten links are left for you to\n" +
			"stage. Run again after it moved, it reports the new name and changes nothing.",
		Example:       "  dotf lessons renumber lesson-497-my-slug.md --next\n  dotf lessons renumber lesson-497-my-slug.md --to 520",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			if next == c.Flags().Changed("to") || (!next && to < 1) {
				return errors.New("lessons renumber: give exactly one of --next or --to N (N >= 1)")
			}
			d, err := lessonsDir("renumber", dir)
			if err != nil {
				return err
			}
			d, file, err := canonicalPaths(d, args[0])
			if err != nil {
				return err
			}
			res, err := lessons.Renumber(d, file, to, gitMove)
			if err != nil {
				c.PrintErrln("lessons renumber:", err)
				return err
			}
			reportRenumber(c, res)
			return nil
		},
	}
	c.Flags().BoolVar(&next, "next", false, "move to one more than the highest lesson number")
	c.Flags().IntVar(&to, "to", 0, "move to this number")
	c.Flags().StringVar(&dir, "dir", "", "lessons directory (default: <repo>/docs/lessons)")
	return c
}

// canonicalPaths resolves symlinks in the lessons directory and in the file's
// directory, so a logical path (macOS /var) and a physical one (/private/var,
// what git answers) name the same lesson. The file itself may already be gone.
func canonicalPaths(dir, file string) (string, string, error) {
	d, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", "", err
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", "", err
	}
	return d, filepath.Join(parent, filepath.Base(abs)), nil
}

// gitMove renames with git mv when git tracks the file, so the rename is staged
// as one; an untracked file is renamed in place.
func gitMove(from, to string) error {
	wd := filepath.Dir(from)
	if exec.Command("git", "-C", wd, "ls-files", "--error-unmatch", "--", filepath.Base(from)).Run() != nil {
		return os.Rename(from, to)
	}
	if out, err := exec.Command("git", "-C", wd, "mv", "--", filepath.Base(from), filepath.Base(to)).CombinedOutput(); err != nil {
		return fmt.Errorf("git mv: %w: %s", err, out)
	}
	return nil
}

func reportRenumber(c *cobra.Command, res lessons.Renumbered) {
	w := c.OutOrStdout()
	if res.Already {
		_, _ = fmt.Fprintf(w, "already renumbered: %s\n", res.To)
		return
	}
	_, _ = fmt.Fprintf(w, "renumbered %s -> %s\n", res.From, res.To)
	for _, p := range res.Rewritten {
		_, _ = fmt.Fprintf(w, "rewrote links in %s\n", p)
	}
	for _, a := range res.Ambiguous {
		_, _ = fmt.Fprintf(w, "left alone, names two lessons: %s\n", a)
	}
	_, _ = fmt.Fprintln(w, "then: review, git add the lessons directory, dotf lessons fmt --check")
}
