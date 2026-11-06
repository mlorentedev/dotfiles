package docpaths

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// RepositoryRoot resolves the checkout containing dir without inheriting
// per-command Git configuration overrides from the caller.
func RepositoryRoot(dir string) (string, error) {
	root, _, err := RepositoryLocation(dir)
	return root, err
}

// RepositoryLocation resolves the checkout root and the slash-separated
// working-directory prefix Git uses inside that checkout.
func RepositoryLocation(dir string) (string, string, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel", "--show-prefix")
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(lines) != 2 {
		return "", "", fmt.Errorf("git rev-parse returned %d lines, want root and prefix", len(lines))
	}
	root := strings.TrimSuffix(lines[0], "\r")
	prefix := filepath.ToSlash(strings.TrimSuffix(lines[1], "\r"))
	return root, prefix, nil
}

func Discover(repo string, policy Policy) ([]string, error) {
	out, err := runGit(repo, "ls-files", "-z", "--", "*.md")
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

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = cleanGitConfigEnv(os.Environ())
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	message := fmt.Sprintf("git %s from %s", strings.Join(args, " "), dir)
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if detail := strings.TrimSpace(string(exit.Stderr)); detail != "" {
			return nil, fmt.Errorf("%s: %w; git said: %s", message, err, detail)
		}
	}
	return nil, fmt.Errorf("%s: %w", message, err)
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
