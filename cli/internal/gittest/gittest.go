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
	"strconv"
	"strings"
)

const (
	countVar  = "GIT_CONFIG_COUNT"
	keyVar    = "GIT_CONFIG_KEY_"
	valueVar  = "GIT_CONFIG_VALUE_"
	systemVar = "GIT_CONFIG_SYSTEM"
	// filePrefix names the system config Quiet writes, so a re-executed test
	// binary recognises its parent's file instead of including it in itself.
	filePrefix = "gittest-"
)

// quietPairs are the keys Quiet appends to whatever the environment already
// carries. Appended, never overwritten: git applies the pairs in index order,
// so ours still win, while a harness that pins config through the same
// variables (Copilot CLI exports safe.bareRepository and friends) keeps its own
// keys instead of aborting the whole test binary (#3253).
var quietPairs = [][2]string{
	{"gc.auto", "0"},
	{"maintenance.auto", "false"},
}

// remoteKeys go in the system config Quiet points GIT_CONFIG_SYSTEM at. A
// local push or fetch starts the remote side with GIT_CONFIG_COUNT unset, so
// the keys above never reach the remote's receive-pack, which then runs
// `git maintenance run --auto --detach` (#3031). GIT_CONFIG_SYSTEM survives
// that transport.
const remoteKeys = "[gc]\n\tauto = 0\n[maintenance]\n\tauto = false\n[receive]\n\tautogc = false\n"

// Quiet turns off auto gc and auto maintenance for every git this process
// starts, the remote side of a local transport included. Call it from TestMain
// before m.Run. It appends its keys after any the environment already carries,
// and a test binary that re-executes itself inherits them, so a second call
// adds nothing.
func Quiet() {
	// Validated before quietSystem writes a file and points GIT_CONFIG_SYSTEM
	// at it, so a refused call leaves the environment as it found it.
	caller, err := callerPairs()
	if err != nil {
		panic("gittest: " + err.Error())
	}
	if err := quietSystem(); err != nil {
		panic("gittest: " + err.Error())
	}
	added := 0
	for _, pair := range quietPairs {
		// Skip only when the caller's last word on the key is already ours.
		// Skipping on "the pair appears anywhere" would let an earlier
		// identical pair hide a later caller override, which is the one shape
		// where our pair stops winning.
		if value, ok := lastValue(caller, pair[0]); ok && value == pair[1] {
			continue
		}
		if err := setPair(len(caller)+added, pair); err != nil {
			panic("gittest: " + err.Error())
		}
		added++
	}
	if added == 0 {
		return
	}
	if err := os.Setenv(countVar, strconv.Itoa(len(caller)+added)); err != nil {
		panic("gittest: " + err.Error())
	}
}

// callerPairs reads the pairs the environment already carries, so Quiet can
// append to them rather than drop them. A missing GIT_CONFIG_COUNT means none;
// a value git would not read is refused (parseCount), so a refusal here always
// means git would have failed on the same environment.
//
// Read one pair at a time rather than preallocating the count: the count is
// caller-controlled, so a huge one would ask for gigabytes before the first
// pair was looked at. Reading stops at the first pair the environment does not
// carry, which is also what git refuses (a count promising keys it cannot read).
func callerPairs() ([][2]string, error) {
	raw, ok := os.LookupEnv(countVar)
	if !ok {
		return nil, nil
	}
	n, err := parseCount(raw)
	if err != nil {
		return nil, err
	}
	var pairs [][2]string
	for i := 0; i < n; i++ {
		suffix := strconv.Itoa(i)
		key, hasKey := os.LookupEnv(keyVar + suffix)
		value, hasValue := os.LookupEnv(valueVar + suffix)
		// Name the half that is actually absent: the refusal exists so the
		// operator can see the caller's bug, and pointing at the pair that is
		// present sends them to fix the wrong one.
		if !hasKey {
			return nil, errors.New(countVar + " is " + raw + " but " + keyVar + suffix + " is not set")
		}
		if !hasValue {
			return nil, errors.New(countVar + " is " + raw + " but " + valueVar + suffix + " is not set")
		}
		pairs = append(pairs, [2]string{key, value})
	}
	return pairs, nil
}

// parseCount reads GIT_CONFIG_COUNT the way git does, so a count git accepts is
// never refused: refusing means panicking, which aborts the test binary that is
// the #3253 symptom. git treats an empty value as no pairs, skips leading
// whitespace, and then reads one decimal integer that has to end the value.
// Anything else is "bogus count" to git, and a negative or oversized value is
// refused as too many entries, which strconv reports as a sign or an overflow.
func parseCount(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.TrimLeft(raw, " \t\n\v\f\r"))
	if err != nil || n < 0 {
		return 0, errors.New(countVar + " is not a non-negative integer: " + raw)
	}
	return n, nil
}

// lastValue is the value git resolves for key from pairs, reporting false when
// pairs does not set key. git applies the pairs in index order and keeps the
// last, so only the final one decides what a child reads.
func lastValue(pairs [][2]string, key string) (string, bool) {
	for i := len(pairs) - 1; i >= 0; i-- {
		if pairs[i][0] == key {
			return pairs[i][1], true
		}
	}
	return "", false
}

// setPair writes one pair at index, which must be past every index the caller
// already uses.
func setPair(index int, pair [2]string) error {
	suffix := strconv.Itoa(index)
	if err := os.Setenv(keyVar+suffix, pair[0]); err != nil {
		return err
	}
	return os.Setenv(valueVar+suffix, pair[1])
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
