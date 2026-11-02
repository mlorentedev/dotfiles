package prreview

import (
	"crypto/sha1" //nolint:gosec // git's blob identity, not a security boundary
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// The real sources at the audited commit, held to the contract by
// TestUpstreamFixtures_AreTheAuditedBlobs. A hand-written handler kept the old
// shape of is_valid_file, and the tests passed while the step read no lockfile.
func upstreamFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/pr-agent/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func realConfig(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(repoFile(".pr_agent.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func noWarn(t *testing.T) warnFunc {
	return func(format string, a ...any) { t.Errorf("unexpected warning: "+format, a...) }
}

func realFilter(t *testing.T) *fileFilter {
	t.Helper()
	f, err := newFileFilter(realConfig(t), upstreamFixture(t, "language_extensions.toml"),
		upstreamFixture(t, "language_handler.py"), noWarn(t))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestUpstreamFixtures_AreTheAuditedBlobs(t *testing.T) {
	raw, err := os.ReadFile(repoFile("harness", "pr-agent-upstream-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Files map[string]string `json:"files"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	for path, name := range map[string]string{
		"pr_agent/algo/language_handler.py":          "language_handler.py",
		"pr_agent/settings/language_extensions.toml": "language_extensions.toml",
	} {
		content := upstreamFixture(t, name)
		h := sha1.New() //nolint:gosec // git's blob identity
		_, _ = fmt.Fprintf(h, "blob %d\x00", len(content))
		_, _ = h.Write(content)
		if got := hex.EncodeToString(h.Sum(nil)); got != contract.Files[path] {
			t.Errorf("%s: fixture blob %s, contract %q: re-audit and refresh both", path, got, contract.Files[path])
		}
	}
}

func TestFileFilter_ReadsEveryUpstreamListAtTheAuditedCommit(t *testing.T) {
	f := realFilter(t)
	if len(f.locks) != 14 || !f.locks["go.sum"] || !f.locks["bun.lockb"] {
		t.Errorf("lockfiles = %v, want upstream's 14", f.locks)
	}
	if len(f.suffixes) != 5 || f.suffixes[0] != ".min.js" {
		t.Errorf("suffixes = %v, want upstream's 5", f.suffixes)
	}
	if len(f.bad) != 56 || !f.bad["gitignore"] {
		t.Errorf("extensions: %d, want upstream's 56 with gitignore", len(f.bad))
	}
}

// The cases of tests/pr-agent-config.bats, against the real config and lists.
func TestFileFilter_Kept(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  int
	}{
		{"an escrow-only PR has nothing to read (#1618)",
			[]string{"sensitive/dr/bitwarden-export.age", "sensitive/dr/escrow-manifest.json"}, 0},
		{"one reviewable file is enough (#1618)",
			[]string{"sensitive/dr/escrow-manifest.json", "cli/internal/secrets/bw.go"}, 1},
		{"a transcript is ignored and its review is not",
			[]string{"specs/X-1/review-transcript.jsonl", "specs/X-1/review.md"}, 1},
		{"file types PR-Agent never reads (#1417)",
			[]string{".gitignore", "site/uv.lock", "go.sum", "assets/Logo.PNG", "web/app.min.js"}, 0},
		{"a readable file keeps the review (#1417)", []string{".gitignore", "scripts/utils.sh"}, 1},
		{"a lockfile is matched by its base name", []string{"a/b/package-lock.json"}, 0},
		{"a backslash separates the base name, as upstream replaces it", []string{`a\go.sum`}, 0},
		{"a lockfile's name inside another name is not one", []string{"notgo.sum"}, 1},
		{"the extension follows the last dot of the whole path", []string{"Makefile", "dir.png/tool"}, 2},
		{"a dotless name that is a bad extension is dropped, as upstream's split does", []string{"lock"}, 0},
		{"a suffix is the end of the name, not the middle", []string{"a.min.js.txt.go"}, 1},
		{"a path outside every glob is kept", []string{"xsensitive/a"}, 1},
	}
	f := realFilter(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := f.kept(c.files); got != c.want {
				t.Errorf("kept(%v) = %d, want %d", c.files, got, c.want)
			}
		})
	}
}

// A failed read of an upstream list narrows the skip, never widens it.
func TestFileFilter_AFailedUpstreamReadNarrows(t *testing.T) {
	ext, handler := upstreamFixture(t, "language_extensions.toml"), upstreamFixture(t, "language_handler.py")
	files := []string{".gitignore", "go.sum"}
	for _, c := range []struct {
		name         string
		ext, handler []byte
		want         int
	}{
		{"no extensions", nil, handler, 1},
		{"no handler", ext, nil, 1},
		{"neither", nil, nil, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			warned := 0
			f, err := newFileFilter(nil, c.ext, c.handler, func(string, ...any) { warned++ })
			if err != nil {
				t.Fatal(err)
			}
			if got := f.kept(files); got != c.want {
				t.Errorf("kept = %d, want %d", got, c.want)
			}
			if warned != 1 {
				t.Errorf("warnings = %d, want the one that says the list is gone", warned)
			}
		})
	}
}

// Upstream's file_filter.py compiles a pattern or logs and drops it. RE2 refuses
// what Python accepts, so a dropped pattern must only keep files in the diff.
func TestFileFilter_AnUncompilablePatternIsDroppedAndNarrows(t *testing.T) {
	cfg := []byte("[ignore]\nregex = ['(?=lookahead)', '^docs/']\n")
	warned := 0
	f, err := newFileFilter(cfg, nil, nil, func(string, ...any) { warned++ })
	if err != nil {
		t.Fatal(err)
	}
	if warned != 2 { // the pattern, and the empty type lists
		t.Errorf("warnings = %d, want 2", warned)
	}
	if got := f.kept([]string{"docs/a.md", "lookahead"}); got != 1 {
		t.Errorf("kept = %d, want 1: only ^docs/ applies", got)
	}
}

func TestNewFileFilter_ASourceThatDoesNotParseIsNoAnswer(t *testing.T) {
	for name, c := range map[string][2][]byte{
		"config":     {[]byte("[ignore\n"), nil},
		"glob type":  {[]byte("[ignore]\nglob = 'vendor/**'\n"), nil},
		"regex type": {[]byte("[ignore]\nregex = 'a'\n"), nil},
		"extensions": {nil, []byte("[bad_extensions\n")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := newFileFilter(c[0], c[1], nil, func(string, ...any) {}); err == nil {
				t.Error("want an error, so the review runs")
			}
		})
	}
}

func TestIgnorePatterns_RegexMatchesFromTheStartOfThePath(t *testing.T) {
	ps, err := ignorePatterns([]byte("[ignore]\nregex = ['docs/', 'a|b']\n"), noWarn(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fileFilter{patterns: ps}
	for file, want := range map[string]bool{"docs/x": true, "x/docs/y": false, "a1": true, "b1": true, "xb": false} {
		if got := f.ignored(file); got != want {
			t.Errorf("ignored(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestIgnorePatterns_ALeadingGlobstarAlsoMatchesAtTheRoot(t *testing.T) {
	ps, err := ignorePatterns([]byte("[ignore]\nglob = ['**/*.gen']\n"), noWarn(t))
	if err != nil {
		t.Fatal(err)
	}
	f := &fileFilter{patterns: ps}
	if !f.ignored("root.gen") || !f.ignored("a/b.gen") || f.ignored("a.genx") {
		t.Errorf("patterns %v", ps)
	}
}

// Each `want` was read from upstream's translate_globs_to_regexes at the audited
// commit, including the bounds: past them PR-Agent keeps the file.
func TestIgnorePatterns_ExpandsInteriorGlobstars(t *testing.T) {
	seven := strings.Repeat("**/", 7) + "x"
	fits := "q/**/" + strings.Repeat("a", 249) + "/x"    // 256 runes
	tooLong := "q/**/" + strings.Repeat("a", 250) + "/x" // 257 runes
	wide := "q/**/" + strings.Repeat("ñ", 249) + "/x"    // 256 runes, 505 bytes
	for _, c := range []struct {
		name  string
		globs []string
		file  string
		want  bool
	}{
		{"an interior globstar also means no directory", []string{"a/**/b"}, "a/b", true},
		{"and still one or more", []string{"a/**/b"}, "a/x/y/b", true},
		{"every subset of two", []string{"a/**/b/**/c"}, "a/b/c", true},
		{"one dropped, one kept", []string{"a/**/b/**/c"}, "a/x/b/c", true},
		{"a leading and an interior one", []string{"**/a/**/b"}, "a/b", true},
		{"only a whole segment expands", []string{"a/x**/b"}, "a/b", false},
		{"a final `**` has no `/` to drop", []string{"a/**"}, "a", false},
		{"a globstar inside a set is not one", []string{"[a/**/]b"}, "b", false},
		{"nor does the set end at its own `/`", []string{"[!a/**/b]c"}, "*c", false},
		{"a `[` no later `]` closes does not hide it", []string{"[a/**/b"}, "[a/b", true},
		{"a set closed by the last `]` hides it", []string{"[!]/**/]b/**/c"}, "xb/c", true},
		{"six globstars expand", []string{strings.Repeat("**/", 6) + "x"}, "x", true},
		{"seven do not", []string{seven}, "a/x", false},
		{"but seven still lose the leading one", []string{seven}, "a/b/c/d/e/f/x", true},
		{"256 runes expand", []string{fits}, "q/" + strings.Repeat("a", 249) + "/x", true},
		{"257 do not", []string{tooLong}, "q/" + strings.Repeat("a", 250) + "/x", false},
		{"the bound counts runes, not bytes", []string{wide}, "q/" + strings.Repeat("ñ", 249) + "/x", true},
		{"a repeated glob does not spend the quota", []string{"a/**/b", "a/**/b"}, "a/b", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &fileFilter{}
			for _, source := range globSources(c.globs) {
				if re, err := regexp.Compile(source); err == nil {
					f.patterns = append(f.patterns, re)
				}
			}
			if got := f.ignored(c.file); got != c.want {
				t.Errorf("ignored(%q) = %v, want %v", c.file, got, c.want)
			}
		})
	}
}

// Upstream stops at 256 new translations across every glob, and a form already
// translated costs nothing: past the quota a file stays in the diff.
// Forms are compared by translation, so a repeat is one regex and costs no quota.
func TestGlobSources_CountsTranslations(t *testing.T) {
	for _, c := range []struct {
		globs []string
		want  int
	}{
		{[]string{"a/**/b", "a/**/b"}, 2},
		{[]string{"**/a/**/b"}, 4},
		{[]string{"a**b", "a*b"}, 1},
		{[]string{"**/"}, 1},
	} {
		if got := globSources(c.globs); len(got) != c.want {
			t.Errorf("globSources(%q) = %q, want %d", c.globs, got, c.want)
		}
	}
}

func TestGlobSources_TheVariantQuotaIsUpstreams(t *testing.T) {
	var globs []string
	for _, name := range []string{"p", "q", "r", "s", "t"} { // 63 forms each
		globs = append(globs, name+"/**/a/**/b/**/c/**/d/**/e/**/x")
	}
	sources := globSources(globs)
	if len(sources) != 5+256 {
		t.Fatalf("sources = %d, want 5 globs and 256 variants", len(sources))
	}
	covered := func(file string) bool {
		f := &fileFilter{}
		for _, source := range sources {
			f.patterns = append(f.patterns, regexp.MustCompile(source))
		}
		return f.ignored(file)
	}
	if !covered("s/a/b/x/c/d/e/x") || covered("t/a/b/c/d/e/x") {
		t.Error("the quota did not fall between the fourth and fifth globs")
	}
}

func TestTranslateGlob(t *testing.T) {
	for _, c := range []struct {
		glob, file string
		want       bool
	}{
		{"*.go", "a/b.go", true}, // `*` crosses `/`
		{"a?c", "abc", true},
		{"a?c", "ac", false},
		{"a?c", "a\nc", true}, // DOTALL
		{"a.c", "abc", false}, // a dot is literal
		{"a", "ab", false},    // the whole path
		{"a", "ba", false},
		{"[ab]x", "bx", true},
		{"[!ab]x", "bx", false},
		{"[!ab]x", "cx", true},
		{"[a-c]x", "bx", true},
		{"[]]x", "]x", true},
		{"[!]]x", "]x", false},
		{"[x", "[x", true}, // unclosed: literal
		{"[!", "[!", true},
		{"[a^]x", "^x", true},
		{`[\d]x`, `\x`, true},
		{`[\d]x`, "1x", false},
		{"[[:alpha:]]x", "ax", false}, // `[[:alpha:]` then a literal `]`
		{"[[:alpha:]]x", ":]x", true},
		{"(a)+", "(a)+", true},
		{"(a)+", "aa", false},
		{"a**b", "a/x/b", true}, // a run of stars is one, as fnmatch compresses it
	} {
		t.Run(c.glob+" "+c.file, func(t *testing.T) {
			ps, err := ignorePatterns([]byte("[ignore]\nglob = ['"+c.glob+"']\n"), noWarn(t))
			if err != nil || len(ps) != 1 {
				t.Fatalf("patterns %v, err %v", ps, err)
			}
			if got := ps[0].MatchString(c.file); got != c.want {
				t.Errorf("%s (%s) on %q = %v, want %v", c.glob, ps[0], c.file, got, c.want)
			}
		})
	}
}

func TestPythonStringList(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"a module frozenset", "X = frozenset({\n  'a', \"b\", # note\n  'c',\n})\n", "a,b,c"},
		{"a function's tuple", "def f():\n    x = ('a', 'b')\n", "a,b"},
		{"a list", "X = ['a']\n", "a"},
		{"another name", "Y = ('a',)\n", ""},
		{"a computed value", "X = sorted(y)\n", ""},
		{"a non-string member", "X = ('a', 1)\n", ""},
		{"an escaped quote", `X = ('a\'b',)`, ""},
		{"an escape", `X = ('a\tb',)`, ""},
		{"a newline inside a string", "X = ('a\nb',)", ""},
		{"never closed", "X = ('a',", ""},
		{"an unclosed string", "X = ('a", ""},
		{"a comment that never ends", "X = ('a', # note", ""},
		{"nothing", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			name := "X"
			if strings.HasPrefix(c.src, "def") {
				name = "x"
			}
			if got := strings.Join(pythonStringList([]byte(c.src), name), ","); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
