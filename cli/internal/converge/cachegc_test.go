package converge

import (
	"errors"

	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
)

// gcFake is the machine behind the cache-gc step: which tools are on PATH,
// which commands fail, and a log of every command run.
type gcFake struct {
	uvBusy  bool // a running uv holds the cache lock
	present map[string]bool
	fail    map[string]error // joined argv -> error
	log     []string
	free    []uint64 // successive answers of the free-space probe
}

func newGCFake(tools ...string) *gcFake {
	f := &gcFake{present: map[string]bool{}, fail: map[string]error{}, free: []uint64{100 << 30, 103 << 30}}
	for _, t := range tools {
		f.present[t] = true
	}
	return f
}

func (f *gcFake) run(name string, args ...string) ([]byte, error) {
	argv := strings.Join(append([]string{name}, args...), " ")
	f.log = append(f.log, argv)
	if argv == "uv cache dir" {
		return []byte("/cache/uv\n"), nil
	}
	return nil, f.fail[argv]
}

func (f *gcFake) tryLock(path string) (func(), error) {
	if path != filepath.Join("/cache/uv", ".lock") {
		return nil, errors.New("unexpected lock path " + path)
	}
	if f.uvBusy {
		return nil, filelock.ErrLocked
	}
	return func() {}, nil
}

func (f *gcFake) has(name string) bool { return f.present[name] }

func (f *gcFake) freeSpace(string) (uint64, error) {
	v := f.free[0]
	if len(f.free) > 1 {
		f.free = f.free[1:]
	}
	return v, nil
}

var gcNow = time.Date(2026, 11, 12, 6, 0, 0, 0, time.UTC)

func (f *gcFake) step(state string) cacheGC {
	return cacheGC{
		run:      f.run,
		has:      f.has,
		free:     f.freeSpace,
		now:      func() time.Time { return gcNow },
		stateDir: func(Env) (string, error) { return state, nil },
		tryLock:  f.tryLock,
	}
}

func writeStamp(t *testing.T, state string, at time.Time) {
	t.Helper()
	p := filepath.Join(state, "cache-gc", "last-run")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(at.Format(time.RFC3339)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCacheGC_PlanWhenDueListsThePresentPrunersAndRunsNothing(t *testing.T) {
	state := t.TempDir()
	f := newGCFake("uv", "npm")
	res, err := f.step(state).Reconcile(Env{Home: state}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Opaque {
		t.Errorf("a due prune cannot say what it frees: want Opaque, got %+v", res)
	}
	if !strings.Contains(res.Detail, "uv cache prune") || !strings.Contains(res.Detail, "npm cache verify") {
		t.Errorf("plan detail should name the present pruners, got %q", res.Detail)
	}
	if strings.Contains(res.Detail, "pre-commit") {
		t.Errorf("an absent tool is not planned: %q", res.Detail)
	}
	if len(f.log) != 0 {
		t.Errorf("a plan runs nothing, ran %q", f.log)
	}
	if _, err := os.Stat(filepath.Join(state, "cache-gc", "last-run")); !os.IsNotExist(err) {
		t.Errorf("a plan writes no stamp: %v", err)
	}
}

func TestCacheGC_NotDueWithinAWeekRunsNothing(t *testing.T) {
	state := t.TempDir()
	writeStamp(t, state, gcNow.Add(-3*24*time.Hour))
	f := newGCFake("uv", "npm", "pre-commit")
	for _, dry := range []bool{true, false} {
		res, err := f.step(state).Reconcile(Env{Home: state}, dry)
		if err != nil {
			t.Fatal(err)
		}
		if res.Opaque || res.Changes != 0 || !strings.Contains(res.Detail, "not due") {
			t.Errorf("dry=%v: want converged and not due, got %+v", dry, res)
		}
	}
	if len(f.log) != 0 {
		t.Errorf("not due runs nothing, ran %q", f.log)
	}
}

func TestCacheGC_ApplyRunsEveryPresentPrunerInOrderAndWritesTheStamp(t *testing.T) {
	state := t.TempDir()
	writeStamp(t, state, gcNow.Add(-8*24*time.Hour))
	f := newGCFake("uv", "pre-commit", "npm", "docker", "limactl", "colima")
	res, err := f.step(state).Reconcile(Env{Home: state}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"uv cache dir",
		"uv cache prune",
		"pre-commit gc",
		"npm cache verify",
		"docker info",
		"docker builder prune -f --filter until=168h",
		"docker image prune -f --filter until=168h",
		"limactl prune --keep-referred",
		"colima status",
		"colima ssh -- sudo fstrim -a",
	}
	if !slices.Equal(f.log, want) {
		t.Errorf("commands:\n got %q\nwant %q", f.log, want)
	}
	if !res.Opaque || !strings.Contains(res.Detail, "3.0 GiB") {
		t.Errorf("apply reports Opaque with the free-space delta, got %+v", res)
	}
	b, err := os.ReadFile(filepath.Join(state, "cache-gc", "last-run"))
	if err != nil || strings.TrimSpace(string(b)) != gcNow.Format(time.RFC3339) {
		t.Errorf("stamp = %q, %v; want %s", b, err, gcNow.Format(time.RFC3339))
	}
	if err := f.step(state).Probe(Env{Home: state}); err != nil {
		t.Errorf("after an apply the step is not due: %v", err)
	}
}

// A disk short of space makes the step due whatever the stamp says: it is the
// reason the step exists, and doctor names this step as the remedy.
func TestCacheGC_LowDiskIsDueInsideTheWeek(t *testing.T) {
	state := t.TempDir()
	writeStamp(t, state, gcNow.Add(-24*time.Hour))
	f := newGCFake("npm")
	f.free = []uint64{5 << 30, 6 << 30}
	res, err := f.step(state).Reconcile(Env{Home: state}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(f.log, "npm cache verify") || !strings.Contains(res.Detail, "+1.0 GiB") {
		t.Errorf("low disk must run the pruners, got %q, %q", f.log, res.Detail)
	}
	f.free = []uint64{5 << 30}
	plan, _ := f.step(state).Reconcile(Env{Home: state}, true)
	if !plan.Opaque || !strings.Contains(plan.Detail, "under 20.0 GiB") {
		t.Errorf("the plan names the low disk as the reason, got %q", plan.Detail)
	}
}

// A daemon that does not answer is a skip of its pruners, never a failure: the
// step runs unattended, and Docker stopped is an ordinary state.
func TestCacheGC_DockerOrColimaNotRunningSkipsTheirPruners(t *testing.T) {
	state := t.TempDir()
	f := newGCFake("docker", "colima")
	f.fail["docker info"] = errors.New("Cannot connect to the Docker daemon")
	f.fail["colima status"] = errors.New("colima is not running")
	res, err := f.step(state).Reconcile(Env{Home: state}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.log, []string{"docker info", "colima status"}) {
		t.Errorf("only the gates run, got %q", f.log)
	}
	if !strings.Contains(res.Detail, "docker: not running") || !strings.Contains(res.Detail, "colima: not running") {
		t.Errorf("detail names the skipped daemons, got %q", res.Detail)
	}
}

// A running uvx holds uv's cache lock for as long as it runs, so a prune would
// wait out uv's lock timeout and fail. A held lock skips uv instead.
func TestCacheGC_AUVCacheInUseSkipsUVWithoutWaiting(t *testing.T) {
	state := t.TempDir()
	f := newGCFake("uv", "npm")
	f.uvBusy = true
	res, err := f.step(state).Reconcile(Env{Home: state}, false)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(f.log, "uv cache prune") {
		t.Errorf("a busy cache is not pruned, ran %q", f.log)
	}
	if !strings.Contains(res.Detail, "uv: cache in use") || !strings.Contains(res.Detail, "npm cache verify") {
		t.Errorf("detail names the skip and the rest still runs, got %q", res.Detail)
	}
}

// A pruner that fails does not stop the others, and leaves the stamp unwritten
// so the next daily run tries again instead of waiting a week.
func TestCacheGC_AFailingPrunerFailsTheStepAndKeepsItDue(t *testing.T) {
	state := t.TempDir()
	f := newGCFake("uv", "npm")
	f.fail["uv cache prune"] = errors.New("permission denied")
	_, err := f.step(state).Reconcile(Env{Home: state}, false)
	if err == nil || !strings.Contains(err.Error(), "uv cache prune") {
		t.Fatalf("want the failing pruner named, got %v", err)
	}
	if !slices.Contains(f.log, "npm cache verify") {
		t.Errorf("the other pruners still run, got %q", f.log)
	}
	if _, err := os.Stat(filepath.Join(state, "cache-gc", "last-run")); !os.IsNotExist(err) {
		t.Errorf("a failed run writes no stamp: %v", err)
	}
}

func TestCacheGC_NoToolPresentIsASkip(t *testing.T) {
	state := t.TempDir()
	res, err := newGCFake().step(state).Reconcile(Env{Home: state}, false)
	if err != nil || res.Skip == "" {
		t.Errorf("want a skip with a reason, got %+v, %v", res, err)
	}
}

// A stamp that cannot be read as a time makes the step due rather than never
// due: a corrupt file must not switch the maintenance off for good.
func TestCacheGC_AnUnreadableStampIsDue(t *testing.T) {
	state := t.TempDir()
	p := filepath.Join(state, "cache-gc", "last-run")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := newGCFake("uv").step(state).Reconcile(Env{Home: state}, true)
	if err != nil || !res.Opaque {
		t.Errorf("want due, got %+v, %v", res, err)
	}
}
