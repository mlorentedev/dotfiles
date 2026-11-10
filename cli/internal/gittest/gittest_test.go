package gittest

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// cleanEnv clears every GIT_CONFIG_* variable Quiet reads or writes, restoring
// them after the test: a harness may pin its own pairs there, and a test that
// asserts what Quiet wrote needs a known starting point.
func cleanEnv(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(name, "GIT_CONFIG_") {
			continue
		}
		t.Setenv(name, "") // registers the restore of the caller's value
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
}

// quietVars names every variable Quiet writes for n pairs.
func quietVars(n int) []string {
	vars := []string{countVar}
	for i := 0; i < n; i++ {
		vars = append(vars, keyVar+strconv.Itoa(i), valueVar+strconv.Itoa(i))
	}
	return vars
}

// gitConfig reads key in dir through a git started with env, or with the
// process environment when env is nil.
func gitConfig(t *testing.T, dir, key string, env []string) string {
	t.Helper()
	c := exec.Command("git", "-C", dir, "config", key)
	c.Env = env
	out, err := c.Output()
	if err != nil {
		t.Fatalf("git config %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}

// TestQuietTurnsOffAutoGCForEveryGitTheProcessStarts — the keys must reach a
// git that no fixture helper builds, so assert them through a plain exec.Command
// that only inherits the environment.
func TestQuietTurnsOffAutoGCForEveryGitTheProcessStarts(t *testing.T) {
	cleanEnv(t)
	Quiet()
	for key, want := range map[string]string{"gc.auto": "0", "maintenance.auto": "false"} {
		if got := gitConfig(t, t.TempDir(), key, nil); got != want {
			t.Errorf("git config %s = %q, want %q", key, got, want)
		}
	}
}

// TestQuietReachesTheRemoteSideOfALocalTransport — a local push starts the
// remote's receive-pack with GIT_CONFIG_COUNT unset (#3031), so assert the keys
// through exactly that environment.
func TestQuietReachesTheRemoteSideOfALocalTransport(t *testing.T) {
	cleanEnv(t)
	Quiet()
	var remote []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, countVar+"=") {
			remote = append(remote, kv)
		}
	}
	for key, want := range map[string]string{"receive.autogc": "false", "gc.auto": "0", "maintenance.auto": "false"} {
		if got := gitConfig(t, t.TempDir(), key, remote); got != want {
			t.Errorf("git config %s on the remote side = %q, want %q", key, got, want)
		}
	}
}

// TestQuietKeepsTheSystemConfigInForce — replacing the system config would drop
// what the host sets there, such as core.autocrlf on Git for Windows.
func TestQuietKeepsTheSystemConfigInForce(t *testing.T) {
	cleanEnv(t)
	host := filepath.Join(t.TempDir(), "sys#tem;config") // comment characters git must not cut at
	if err := os.WriteFile(host, []byte("[gittestdemo]\n\tkept = yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(systemVar, host)
	Quiet()
	// Its name hashes this test's own temp path, so no other run shares it.
	written := os.Getenv(systemVar)
	t.Cleanup(func() { _ = os.Remove(written) })
	if got := gitConfig(t, t.TempDir(), "gittestdemo.kept", nil); got != "yes" {
		t.Errorf("the host's system config was dropped: gittestdemo.kept = %q", got)
	}
}

// TestQuietKeepsTheCallersConfig — a harness pins config through the same
// variables (Copilot CLI exports safe.bareRepository, credential.interactive
// and core.fsmonitor). Overwriting them would drop keys the harness relies on,
// and refusing would abort the whole test binary, which is how #3253 surfaced.
func TestQuietKeepsTheCallersConfig(t *testing.T) {
	cleanEnv(t)
	t.Setenv(countVar, "1")
	t.Setenv(keyVar+"0", "gittestdemo.caller")
	t.Setenv(valueVar+"0", "kept")
	Quiet()
	if got := gitConfig(t, t.TempDir(), "gittestdemo.caller", nil); got != "kept" {
		t.Errorf("the caller's pair was dropped: gittestdemo.caller = %q", got)
	}
	if got := gitConfig(t, t.TempDir(), "gc.auto", nil); got != "0" {
		t.Errorf("gc.auto = %q, want 0: Quiet did not append after the caller's pair", got)
	}
	if got, want := os.Getenv(countVar), strconv.Itoa(1+len(quietPairs)); got != want {
		t.Errorf("%s = %q, want %q (1 caller pair + %d appended)", countVar, got, want, len(quietPairs))
	}
}

// TestQuietAppendsAfterAForeignEnvironment — the shape that aborted the test
// binary: three pairs the harness pinned, none of them ours.
func TestQuietAppendsAfterAForeignEnvironment(t *testing.T) {
	cleanEnv(t)
	foreign := [][2]string{
		{"safe.bareRepository", "explicit"},
		{"credential.interactive", "never"},
		{"core.fsmonitor", "false"},
	}
	t.Setenv(countVar, strconv.Itoa(len(foreign)))
	for i, pair := range foreign {
		t.Setenv(keyVar+strconv.Itoa(i), pair[0])
		t.Setenv(valueVar+strconv.Itoa(i), pair[1])
	}
	Quiet()
	if got, want := os.Getenv(countVar), strconv.Itoa(len(foreign)+len(quietPairs)); got != want {
		t.Errorf("%s = %q, want %q", countVar, got, want)
	}
	for _, pair := range foreign {
		if got := gitConfig(t, t.TempDir(), pair[0], nil); got != pair[1] {
			t.Errorf("%s = %q, want %q: the harness pair was dropped", pair[0], got, pair[1])
		}
	}
	for _, pair := range quietPairs {
		if got := gitConfig(t, t.TempDir(), pair[0], nil); got != pair[1] {
			t.Errorf("%s = %q, want %q: Quiet's pair does not outrank the caller's", pair[0], got, pair[1])
		}
	}
}

// TestQuietOutranksAConflictingCallerPair — the precedence the spec claims in
// Risks, and the only case where "applies on top" is observable: a caller pair
// on a key Quiet also sets. git applies the pairs in index order, so the pair
// Quiet appends last must be the one a child reads. A caller pair on a key
// Quiet does not set (TestQuietAppendsAfterAForeignEnvironment) cannot show
// this — where the keys differ, "on top" has no effect to see.
func TestQuietOutranksAConflictingCallerPair(t *testing.T) {
	cleanEnv(t)
	conflicting := [][2]string{
		{"gc.auto", "1"},
		{"maintenance.auto", "true"},
	}
	t.Setenv(countVar, strconv.Itoa(len(conflicting)))
	for i, pair := range conflicting {
		t.Setenv(keyVar+strconv.Itoa(i), pair[0])
		t.Setenv(valueVar+strconv.Itoa(i), pair[1])
	}
	Quiet()
	for _, pair := range quietPairs {
		if got := gitConfig(t, t.TempDir(), pair[0], nil); got != pair[1] {
			t.Errorf("%s = %q, want %q: the caller's conflicting pair outranked Quiet's", pair[0], got, pair[1])
		}
	}
	if got, want := os.Getenv(countVar), strconv.Itoa(len(conflicting)+len(quietPairs)); got != want {
		t.Errorf("%s = %q, want %q (%d caller pairs + %d appended)", countVar, got, want, len(conflicting), len(quietPairs))
	}
}

// TestQuietOutranksACallerThatSetsTheSameKeyTwice — git keeps the last pair for
// a key, so a caller that pins gc.auto twice ends on its second value. A check
// for "the pair appears anywhere" would read the caller's first pair, decide it
// already had what Quiet wants, and leave the second one in force — the exact
// condition #2803 exists to prevent.
func TestQuietOutranksACallerThatSetsTheSameKeyTwice(t *testing.T) {
	cleanEnv(t)
	repinned := [][2]string{
		{"gc.auto", "0"}, // already what Quiet wants, so a naive check skips it
		{"gc.auto", "1"}, // ...and this is what git would then resolve
	}
	t.Setenv(countVar, strconv.Itoa(len(repinned)))
	for i, pair := range repinned {
		t.Setenv(keyVar+strconv.Itoa(i), pair[0])
		t.Setenv(valueVar+strconv.Itoa(i), pair[1])
	}
	Quiet()
	if got, want := gitConfig(t, t.TempDir(), "gc.auto", nil), "0"; got != want {
		t.Errorf("gc.auto = %q, want %q: the caller's last pair outranked Quiet's", got, want)
	}
	if got, want := os.Getenv(countVar), strconv.Itoa(len(repinned)+len(quietPairs)); got != want {
		t.Errorf("%s = %q, want %q (%d caller pairs + %d appended)", countVar, got, want, len(repinned), len(quietPairs))
	}
}

// TestQuietRefusesBeforeItTouchesTheEnvironment — a count Quiet cannot use is
// refused before quietSystem writes a file and points GIT_CONFIG_SYSTEM at it,
// so a refused call leaves the environment as it found it. The count is
// caller-controlled: one that promises pairs the environment does not carry is
// refused the way git refuses it, and a huge one costs nothing to reject.
func TestQuietRefusesBeforeItTouchesTheEnvironment(t *testing.T) {
	for name, count := range map[string]string{
		"malformed count": "abc",
		"missing pair":    "1",
		"huge count":      "1000000000",
	} {
		t.Run(name, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv(countVar, count)
			before := os.Getenv(systemVar)
			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Fatalf("Quiet accepted %s=%q", countVar, count)
					}
					// Naming the variable is what makes this the count refusal
					// and not any other panic on the way in.
					if msg, ok := r.(string); !ok || !strings.Contains(msg, countVar) {
						t.Errorf("panic = %v, want a string naming %s: it did not come from the count refusal", r, countVar)
					}
				}()
				Quiet()
			}()
			if got := os.Getenv(systemVar); got != before {
				t.Errorf("%s = %q after a refused Quiet, want %q: the refusal is not side-effect free", systemVar, got, before)
			}
		})
	}
}

// TestQuietRefusesAMalformedCount — git reads the same number, so a count Quiet
// cannot parse is a failure to surface, not a value to guess at: appending at a
// guessed index would silently drop the pairs the caller meant to set.
func TestQuietRefusesAMalformedCount(t *testing.T) {
	for _, raw := range []string{"abc", "-1"} {
		t.Run(raw, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv(countVar, raw)
			defer func() {
				if recover() == nil {
					t.Fatalf("Quiet accepted %s=%q, which git cannot read either", countVar, raw)
				}
			}()
			Quiet()
		})
	}
}

// gitEnv is the process environment with every GIT_CONFIG_* variable dropped,
// then the count form under test and one pair for git to read.
func gitEnv(raw string) []string {
	env := make([]string, 0, len(os.Environ())+3)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_CONFIG_") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, countVar+"="+raw, keyVar+"0=gittestdemo.caller", valueVar+"0=kept")
}

// gitReadsCount reports whether git reads a count form without failing. A form
// git reads as zero finds no key and exits 1, which is still a read rather than
// a refusal, so only an exit past 1 means git rejected the count.
func gitReadsCount(t *testing.T, raw string) bool {
	t.Helper()
	c := exec.Command("git", "config", "--get-all", "gittestdemo.caller")
	c.Env = gitEnv(raw)
	err := c.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode() == 1
	}
	return err == nil
}

// TestQuietReadsTheCountTheWayGitDoes — refusing means panicking, and that
// aborts the test binary, which is the #3253 symptom, so Quiet may only refuse
// a count git refuses too. git treats an empty value as no pairs, skips leading
// whitespace, and then reads one decimal integer that has to end the value.
// Assert the two agree form by form rather than against a table that could
// drift away from git.
func TestQuietReadsTheCountTheWayGitDoes(t *testing.T) {
	forms := []string{"", "0", "1", " 1", "\t1", "+1", "01", "1 ", "-1", "0x1", " ", "abc", "1a"}
	for _, raw := range forms {
		t.Run(strconv.Quote(raw), func(t *testing.T) {
			cleanEnv(t)
			gitReads := gitReadsCount(t, raw)
			// The pair is complete, so the only thing left to refuse is the
			// count itself: a count git reads as one finds this pair, and a
			// count it reads as zero ignores it.
			t.Setenv(countVar, raw)
			t.Setenv(keyVar+"0", "gittestdemo.caller")
			t.Setenv(valueVar+"0", "kept")
			func() {
				defer func() {
					r := recover()
					if gitReads && r != nil {
						t.Errorf("Quiet refused %s=%s, which git reads: %v", countVar, strconv.Quote(raw), r)
					}
					if !gitReads && r == nil {
						t.Fatalf("Quiet accepted %s=%s, which git cannot read", countVar, strconv.Quote(raw))
					}
				}()
				Quiet()
			}()
		})
	}
}

// TestQuietNamesTheVariableThatIsMissing — the refusal exists so the operator
// can see the caller's bug, so it has to name the half of the pair that is
// actually absent rather than always the key.
func TestQuietNamesTheVariableThatIsMissing(t *testing.T) {
	for name, tc := range map[string]struct {
		setKey, setValue bool
		wantVar          string
	}{
		"key absent":   {setValue: true, wantVar: keyVar},
		"value absent": {setKey: true, wantVar: valueVar},
	} {
		t.Run(name, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv(countVar, "1")
			if tc.setKey {
				t.Setenv(keyVar+"0", "gittestdemo.caller")
			}
			if tc.setValue {
				t.Setenv(valueVar+"0", "kept")
			}
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("Quiet accepted an incomplete pair: %s0 set=%v, %s0 set=%v", keyVar, tc.setKey, valueVar, tc.setValue)
				}
				if msg, ok := r.(string); !ok || !strings.Contains(msg, tc.wantVar+"0") {
					t.Errorf("panic = %v, want a string naming %s0: it points at the wrong half of the pair", r, tc.wantVar)
				}
			}()
			Quiet()
		})
	}
}

// TestQuietRefusesNoSystem — git would skip the file that quiets a remote.
func TestQuietRefusesNoSystem(t *testing.T) {
	cleanEnv(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	defer func() {
		if recover() == nil {
			t.Fatal("Quiet accepted GIT_CONFIG_NOSYSTEM, which leaves remotes unquieted")
		}
	}()
	Quiet()
}

// TestQuietAcceptsItsOwnKeys — a test binary that re-executes itself as a
// helper process inherits them, and its TestMain calls Quiet again. A second
// call must add nothing rather than append duplicates on every generation.
func TestQuietAcceptsItsOwnKeys(t *testing.T) {
	cleanEnv(t)
	Quiet()
	Quiet()
	if got, want := os.Getenv(countVar), strconv.Itoa(len(quietPairs)); got != want {
		t.Errorf("%s = %q after a second Quiet, want %q", countVar, got, want)
	}
	if got := gitConfig(t, t.TempDir(), "maintenance.auto", nil); got != "false" {
		t.Errorf("maintenance.auto = %q, want false", got)
	}
}

// TestQuietDoesNotIncludeItsOwnFile — a re-executed binary whose COUNT keys
// were dropped still inherits GIT_CONFIG_SYSTEM; including that file in a new
// one would nest it on every generation.
func TestQuietDoesNotIncludeItsOwnFile(t *testing.T) {
	cleanEnv(t)
	Quiet()
	first := os.Getenv(systemVar)
	for _, k := range quietVars(len(quietPairs)) {
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	Quiet()
	if got := os.Getenv(systemVar); got != first {
		t.Errorf("a second Quiet wrote a new system config %q over its own %q", got, first)
	}
}
