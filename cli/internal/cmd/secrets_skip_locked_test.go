package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// lockedBW fails every read the way lockHintReader does on a locked vault, so
// the error reaches resolveOrSkipLocked through resolve.go's real wrapping.
type lockedBW struct{}

func (lockedBW) Field(item, field string) (string, error) {
	return "", fmt.Errorf("%w: no bw serve daemon is running: Vault is locked.", secrets.ErrBWVaultLocked)
}

var fixedNow = time.Date(2026, 10, 30, 22, 54, 0, 0, time.UTC)

const skipLockedRegistry = "version: 1\nsecrets:\n" +
	"  - {id: nan-api-key, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: NAN_API_KEY}}\n"

// useSkipSeams runs the test as if inside unit (or outside any unit when ""),
// with a temporary runtime dir and a systemctl that records its calls.
func useSkipSeams(t *testing.T, unit string, restartErr error) (dir string, restarts *[]string) {
	t.Helper()
	dir = filepath.Join(t.TempDir(), "skipped-secrets")
	calls := []string{}
	oldCg, oldDir, oldCtl, oldHas, oldPID := selfCgroup, skippedSecretsDir, systemctlUser, hasSystemctl, unitMainPID
	unitMainPID = func(string) (int, error) { return os.Getpid(), nil }
	selfCgroup = func() (string, error) {
		return "0::/user.slice/user-1000.slice/user@1000.service/app.slice/" + unit + "\n", nil
	}
	skippedSecretsDir = func() string { return dir }
	systemctlUser = func(args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		return restartErr
	}
	hasSystemctl = func() bool { return true }
	t.Cleanup(func() {
		selfCgroup, skippedSecretsDir, systemctlUser, hasSystemctl, unitMainPID = oldCg, oldDir, oldCtl, oldHas, oldPID
	})
	return dir, &calls
}

func resolveNan(t *testing.T, skip bool) ([]string, string, error) {
	t.Helper()
	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	sel, err := resolveOnly(reg, "NAN_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	injected, err := resolveOrSkipLocked(reg, sel, skip, &errOut)
	return injected, errOut.String(), err
}

func TestSkipLocked_LockedVaultRunsWithoutTheSecretsAndMarksTheUnit(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	dir, _ := useSkipSeams(t, "hive.service", nil)

	injected, warning, err := resolveNan(t, true)
	if err != nil || injected != nil {
		t.Fatalf("got %v, %v; want no secrets and no error", injected, err)
	}
	if !strings.Contains(warning, "without NAN_API_KEY") || !strings.Contains(warning, "restarts hive.service") {
		t.Errorf("warning does not name the skipped secret and the restart: %q", warning)
	}
	if units, _ := secrets.DegradedUnits(dir); units["hive.service"] == "" {
		t.Errorf("no trace for hive.service: %v", units)
	}
}

func TestSkipLocked_WithoutTheFlagALockedVaultStillRefuses(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	dir, _ := useSkipSeams(t, "hive.service", nil)

	if _, _, err := resolveNan(t, false); !errors.Is(err, secrets.ErrBWVaultLocked) {
		t.Fatalf("got %v, want the locked-vault error", err)
	}
	if units, _ := secrets.DegradedUnits(dir); len(units) != 0 {
		t.Errorf("a refused run left a trace: %v", units)
	}
}

// Only a locked vault is skipped: a typo, a wrong key or an empty value still
// refuses, or the flag would launch a daemon unauthenticated with no signal.
func TestSkipLocked_AnyOtherFailureStillRefuses(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, fakeBW{})
	useSkipSeams(t, "hive.service", nil)

	if _, _, err := resolveNan(t, true); err == nil || errors.Is(err, secrets.ErrBWVaultLocked) {
		t.Fatalf("got %v, want the missing-item error to refuse", err)
	}
}

func TestSkipLocked_OutsideAUnitLeavesNoTrace(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	dir, _ := useSkipSeams(t, "session-3.scope", nil)

	if _, warning, err := resolveNan(t, true); err != nil || strings.Contains(warning, "restarts") {
		t.Fatalf("got %q, %v", warning, err)
	}
	if units, _ := secrets.DegradedUnits(dir); len(units) != 0 {
		t.Errorf("a shell run left a trace: %v", units)
	}
}

// A shell inside a terminal that runs as a service is in that service's cgroup
// too. It must leave no trace, or the unlock would restart the terminal.
func TestSkipLocked_ARunThatIsNotTheUnitsMainProcessLeavesNoTrace(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	dir, _ := useSkipSeams(t, "app-terminal.service", nil)
	unitMainPID = func(string) (int, error) { return os.Getpid() + 1, nil }

	if _, _, err := resolveNan(t, true); err != nil {
		t.Fatal(err)
	}
	if units, _ := secrets.DegradedUnits(dir); len(units) != 0 {
		t.Errorf("a child of the unit left a trace for it: %v", units)
	}
}

func TestSkipLocked_AResolvedRunClearsItsUnitsTrace(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, fakeBW{"nan-api-key/api-key": "synthetic-value"})
	dir, _ := useSkipSeams(t, "hive.service", nil)
	if err := secrets.MarkDegraded(dir, "hive.service", fixedNow); err != nil {
		t.Fatal(err)
	}

	injected, _, err := resolveNan(t, true)
	if err != nil || len(injected) != 1 {
		t.Fatalf("got %v, %v", injected, err)
	}
	if units, _ := secrets.DegradedUnits(dir); len(units) != 0 {
		t.Errorf("the trace survived a resolved run: %v", units)
	}
}

func TestSkipLocked_TheRunCommandLaunchesTheChild(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	useSkipSeams(t, "hive.service", nil)

	c := newSecretsRunCmd()
	c.SetArgs([]string{"--only", "NAN_API_KEY", "--skip-locked", "--", "true"})
	c.SetOut(io.Discard)
	var errOut bytes.Buffer
	c.SetErr(&errOut)
	if err := c.Execute(); err != nil {
		t.Fatalf("run --skip-locked on a locked vault: %v", err)
	}
	if !strings.Contains(errOut.String(), "Bitwarden is locked") {
		t.Errorf("no warning on stderr: %q", errOut.String())
	}
}

func TestRestartSkippedUnits_RestartsAndClears(t *testing.T) {
	dir, restarts := useSkipSeams(t, "", nil)
	if err := secrets.MarkDegraded(dir, "hive.service", fixedNow); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	restartSkippedUnits(&out)

	if len(*restarts) != 1 || (*restarts)[0] != "try-restart hive.service" {
		t.Fatalf("systemctl calls: %v", *restarts)
	}
	if !strings.Contains(out.String(), "restarted hive.service") {
		t.Errorf("output: %q", out.String())
	}
	if units, _ := secrets.DegradedUnits(dir); len(units) != 0 {
		t.Errorf("trace kept after a restart: %v", units)
	}
}

func TestRestartSkippedUnits_AFailedRestartKeepsTheTraceAndSaysSo(t *testing.T) {
	dir, _ := useSkipSeams(t, "", errors.New("exit status 5"))
	if err := secrets.MarkDegraded(dir, "hive.service", fixedNow); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	restartSkippedUnits(&out)

	if !strings.Contains(out.String(), "systemctl --user restart hive.service") {
		t.Errorf("output does not name the manual restart: %q", out.String())
	}
	if units, _ := secrets.DegradedUnits(dir); units["hive.service"] == "" {
		t.Errorf("trace dropped although the restart failed: %v", units)
	}
}

// The unlock path is wired: an unlock restarts what waited for it.
func TestSecretsUnlock_RestartsUnitsThatSkippedTheirSecrets(t *testing.T) {
	status := "unlocked"
	withFakeDaemon(t, &status, "")
	dir, restarts := useSkipSeams(t, "", nil)
	if err := secrets.MarkDegraded(dir, "hive.service", fixedNow); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runUnlock(t); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if len(*restarts) != 1 {
		t.Fatalf("systemctl calls: %v", *restarts)
	}
}

// A failed sync still fails the unlock, but the vault is unlocked, so the
// units waiting for it restart anyway: leaving them keyless until a second
// unlock is the state #3005 removes.
func TestSecretsUnlock_AFailedSyncStillRestartsWaitingUnits(t *testing.T) {
	status := "unlocked"
	withFakeDaemonSync(t, &status, "", &fakeSync{fail: true})
	dir, restarts := useSkipSeams(t, "", nil)
	if err := secrets.MarkDegraded(dir, "hive.service", fixedNow); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runUnlock(t); err == nil {
		t.Fatal("a failed sync must still fail the unlock")
	}
	if len(*restarts) != 1 {
		t.Fatalf("systemctl calls: %v", *restarts)
	}
}

// Inside a unit whose main process cannot be confirmed, the run cannot leave a
// trace, so it says so instead of letting doctor pass a keyless daemon.
func TestSkipLocked_AnUnconfirmedUnitWarnsThatNothingWillRestartIt(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	useSkipSeams(t, "hive.service", nil)
	unitMainPID = func(string) (int, error) { return 0, errors.New("no user bus") }

	_, warning, err := resolveNan(t, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warning, "no user bus") || !strings.Contains(warning, "systemctl --user restart hive.service") {
		t.Errorf("warning does not name the failure and the manual restart: %q", warning)
	}
}

func TestSkipLocked_ATraceThatCannotBeWrittenIsReported(t *testing.T) {
	useTempRegistry(t, skipLockedRegistry)
	useBwReader(t, lockedBW{})
	useSkipSeams(t, "hive.service", nil)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	skippedSecretsDir = func() string { return filepath.Join(blocker, "skipped-secrets") }

	_, warning, err := resolveNan(t, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warning, "could not record") || !strings.Contains(warning, "systemctl --user restart hive.service") {
		t.Errorf("warning does not report the lost trace: %q", warning)
	}
}
