package worktree

import (
	"fmt"
	"path/filepath"
	"strings"
)

var disposableDirNames = map[string]bool{
	"node_modules":      true,
	"target":            true,
	".venv":             true,
	"venv":              true,
	".cache":            true,
	".cache_ggshield":   true,
	".npm":              true,
	".yarn":             true,
	"dist":              true,
	"build":             true,
	"out":               true,
	"bin":               true,
	"obj":               true,
	"vendor":            true,
	".turbo":            true,
	".next":             true,
	".nuxt":             true,
	".astro":            true,
	".svelte-kit":       true,
	".output":           true,
	".gradle":           true,
	".mvn":              true,
	".tox":              true,
	"__pycache__":       true,
	".pytest_cache":     true,
	".mypy_cache":       true,
	".ruff_cache":       true,
	"htmlcov":           true,
	".terraform":        true,
	".terragrunt-cache": true,
}

func isDisposableBinaryOrLog(path string) bool {
	if path == "dotf" || path == "dotf.exe" || path == "cli/dotf" || path == "cli/dotf.exe" {
		return true
	}
	if strings.HasPrefix(path, "specs/") && (strings.HasSuffix(path, "review-transcript.jsonl") || strings.HasSuffix(path, "review-transcript.jsonl.stderr")) {
		return true
	}
	ext := filepath.Ext(path)
	switch ext {
	case ".pyc", ".class", ".o", ".a", ".so", ".dylib", ".dll":
		return true
	default:
		return false
	}
}

func isDisposableDirectoryPath(path string) bool {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if disposableDirNames[part] {
			return true
		}
	}
	return false
}

// IsDisposableIgnoredPath checks if an ignored relative path is disposable (e.g. build cache, metadata).
func IsDisposableIgnoredPath(relPath string) bool {
	clean := strings.TrimSpace(relPath)
	clean = strings.TrimPrefix(clean, "!! ")
	return isDisposableExactPath(strings.TrimSpace(clean))
}

// isDisposableExactPath classifies a path exactly as git spelled it: no
// whitespace is trimmed, so " node_modules" is not node_modules.
func isDisposableExactPath(relPath string) bool {
	clean := filepath.ToSlash(relPath)
	clean = strings.TrimPrefix(clean, "./")
	clean = strings.TrimSuffix(clean, "/")

	if clean == "" || clean == MetadataFileName || clean == ".DS_Store" || clean == "Thumbs.db" {
		return true
	}
	if clean == "coverage.out" || clean == "coverage.html" || clean == ".coverage" {
		return true
	}
	if isDisposableBinaryOrLog(clean) {
		return true
	}
	return isDisposableDirectoryPath(clean)
}

// HasNonDisposableIgnored inspects porcelain output and returns true if any non-disposable ignored file exists.
func HasNonDisposableIgnored(porcelainOutput string) bool {
	return len(NonDisposableIgnored(porcelainOutput)) > 0
}

// NonDisposableIgnored lists, in git's order, the ignored paths in
// `git status --ignored --porcelain` output that are not known build output.
// Entries are NUL-separated when the output came from -z, which leaves paths
// unquoted and exact, so nothing is trimmed from them: a name with a leading
// or trailing space is classified, and reported, as git spelled it. Output
// without -z is newline-separated and trimmed.
func NonDisposableIgnored(porcelainOutput string) []string {
	exact := strings.Contains(porcelainOutput, "\x00")
	sep := "\n"
	if exact {
		sep = "\x00"
	}
	var paths []string
	for _, entry := range strings.Split(porcelainOutput, sep) {
		if !exact {
			entry = strings.TrimSpace(entry)
		}
		relPath, ok := strings.CutPrefix(entry, "!! ")
		if !ok {
			continue
		}
		disposable := isDisposableExactPath(relPath)
		if !exact {
			relPath = strings.TrimSpace(relPath)
			disposable = IsDisposableIgnoredPath(relPath)
		}
		if !disposable {
			paths = append(paths, relPath)
		}
	}
	return paths
}

// summarizePaths names the first few paths and counts the rest, so a refusal
// stays one line on a worktree holding hundreds of them.
func summarizePaths(paths []string) string {
	const shown = 3
	if len(paths) <= shown {
		return strings.Join(paths, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(paths[:shown], ", "), len(paths)-shown)
}
