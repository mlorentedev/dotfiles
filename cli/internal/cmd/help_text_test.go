package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// fixedPrefixes are id prefixes that do not come from a spec folder: ADRs,
// guards and the other ids the repository uses outside specs/.
var fixedPrefixes = []string{"ADR", "GUARD", "CLI", "OPS", "BUG", "HARNESS", "SEC", "AI", "TOOL", "WIN", "SDD", "MEMORY", "REFACTOR", "TEST", "CI", "SKILL", "DOC", "INFRA", "SPEC", "APP", "HIVE"}

// specArea reads the AREA of a spec id (DX-006-orca -> DX). Dated ids have none.
var specArea = regexp.MustCompile(`^([A-Z]+[0-9]*(?:-[A-Z]+[0-9]*)*)-[0-9]+`)

// internalRef matches references that mean something only inside this
// repository: ticket and decision ids (ADR-036, CLI-067), issue numbers (#1381),
// lesson numbers and the migration jargon "twin". Help text is read by people
// who have none of that context, so it names behaviour instead. Code comments
// keep the ids.
//
// The id prefixes are read from specs/ and specs/archive/ as well as the fixed
// list, so a spec area added later is covered without editing this file. A
// hand-kept list missed DX, and a DX-006 reached --help.
func internalRef(t *testing.T) *regexp.Regexp {
	t.Helper()
	seen := map[string]bool{}
	areas := 0
	prefixes := append([]string(nil), fixedPrefixes...)
	for _, p := range prefixes {
		seen[p] = true
	}
	for _, dir := range []string{"../../../specs", "../../../specs/archive"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			m := specArea.FindStringSubmatch(e.Name())
			if !e.IsDir() || m == nil {
				continue
			}
			areas++
			if !seen[m[1]] {
				seen[m[1]] = true
				prefixes = append(prefixes, m[1])
			}
		}
	}
	if areas == 0 {
		t.Fatal("no spec area found under specs/: the prefix list would silently stay hand-kept")
	}
	sort.Slice(prefixes, func(i, j int) bool { return len(prefixes[i]) > len(prefixes[j]) })
	return regexp.MustCompile(`\b(` + strings.Join(prefixes, "|") + `)-[0-9]+|#[0-9]{2,5}\b|\blessons? [0-9]+\b|\btwins?\b`)
}

// TestHelpTextHasNoInternalReferences walks every command and checks each piece
// of text `--help` prints: the short and long descriptions, the examples and
// every flag's usage line.
func TestHelpTextHasNoInternalReferences(t *testing.T) {
	ref := internalRef(t)
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		check := func(field, text string) {
			for _, line := range strings.Split(text, "\n") {
				if m := ref.FindString(line); m != "" {
					t.Errorf("%q %s mentions %q: %s", c.CommandPath(), field, m, strings.TrimSpace(line))
				}
			}
		}
		check("Short", c.Short)
		check("Long", c.Long)
		check("Example", c.Example)
		c.LocalFlags().VisitAll(func(f *pflag.Flag) { check("flag --"+f.Name, f.Usage) })
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(New("dev", ""))
}

// TestReadmeHasNoInternalReferences holds cli/README.md to the same pattern as
// --help, so the two cannot drift apart: the README is the other thing a new
// user reads first.
func TestReadmeHasNoInternalReferences(t *testing.T) {
	ref := internalRef(t)
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read cli/README.md: %v", err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		if m := ref.FindString(line); m != "" {
			t.Errorf("cli/README.md:%d mentions %q: %s", i+1, m, strings.TrimSpace(line))
		}
	}
}

// outputCleanFiles are the sources whose printed strings have been cleaned of
// internal ids. Output is built inside RunE and the doctor checks, where the
// cobra walk above cannot reach, so these are read as Go source. A file joins
// the list once its strings name behaviour; the rest are tracked in #1891.
var outputCleanFiles = []string{"orca.go", "../doctor/checks_orca.go"}

// TestOutputStringsHaveNoInternalReferences checks every string literal in
// outputCleanFiles against the same pattern as --help. Comments are not string
// literals, so they keep their ids.
func TestOutputStringsHaveNoInternalReferences(t *testing.T) {
	ref := internalRef(t)
	for _, file := range outputCleanFiles {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		literals := 0
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			literals++
			if m := ref.FindString(lit.Value); m != "" {
				t.Errorf("%s mentions %q: %s", fset.Position(lit.Pos()), m, lit.Value)
			}
			return true
		})
		if literals == 0 {
			t.Errorf("%s has no string literals: the path is wrong or the file moved", file)
		}
	}
}
