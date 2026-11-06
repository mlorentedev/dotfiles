package docpaths

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func Discover(repo string, policy Policy) ([]string, error) {
	cmd := exec.Command("git", "-C", repo, "ls-files", "-z", "--", "*.md")
	cmd.Env = cleanGitConfigEnv(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked markdown files: %w", err)
	}
	files := governedFiles(strings.Split(string(bytes.TrimRight(out, "\x00")), "\x00"), policy)
	if err := requireMembers(files, policy.RequiredMembers); err != nil {
		return nil, err
	}
	return files, nil
}

func governedFiles(files []string, policy Policy) []string {
	names := make(map[string]bool, len(policy.GovernedBasenames))
	for _, name := range policy.GovernedBasenames {
		names[name] = true
	}
	var out []string
	for _, file := range files {
		file = filepath.ToSlash(file)
		if file == "" || excluded(file, policy.ExcludePrefixes) || !names[filepath.Base(file)] {
			continue
		}
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}

func excluded(file string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(file, filepath.ToSlash(prefix)) {
			return true
		}
	}
	return false
}

func requireMembers(files, required []string) error {
	found := make(map[string]bool, len(files))
	for _, file := range files {
		found[file] = true
	}
	var missing []string
	for _, file := range required {
		if !found[file] {
			missing = append(missing, file)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required governed member(s) missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

func cleanGitConfigEnv(environment []string) []string {
	out := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if name == "GIT_CONFIG_COUNT" || strings.HasPrefix(name, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(name, "GIT_CONFIG_VALUE_") {
			continue
		}
		out = append(out, entry)
	}
	return out
}
