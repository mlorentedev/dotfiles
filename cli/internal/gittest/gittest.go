// Package gittest keeps a test process's git from forking a background gc that
// outlives the test (#2803). It works through the environment, not repo config,
// because that reaches clones and remotes too; for a remote reached over a local
// transport, only GIT_CONFIG_SYSTEM does. See lesson 517.
package gittest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	countVar  = "GIT_CONFIG_COUNT"
	systemVar = "GIT_CONFIG_SYSTEM"
	// filePrefix names the system config Quiet writes, so a re-executed test
	// binary recognises its parent's file instead of including it in itself.
	filePrefix = "gittest-"
)

var quiet = map[string]string{
	countVar:             "2",
	"GIT_CONFIG_KEY_0":   "gc.auto",
	"GIT_CONFIG_VALUE_0": "0",
	"GIT_CONFIG_KEY_1":   "maintenance.auto",
	"GIT_CONFIG_VALUE_1": "false",
}

// remoteKeys go in the system config Quiet points GIT_CONFIG_SYSTEM at. A
// local push or fetch starts the remote side with GIT_CONFIG_COUNT unset, so
// the keys above never reach the remote's receive-pack, which then runs
// `git maintenance run --auto --detach` (#3031). GIT_CONFIG_SYSTEM survives
// that transport.
const remoteKeys = "[gc]\n\tauto = 0\n[maintenance]\n\tauto = false\n[receive]\n\tautogc = false\n"

// Quiet turns off auto gc and auto maintenance for every git this process
// starts, the remote side of a local transport included. Call it from TestMain
// before m.Run. A test binary that re-executes itself inherits the keys, so
// finding exactly these is fine; any other GIT_CONFIG_COUNT panics, since
// overwriting it would drop the caller's keys.
func Quiet() {
	if v, ok := os.LookupEnv(countVar); ok {
		if alreadyQuiet() {
			return
		}
		panic("gittest: " + countVar + " is already set (" + v + ")")
	}
	if err := quietSystem(); err != nil {
		panic("gittest: " + err.Error())
	}
	for k, v := range quiet {
		if err := os.Setenv(k, v); err != nil {
			panic("gittest: " + err.Error())
		}
	}
}

func alreadyQuiet() bool {
	for k, v := range quiet {
		if os.Getenv(k) != v {
			return false
		}
	}
	return true
}

// quietSystem points GIT_CONFIG_SYSTEM at a file that includes the system
// config git would have read, then adds remoteKeys. The include keeps what the
// host's system config sets (core.autocrlf on Git for Windows) in force.
func quietSystem() error {
	if os.Getenv("GIT_CONFIG_NOSYSTEM") != "" {
		return errors.New("GIT_CONFIG_NOSYSTEM is set, so git would ignore the system config that quiets a remote")
	}
	out, err := exec.Command("git", "var", systemVar).Output()
	if errors.Is(err, exec.ErrNotFound) {
		return nil // no git, so nothing to quiet; the tests that need git skip
	}
	if err != nil {
		return errors.New("git var " + systemVar + ": " + err.Error())
	}
	current := strings.TrimSpace(string(out))
	if strings.HasPrefix(filepath.Base(current), filePrefix) {
		return nil
	}
	// Quoted, so a `#` or `;` in the path is not read as a comment.
	quoted := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(filepath.ToSlash(current))
	body := "[include]\n\tpath = \"" + quoted + "\"\n" + remoteKeys
	path, err := writeOnce(body)
	if err != nil {
		return err
	}
	return os.Setenv(systemVar, path)
}

// writeOnce writes body to a file named by its hash, so concurrent test
// packages share one file and a rename makes each write atomic. The file lives
// in a directory only this user can write: git runs what a config names (an
// fsmonitor, an sshCommand), and a predictable name in a shared /tmp could be
// planted by another user first.
func writeOnce(body string) (string, error) {
	dir, err := privateDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(body))
	path := filepath.Join(dir, filePrefix+hex.EncodeToString(sum[:6])+".gitconfig")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	tmp, err := os.CreateTemp(dir, filePrefix+"*.tmp")
	if err != nil {
		return "", err
	}
	_, werr := tmp.WriteString(body)
	if err := errors.Join(werr, tmp.Close()); err != nil {
		return "", errors.Join(err, os.Remove(tmp.Name()))
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		// Another package won the race; its file holds the same bytes. Windows
		// refuses a rename onto a file another process has open.
		if _, serr := os.Stat(path); serr == nil {
			return path, os.Remove(tmp.Name())
		}
		return "", errors.Join(err, os.Remove(tmp.Name()))
	}
	return path, nil
}

// privateDir is this user's cache directory for gittest, created 0700. Without
// a cache directory (no HOME), a fresh private temp directory stands in.
func privateDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return os.MkdirTemp("", filePrefix)
	}
	dir := filepath.Join(base, "dotf-gittest")
	return dir, os.MkdirAll(dir, 0o700)
}
