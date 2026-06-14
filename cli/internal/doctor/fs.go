package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"runtime"
	"strings"
)

// pathExists reports whether p exists (file, dir, or valid symlink target).
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// isDir reports whether p exists and is a directory (follows symlinks).
func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// isExecFile reports whether p is a regular executable file — the `[ -x ]`
// test the twins used for versioned tool binaries. On Windows the POSIX exec
// bit does not exist (os.Chmod cannot set it), so a regular file suffices;
// that branch is only exercised by the cross-OS `go test` matrix, since the
// Windows production wiring of `dotf doctor` is deferred.
func isExecFile(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return fi.Mode().Perm()&0o111 != 0
}

// isSymlink reports whether p is itself a symlink (does not follow it).
func isSymlink(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// filesEqual reports byte-for-byte equality of two files — the `cmp -s` test
// behind check_deployed (drift detection for copy-deployed configs, ADR-012).
func filesEqual(a, b string) bool {
	ba, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	bb, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ba, bb)
}

// isValidJSON reports whether p contains parseable JSON (replaces `jq '.'`).
func isValidJSON(p string) bool {
	raw, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return json.Valid(raw)
}

// fileContains reports whether p exists and contains substr. Replaces the
// `grep -qF` membership checks (installed_plugins.json, $schema, {env:}).
func fileContains(p, substr string) bool {
	raw, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return strings.Contains(string(raw), substr)
}

// expandHome substitutes $HOME and ${HOME} with the resolved home dir, leaving
// other variables untouched — the exact scope of doctor.sh's expand_path.
func expandHome(sys *System, s string) string {
	home := sys.home()
	s = strings.ReplaceAll(s, "${HOME}", home)
	s = strings.ReplaceAll(s, "$HOME", home)
	return s
}

// pathContains reports whether entry is one of the PATH entries.
func pathContains(sys *System, entry string) bool {
	for _, e := range sys.pathEntries() {
		if e == entry {
			return true
		}
	}
	return false
}
