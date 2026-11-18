package docpaths

import (
	"testing"
)

// The glob contract, where it differs from the retired shell guard (CLI-141).
// The guard expanded `*` only and let the shell decide; this scanner reads a
// token as a pattern when it holds any of `*`, `?` or `[`, with the meaning
// path/filepath.Match gives them, and asks only whether something matches:
// a trailing slash does not narrow the match to directories.
func TestCheckFilesGlobContract(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "scripts/utils.sh", "ok\n")
	writeRepoFile(t, repo, "scripts/a1.sh", "ok\n")
	policy := Policy{Version: 1, GovernedBasenames: []string{"AGENTS.md"}, KnownExtensions: []string{".sh"}}

	for _, tc := range []struct {
		name, token, want string
	}{
		{"a question mark matches one character", "scripts/a?.sh", ""},
		{"a question mark that matches nothing", "scripts/?.sh", "empty-glob:scripts/?.sh"},
		{"a class matches one of its characters", "scripts/[au]1.sh", ""},
		{"a class that matches nothing", "scripts/[xy]1.sh", "empty-glob:scripts/[xy]1.sh"},
		{"a trailing slash still matches a regular file", "scripts/*.sh/", ""},
		{"a trailing slash on a pattern that matches nothing", "scripts/*.zsh/", "empty-glob:scripts/*.zsh/"},
		{"a pattern below a regular file matches nothing", "scripts/utils.sh/*", "empty-glob:scripts/utils.sh/*"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeRepoFile(t, repo, "AGENTS.md", "See `"+tc.token+"`.\n")
			report, err := CheckFiles(CheckOptions{RepoRoot: repo, Policy: policy}, []string{"AGENTS.md"})
			if err != nil {
				t.Fatal(err)
			}
			got := findingKeys(report.Findings)
			if tc.want == "" && len(got) != 0 || tc.want != "" && (len(got) != 1 || !got[tc.want]) {
				t.Errorf("findings = %+v, want %q", report.Findings, tc.want)
			}
		})
	}
}
