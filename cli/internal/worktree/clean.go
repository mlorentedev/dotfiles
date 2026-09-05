package worktree

import (
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
	".svelte-kit":       true,
	".output":           true,
	".gradle":           true,
	".mvn":              true,
	".tox":              true,
	"__pycache__":        true,
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
	clean := filepath.ToSlash(strings.TrimSpace(relPath))
	clean = strings.TrimPrefix(clean, "!! ")
	clean = strings.TrimSpace(clean)
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
	lines := strings.Split(porcelainOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "!! ") {
			continue
		}
		relPath := strings.TrimSpace(strings.TrimPrefix(line, "!!"))
		if !IsDisposableIgnoredPath(relPath) {
			return true
		}
	}
	return false
}
