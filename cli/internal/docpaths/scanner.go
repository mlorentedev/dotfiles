package docpaths

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	tokenPattern   = regexp.MustCompile("`([^`\\s]*)`")
	allCapsSegment = regexp.MustCompile(`(^|/)[A-Z][A-Z0-9_]+(\.|/|$)`)
	bareSource     = regexp.MustCompile(`(^|[^./a-zA-Z0-9_-])\. ([a-zA-Z_][a-zA-Z0-9_.-]*\.[a-zA-Z0-9]+)`)
)

type CheckOptions struct {
	RepoRoot  string
	VaultRoot string
	Policy    Policy
}

type Finding struct {
	File    string
	Token   string
	Kind    string
	Message string
}

type Report struct {
	Checked  []string
	Findings []Finding
}

func CheckFiles(opts CheckOptions, files []string) (Report, error) {
	top, err := topLevelEntries(opts.RepoRoot)
	if err != nil {
		return Report{}, err
	}
	report := Report{Checked: append([]string(nil), files...)}
	for _, file := range files {
		findings, err := checkFile(opts, top, file)
		if err != nil {
			return Report{}, err
		}
		report.Findings = append(report.Findings, findings...)
	}
	sort.Strings(report.Checked)
	sort.Slice(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		return a.File+"\x00"+a.Token+"\x00"+a.Kind < b.File+"\x00"+b.Token+"\x00"+b.Kind
	})
	return report, nil
}

func topLevelEntries(root string) (map[string]bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read repository root: %w", err)
	}
	out := make(map[string]bool, len(entries))
	for _, entry := range entries {
		out[entry.Name()] = true
	}
	delete(out, ".git")
	return out, nil
}

func checkFile(opts CheckOptions, top map[string]bool, rel string) ([]Finding, error) {
	path := rel
	if !filepath.IsAbs(path) {
		path = filepath.Join(opts.RepoRoot, filepath.FromSlash(rel))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	var findings []Finding
	for _, token := range uniqueTokens(string(raw)) {
		finding, ok, err := checkToken(opts, top, rel, token)
		if err != nil {
			return nil, err
		}
		if ok {
			findings = append(findings, finding)
		}
	}
	findings = append(findings, checkBareSources(rel, string(raw), opts.Policy.SourceExtensions)...)
	return findings, nil
}

func uniqueTokens(text string) []string {
	seen := map[string]bool{}
	for _, match := range tokenPattern.FindAllStringSubmatch(text, -1) {
		if match[1] != "" {
			seen[match[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for token := range seen {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func checkToken(opts CheckOptions, top map[string]bool, file, token string) (Finding, bool, error) {
	if finding, handled := checkVaultToken(opts, file, token); handled {
		return finding, finding.Kind != "", nil
	}
	if ignoreToken(token, opts.Policy) {
		return Finding{}, false, nil
	}
	token = strings.TrimPrefix(token, "./")
	if !strings.Contains(token, "/") || !top[firstSegment(token)] {
		return Finding{}, false, nil
	}
	if hasParentSegment(token) {
		return newFinding(file, token, "escape", "path escapes the repository root"), true, nil
	}
	if strings.ContainsAny(token, "*?[") {
		matched, err := globExists(opts.RepoRoot, token)
		if err != nil {
			return Finding{}, false, fmt.Errorf("expand glob %q: %w", token, err)
		}
		if !matched {
			return newFinding(file, token, "empty-glob", "glob matches nothing"), true, nil
		}
		return Finding{}, false, nil
	}
	if _, err := os.Stat(filepath.Join(opts.RepoRoot, filepath.FromSlash(token))); err != nil {
		return newFinding(file, token, "missing", "referenced path does not exist"), true, nil
	}
	return Finding{}, false, nil
}

func globExists(root, pattern string) (bool, error) {
	parts := strings.Split(path.Clean(filepath.ToSlash(pattern)), "/")
	return matchGlob(root, parts)
}

func matchGlob(path string, parts []string) (bool, error) {
	if len(parts) == 0 {
		return true, nil
	}
	part := parts[0]
	if !strings.ContainsAny(part, "*?[") {
		next := filepath.Join(path, filepath.FromSlash(part))
		info, err := os.Stat(next)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if len(parts) > 1 && !info.IsDir() {
			return false, nil
		}
		return matchGlob(next, parts[1:])
	}
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	found := false
	for _, entry := range entries {
		matched, matchErr := filepath.Match(part, entry.Name())
		if errors.Is(matchErr, filepath.ErrBadPattern) {
			return false, nil
		}
		if matchErr != nil {
			return false, matchErr
		}
		if !matched {
			continue
		}
		child := filepath.Join(path, entry.Name())
		if len(parts) > 1 {
			info, statErr := os.Stat(child)
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			if statErr != nil {
				return false, statErr
			}
			if !info.IsDir() {
				continue
			}
		}
		childFound, childErr := matchGlob(child, parts[1:])
		if childErr != nil {
			return false, childErr
		}
		found = found || childFound
	}
	return found, nil
}

func checkVaultToken(opts CheckOptions, file, token string) (Finding, bool) {
	const prefix = "$VAULT_PATH/"
	const braced = "${VAULT_PATH}/"
	if !strings.HasPrefix(token, prefix) && !strings.HasPrefix(token, braced) {
		return Finding{}, false
	}
	if opts.VaultRoot == "" || strings.ContainsAny(token, "<>&|;=@()*") {
		return Finding{}, true
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(token, prefix), braced)
	if _, err := os.Stat(filepath.Join(opts.VaultRoot, filepath.FromSlash(rel))); err != nil {
		return newFinding(file, token, "missing-vault", "referenced vault path does not exist"), true
	}
	return Finding{}, true
}

func ignoreToken(token string, policy Policy) bool {
	if token == "" || strings.HasPrefix(token, "/") || strings.HasPrefix(token, "~") ||
		strings.HasPrefix(token, "$") || strings.HasPrefix(token, "-") ||
		strings.Contains(token, "://") || strings.ContainsAny(token, "<>&|;=@()") {
		return true
	}
	if allCapsSegment.MatchString(token) && !hasKnownExtension(token, policy.KnownExtensions) {
		return true
	}
	return false
}

func hasKnownExtension(token string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(token, ext) {
			return true
		}
	}
	return false
}

func firstSegment(token string) string {
	first, _, _ := strings.Cut(filepath.ToSlash(token), "/")
	return first
}

func hasParentSegment(token string) bool {
	for _, part := range strings.Split(filepath.ToSlash(token), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func newFinding(file, token, kind, message string) Finding {
	return Finding{File: file, Token: token, Kind: kind, Message: message}
}

func checkBareSources(file, text string, extensions []string) []Finding {
	var out []Finding
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, match := range bareSource.FindAllStringSubmatch(line, -1) {
			if hasKnownExtension(match[2], extensions) {
				out = append(out, newFinding(file, match[2], "bare-source", "source path needs ./ for zsh"))
			}
		}
	}
	return out
}
