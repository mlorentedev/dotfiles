package lessons

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Renumbered says what Renumber did.
type Renumbered struct {
	From, To  string   // the lesson's path before and after
	Rewritten []string // other lesson files whose links now name To
	Ambiguous []string // links left alone: they name the old number, held by another lesson too
	Already   bool     // From was gone and To already carries its slug: nothing was done
}

type entry struct {
	path   string
	number int
}

// Renumber moves the lesson at file to number to, or to the next free number
// when to is 0: one more than the highest number in the tree, never a gap,
// because a retired number may still be cited outside it. It rewrites the
// lesson's id and every wikilink in the tree that names it and no other lesson,
// in both forms ([[lesson-N]] and [[lesson-N-slug]]). A number another lesson
// holds is refused, so nothing is overwritten.
//
// It reads a tree in which two lessons share a number, which is the state that
// needs it (#2837): main merged into a branch whose lesson took a number the
// base then used. move does the rename (git mv in the command); nothing is
// written before it succeeds. When file is gone and a lesson with its slug sits
// in the same directory, the earlier run is reported and nothing changes.
func Renumber(dir, file string, to int, move func(from, to string) error) (Renumbered, error) {
	all, err := listAll(dir)
	if err != nil {
		return Renumbered{}, err
	}
	self, done, err := locate(dir, file, all)
	if err != nil || done != nil {
		return deref(done), err
	}
	if to, err = target(all, self, to); err != nil {
		return Renumbered{}, err
	}
	if to == all[self].number {
		return Renumbered{From: file, To: file, Already: true}, nil
	}
	digits, slug, _ := split(filepath.Base(file))
	dest := filepath.Join(filepath.Dir(file), fmt.Sprintf("lesson-%03d%s.md", to, slug))
	res := Renumbered{From: file, To: dest}
	rewrites, err := relink(dir, all, self, "lesson-"+digits, fmt.Sprintf("lesson-%03d", to), &res)
	if err != nil {
		return Renumbered{}, err
	}
	if err := move(file, dest); err != nil {
		return Renumbered{}, err
	}
	for p, body := range rewrites {
		if p == file {
			if _, body, err = Normalize(filepath.Base(dest), body); err != nil {
				return res, err
			}
			p = dest
		} else {
			res.Rewritten = append(res.Rewritten, p)
		}
		if err := os.WriteFile(p, body, 0o644); err != nil { //nolint:gosec // docs, world-readable by design
			return res, err
		}
	}
	sort.Strings(res.Rewritten)
	return res, nil
}

func deref(r *Renumbered) Renumbered {
	if r == nil {
		return Renumbered{}
	}
	return *r
}

// split returns the digits and the "-slug" of a lesson file name.
func split(name string) (digits, slug string, ok bool) {
	m := fileRE.FindStringSubmatch(name)
	if m == nil {
		return "", "", false
	}
	return m[1], strings.TrimPrefix(strings.TrimSuffix(name, ".md"), "lesson-"+m[1]), true
}

// locate finds file among the tree's lessons. A file that is gone while a
// lesson with its slug sits in the same directory is an earlier run: done
// reports it.
func locate(dir, file string, all []entry) (self int, done *Renumbered, err error) {
	_, slug, ok := split(filepath.Base(file))
	if !ok {
		return 0, nil, fmt.Errorf("%s: not a lesson-NNN-<slug>.md file", file)
	}
	for i, e := range all {
		if e.path == file {
			return i, nil, nil
		}
	}
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		return 0, nil, fmt.Errorf("%s: not a lesson under %s", file, dir)
	}
	for _, e := range all {
		if _, s, _ := split(filepath.Base(e.path)); s == slug && filepath.Dir(e.path) == filepath.Dir(file) {
			return 0, &Renumbered{From: file, To: e.path, Already: true}, nil
		}
	}
	return 0, nil, fmt.Errorf("%s: no such lesson", file)
}

// target resolves the number to move to: to, or one more than the highest
// number in the tree when to is 0. A number another lesson holds is refused.
func target(all []entry, self, to int) (int, error) {
	if to == 0 {
		for _, e := range all {
			to = max(to, e.number+1)
		}
	}
	for i, e := range all {
		if e.number == to && i != self {
			return 0, fmt.Errorf("lesson number %03d is taken by %s", to, e.path)
		}
	}
	return to, nil
}

// relink returns the new content of every lesson whose wikilinks name the
// lesson at all[self] and no other one, plus that lesson's own content. A link
// that also names another lesson is recorded in res.Ambiguous and left alone.
func relink(dir string, all []entry, self int, oldPrefix, newPrefix string, res *Renumbered) (map[string][]byte, error) {
	stems := make([]string, len(all))
	for i, e := range all {
		stems[i] = strings.TrimSuffix(filepath.Base(e.path), ".md")
	}
	names := func(stem, target string) bool { return stem == target || strings.HasPrefix(stem, target+"-") }
	rewrites := map[string][]byte{}
	for i, e := range all {
		raw, err := os.ReadFile(e.path) //nolint:gosec // the repository's own docs
		if err != nil {
			return nil, err
		}
		out := wikilink.ReplaceAllStringFunc(string(raw), func(link string) string {
			target := strings.TrimSpace(link[2:])
			if !names(stems[self], target) {
				return link
			}
			for j, s := range stems {
				if j != self && names(s, target) {
					rel, _ := filepath.Rel(dir, e.path)
					res.Ambiguous = append(res.Ambiguous, fmt.Sprintf("%s links [[%s]]", filepath.ToSlash(rel), target))
					return link
				}
			}
			return "[[" + newPrefix + strings.TrimPrefix(link[2:], oldPrefix)
		})
		if out != string(raw) || i == self {
			rewrites[e.path] = []byte(out)
		}
	}
	return rewrites, nil
}

// listAll lists every lesson file under dir and its first-level category
// directories. Unlike readDir it accepts a number used twice.
func listAll(dir string) ([]entry, error) {
	dirs, err := lessonDirs(dir)
	if err != nil {
		return nil, err
	}
	var all []entry
	for _, d := range dirs {
		es, err := os.ReadDir(d)
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			if n, ok := FileNumber(e.Name()); ok && !e.IsDir() {
				all = append(all, entry{path: filepath.Join(d, e.Name()), number: n})
			}
		}
	}
	return all, nil
}
