package evals

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// ReplayCaseDir holds a repository's replay cases, one <id>.json each, beside
// the archive they replay (ADR-063 §1).
const ReplayCaseDir = ".dotf/evals/cases"

// PlantedCaseDir holds planted cases, one <id>/case.json each with its prompt
// and reference answer. Only dotfiles carries it; elsewhere it is absent.
const PlantedCaseDir = "harness/evals/planted"

// CheckResult is the outcome of checking one case file.
type CheckResult struct {
	Path     string // repo-relative, slash-separated
	ID       string
	Kind     string
	Problems []string
}

// Check validates and self-grades every case under root. A case passes when it
// matches the schema, agrees with its source's frontmatter, and its key graded
// against its own source scores full recall and matches every optional entry;
// when a planted reference hits no decoy; and when each decoy and noise sample
// matches its own anchors.
//
// The error is for a tree that cannot be read; a broken case is a problem in
// its result, so one bad case never hides the rest.
func Check(root string) ([]CheckResult, error) {
	paths, err := casePaths(root)
	if err != nil {
		return nil, err
	}
	results := make([]CheckResult, 0, len(paths))
	for _, rel := range paths {
		results = append(results, checkOne(root, rel))
	}
	return results, nil
}

// casePaths lists the case files under root, sorted, slash-separated. An
// absent case directory holds no cases; any other read error is returned,
// because a directory that cannot be read would otherwise check clean.
func casePaths(root string) ([]string, error) {
	var out []string
	replay, err := readDirIfExists(filepath.Join(root, filepath.FromSlash(ReplayCaseDir)))
	if err != nil {
		return nil, err
	}
	for _, e := range replay {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			out = append(out, ReplayCaseDir+"/"+e.Name())
		}
	}
	planted, err := readDirIfExists(filepath.Join(root, filepath.FromSlash(PlantedCaseDir)))
	if err != nil {
		return nil, err
	}
	for _, e := range planted {
		if !e.IsDir() {
			continue
		}
		rel := PlantedCaseDir + "/" + e.Name() + "/case.json"
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

func readDirIfExists(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}

// LoadCase reads, validates and decodes one case file.
func LoadCase(path string) (Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Case{}, err
	}
	if err := ValidateCase(data); err != nil {
		return Case{}, fmt.Errorf("schema: %w", err)
	}
	var c Case
	if err := json.Unmarshal(data, &c); err != nil {
		return Case{}, fmt.Errorf("decode: %w", err)
	}
	return c, nil
}

func checkOne(root, rel string) CheckResult {
	res := CheckResult{Path: rel}
	c, err := LoadCase(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	res.ID, res.Kind = c.ID, c.Kind
	if want := caseNameFor(rel); want != c.ID {
		res.Problems = append(res.Problems, fmt.Sprintf("file name says %q but id is %q", want, c.ID))
	}

	sourceField, sourcePath := "source_review", c.SourceReview
	if c.Kind == "planted" {
		sourceField, sourcePath = "reference_answer", c.ReferenceAnswer
		if p := promptProblem(root, c.Prompt); p != "" {
			res.Problems = append(res.Problems, p)
		}
	}
	full, err := insideRoot(root, sourcePath)
	if err != nil {
		res.Problems = append(res.Problems, fmt.Sprintf("%s: %v", sourceField, err))
		return res
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = fmt.Errorf("%s %s does not exist", sourceField, sourcePath)
		}
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	source := string(data)
	res.Problems = append(res.Problems, frontmatterProblems(c, source, sourceField)...)
	res.Problems = append(res.Problems, keyProblems(c, source, sourceField)...)
	res.Problems = append(res.Problems, sampleProblems(c)...)
	return res
}

// insideRoot resolves a repo-relative case path, refusing one that climbs out
// of the repository: a case is checked against files its repository owns.
func insideRoot(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the repository", rel)
	}
	return filepath.Join(root, clean), nil
}

// promptProblem reports a planted prompt that is not a readable file. The
// check grades the reference answer; a run sends the prompt.
func promptProblem(root, rel string) string {
	full, err := insideRoot(root, rel)
	if err != nil {
		return "prompt: " + err.Error()
	}
	info, err := os.Stat(full)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Sprintf("prompt %s does not exist", rel)
		}
		return "prompt: " + err.Error()
	}
	if info.IsDir() {
		return fmt.Sprintf("prompt %s is a directory", rel)
	}
	return ""
}

// caseNameFor is the id a case file's location promises.
func caseNameFor(rel string) string {
	if strings.HasSuffix(rel, "/case.json") {
		return filepath.Base(filepath.Dir(filepath.FromSlash(rel)))
	}
	return strings.TrimSuffix(filepath.Base(filepath.FromSlash(rel)), ".json")
}

// frontmatterProblems holds the case's header to its source's. A case that
// copies its sha or verdict wrong would replay a tree nobody reviewed. A
// planted reference names no commit, so only its verdict is held.
func frontmatterProblems(c Case, source, field string) []string {
	if c.Kind != "replay" {
		if v := string(spec.FrontmatterVerdict(source)); v != c.ExpectedVerdict {
			return []string{fmt.Sprintf("expected_verdict %s, but %s says %q", c.ExpectedVerdict, field, v)}
		}
		return nil
	}
	r, err := spec.ParseReview(source)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", field, err)}
	}
	var out []string
	if string(r.Verdict) != c.ExpectedVerdict {
		out = append(out, fmt.Sprintf("expected_verdict %s, but %s says %s", c.ExpectedVerdict, field, r.Verdict))
	}
	if r.ReviewedSHA != c.ReviewedSHA {
		out = append(out, fmt.Sprintf("reviewed_sha %s, but %s says %s", c.ReviewedSHA, field, r.ReviewedSHA))
	}
	if r.Reviewer != c.KeyAuthor {
		out = append(out, fmt.Sprintf("key_author %s, but %s was written by %s", c.KeyAuthor, field, r.Reviewer))
	}
	return out
}

// keyProblems grades the key against its own source.
func keyProblems(c Case, source, field string) []string {
	g := GradeAnswer(c, source)
	var out []string
	if g.Format != FormatOK {
		out = append(out, fmt.Sprintf("%s has no findings rows (%s)", field, g.Format))
	}
	for _, id := range g.Missed {
		out = append(out, fmt.Sprintf("expected %s matches no row of %s", id, field))
	}
	got := map[string]bool{}
	for _, m := range g.OptionalMatched {
		got[m.ID] = true
	}
	for _, f := range c.Optional {
		if !got[f.ID] {
			out = append(out, fmt.Sprintf("optional %s matches no row of %s left after the expected entries", f.ID, field))
		}
	}
	// A replay source is a model's review and may carry the false claim its
	// decoy was mined from. A planted reference is the ideal answer and may not.
	// A decoy broad enough to swallow a true row is caught either way: decoys
	// are graded first, so that row's expected entry is reported unmatched.
	if c.Kind == "planted" {
		for _, m := range g.DecoyHits {
			out = append(out, fmt.Sprintf("decoy %s matches line %d of %s, which is the ideal answer", m.ID, m.Row.Line, field))
		}
	}
	return out
}

// sampleProblems asserts each decoy and noise entry matches its own sample. A
// source never contains a decoy, so nothing else exercises its anchors.
func sampleProblems(c Case) []string {
	var out []string
	for _, group := range []struct {
		name    string
		entries []Finding
	}{{"decoy", c.Decoys}, {"noise", c.Noise}} {
		for _, f := range group.entries {
			if !matches(f, Row{Severity: "Blocker", Text: f.Sample}) {
				out = append(out, fmt.Sprintf("%s %s does not match its own sample", group.name, f.ID))
			}
		}
	}
	return out
}
