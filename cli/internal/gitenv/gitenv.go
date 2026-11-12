// Package gitenv holds the one list of the variables that tell git which
// repository to read. git reads them ahead of -C, so a GIT_DIR inherited from a
// hook or a parent git points a `git -C <dir>` call at another repository
// (#3122, #3275). Callers that name their repository drop them first.
package gitenv

import (
	"slices"
	"strings"
)

// RepositoryVars are the variables `git rev-parse --local-env-vars` names,
// less the GIT_CONFIG* ones, which carry configuration rather than a location.
// GIT_INTERNAL_SUPER_PREFIX left git's list in later versions and is kept for
// older ones. Upper case and sorted; the tests pin both and compare the list
// with the installed git's.
var RepositoryVars = []string{
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_COMMON_DIR",
	"GIT_DIR",
	"GIT_GRAFT_FILE",
	"GIT_IMPLICIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_INTERNAL_SUPER_PREFIX",
	"GIT_NO_REPLACE_OBJECTS",
	"GIT_OBJECT_DIRECTORY",
	"GIT_PREFIX",
	"GIT_REPLACE_REF_BASE",
	"GIT_SHALLOW_FILE",
	"GIT_WORK_TREE",
}

// WithoutRepository returns environ without RepositoryVars. Keys compare
// upper-cased, because Windows environment names are case-insensitive.
func WithoutRepository(environ []string) []string {
	clean := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		if !slices.Contains(RepositoryVars, strings.ToUpper(key)) {
			clean = append(clean, entry)
		}
	}
	return clean
}
