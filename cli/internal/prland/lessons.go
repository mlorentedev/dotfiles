package prland

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/lessons"
)

// lessonsRoot is where `dotf lessons` keeps a repository's lessons. The root
// and its first-level category directories share one numbering.
const lessonsRoot = "docs/lessons/"

// lessonNumber returns the number of the lesson at path, when path is a
// lesson file in lessonsRoot or in one of its category directories.
func lessonNumber(path string) (int, bool) {
	rel, ok := strings.CutPrefix(path, lessonsRoot)
	if !ok || strings.Count(rel, "/") > 1 {
		return 0, false
	}
	return lessons.FileNumber(rel[strings.LastIndex(rel, "/")+1:])
}

// bringsLesson reports whether the PR leaves a lesson file at path, which
// then holds its number on the base.
func (f Facts) bringsLesson(path string) bool {
	_, ok := lessonNumber(path)
	return ok && !slices.Contains(f.Freed, path)
}

// lessonReasons names each lesson the PR brings whose number another file
// already holds on the base (#2752). Two PRs that each took the next free
// number pass CI against the base they were checked on, and collide only once
// both have merged, which turns the base red for everyone.
func lessonReasons(f Facts) []string {
	var r []string
	for _, file := range f.Files {
		if !f.bringsLesson(file) {
			continue
		}
		n, _ := lessonNumber(file)
		if held, taken := f.BaseLessons[n]; taken && held != file && !slices.Contains(f.Freed, held) {
			r = append(r, fmt.Sprintf("lesson %d is already %s on %s: renumber %s to the next free number", n, held, f.BaseRef, file))
		}
	}
	return r
}

// readBaseLessons maps each lesson number on base to its path, in one read of
// the git trees endpoint. A base without lessonsRoot holds no number; any
// other failure, a truncated listing included, is an error, because a number
// that could not be checked must not read as free.
func readBaseLessons(ctx context.Context, o Options, base string) (map[int]string, error) {
	out, err := o.Run(ctx, "api", "repos/"+apiRepo(o)+"/git/trees/"+base+":"+strings.TrimSuffix(lessonsRoot, "/")+"?recursive=1",
		"--jq", `if .truncated then "truncated" else .tree[] | select(.type == "blob") | .path end`)
	switch {
	case err == nil:
	case strings.Contains(err.Error(), "HTTP 404"):
		return map[int]string{}, nil
	default:
		return nil, fmt.Errorf("list the lessons on %s: %w", base, err)
	}
	taken := map[int]string{}
	for _, p := range strings.Fields(string(out)) {
		if p == "truncated" {
			return nil, fmt.Errorf("list the lessons on %s: the listing is truncated", base)
		}
		if n, ok := lessonNumber(lessonsRoot + p); ok {
			taken[n] = lessonsRoot + p
		}
	}
	return taken, nil
}
