package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/prland"
)

func TestPrLand_ARefusalListsTheReasonsAndExitsNonZero(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	saved := prLandOptions
	prLandOptions = func(repo, _ string) prland.Options {
		return prland.Options{
			Repo: repo,
			Run: func(_ context.Context, args ...string) ([]byte, error) {
				if args[1] == "view" {
					return []byte(`{"state":"OPEN","isDraft":false,"headRefOid":"abc1234567","headRefName":"feat/x","baseRefName":"main","mergeStateStatus":"BEHIND"}`), nil
				}
				return []byte(`[{"name":"test","bucket":"pass"}]`), nil
			},
			Untriaged: func(context.Context) ([]int, error) { return nil, nil },
		}
	}
	t.Cleanup(func() { prLandOptions = saved })

	stdout, _, err := execute(t, "pr", "land", "#30", "--repo", "o/r")
	if err == nil {
		t.Fatal("a refusal exited 0")
	}
	if !strings.Contains(stdout, "[NOT MERGED] #30 at abc1234") || !strings.Contains(stdout, "merge state is BEHIND") {
		t.Errorf("refusal output:\n%s", stdout)
	}
}

func TestPrLand_RejectsANonNumber(t *testing.T) {
	if _, _, err := execute(t, "pr", "land", "feat/x"); err == nil {
		t.Fatal("a non-number was accepted")
	}
}

// 0 and negative numbers parse as integers but name no PR; they must be refused
// when the arguments are read, before the lock is taken or gh is asked.
func TestParseLandNumbers_RejectsANonPositiveNumber(t *testing.T) {
	for _, a := range []string{"0", "-5", "#0"} {
		if _, err := parseLandNumbers([]string{a}); err == nil {
			t.Errorf("%q was accepted as a PR number", a)
		}
	}
	if got, err := parseLandNumbers([]string{"#7", "8"}); err != nil || len(got) != 2 {
		t.Errorf("valid numbers = %v, %v", got, err)
	}
}

// queueFake answers gh for a queue of PRs: a PR with no entry in states is
// CLEAN and green, and the ones in broken fail to be read at all. merges lists
// the PRs merged, in order.
type queueFake struct {
	states map[string]string
	broken map[string]bool
	merges []string
	calls  int
}

func (q *queueFake) options(repo, _ string) prland.Options {
	return prland.Options{
		Repo: repo,
		Run: func(_ context.Context, args ...string) ([]byte, error) {
			q.calls++
			line := strings.Join(args, " ")
			f := strings.Fields(line)
			switch {
			case strings.HasPrefix(line, "pr view"):
				if q.broken[f[2]] {
					return nil, errors.New("HTTP 502")
				}
				state := "CLEAN"
				if s, ok := q.states[f[2]]; ok {
					state = s
				}
				return fmt.Appendf(nil, `{"state":"OPEN","isDraft":false,"headRefOid":"sha%s0000","headRefName":"feat/%s","baseRefName":"main","mergeStateStatus":%q}`, f[2], f[2], state), nil
			case strings.HasPrefix(line, "pr checks"):
				return []byte(`[{"name":"test","bucket":"pass"}]`), nil
			case strings.HasPrefix(line, "pr list"):
				return []byte(`[]`), nil
			case strings.HasPrefix(line, "pr merge"):
				q.merges = append(q.merges, f[2])
				return nil, nil
			case strings.HasPrefix(line, "api "):
				return []byte(`{"type":"file"}`), nil // the repository opts in
			}
			return nil, errors.New("unexpected gh call: " + line)
		},
		Untriaged: func(context.Context) ([]int, error) { return nil, nil },
	}
}

// useQueue replaces the GitHub wiring with q and points the state dir at a
// fresh temporary directory, which it returns.
func useQueue(t *testing.T, q *queueFake) string {
	t.Helper()
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	saved := prLandOptions
	prLandOptions = q.options
	t.Cleanup(func() { prLandOptions = saved })
	return filepath.Join(state, "dotfiles", "pr-land")
}

func TestPrLand_AQueueLandsInOrderAndSummarisesWithExitOne(t *testing.T) {
	q := &queueFake{states: map[string]string{"12": "DIRTY"}}
	useQueue(t, q)

	stdout, _, err := execute(t, "pr", "land", "30", "12", "31", "--repo", "o/r")

	if !errors.Is(err, errNotLanded) {
		t.Fatalf("a queue with a PR left behind returned %v, want errNotLanded (exit 1)", err)
	}
	if got := strings.Join(q.merges, " "); got != "30 31" {
		t.Errorf("merged %q, want the PRs that can land, in the order given: 30 31", got)
	}
	for _, want := range []string{
		"[MERGED] #30", "[NOT MERGED] #12", "[MERGED] #31",
		"summary: 2 merged, 1 not merged",
		"merged: #30, #31",
		"#12 at sha1200:",
		"  - merge state is DIRTY",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
	// the summary closes the output; the per-PR lines come first
	if strings.Index(stdout, "[MERGED] #31") > strings.Index(stdout, "summary:") {
		t.Errorf("the summary is not last:\n%s", stdout)
	}
}

func TestPrLand_AQueueThatAllMergesExitsZero(t *testing.T) {
	q := &queueFake{}
	useQueue(t, q)

	stdout, _, err := execute(t, "pr", "land", "30", "31", "--repo", "o/r")

	if err != nil {
		t.Fatalf("a queue that all merged returned %v", err)
	}
	if !strings.Contains(stdout, "summary: 2 merged, 0 not merged") || strings.Contains(stdout, "not merged:") {
		t.Errorf("summary:\n%s", stdout)
	}
}

func TestPrLand_OnePRPrintsNoSummary(t *testing.T) {
	q := &queueFake{}
	useQueue(t, q)

	stdout, _, err := execute(t, "pr", "land", "30", "--repo", "o/r")

	if err != nil || stdout != "[MERGED] #30 at sha3000\n" {
		t.Errorf("one PR: err %v, output %q", err, stdout)
	}
}

func TestPrLand_RejectsADuplicateNumber(t *testing.T) {
	q := &queueFake{}
	useQueue(t, q)
	if _, _, err := execute(t, "pr", "land", "30", "#30", "--repo", "o/r"); err == nil || q.calls != 0 {
		t.Fatalf("a PR listed twice: err %v, gh calls %d", err, q.calls)
	}
}

func TestPrLand_ASecondLandRefusesWhileTheLockIsHeld(t *testing.T) {
	q := &queueFake{}
	dir := useQueue(t, q)
	release, err := prland.AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	_, stderr, err := execute(t, "pr", "land", "30", "31", "--repo", "o/r")

	if err == nil || !errors.Is(err, prland.ErrLandRunning) {
		t.Fatalf("a second lander returned %v, want a refusal", err)
	}
	if !strings.Contains(stderr, fmt.Sprint(os.Getpid())) {
		t.Errorf("the refusal does not name the holder's PID %d: %q", os.Getpid(), stderr)
	}
	if q.calls != 0 {
		t.Errorf("a refused lander still called gh %d times", q.calls)
	}
}

func TestPrLand_AStaleLockIsTakenOverWithANote(t *testing.T) {
	q := &queueFake{}
	dir := useQueue(t, q)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// a PID no live process has: past every platform's PID range
	if err := os.WriteFile(filepath.Join(dir, "o+r.pid"), []byte("2147483646\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := execute(t, "pr", "land", "30", "--repo", "o/r")

	if err != nil || !strings.Contains(stdout, "[MERGED] #30") {
		t.Fatalf("a stale lock stopped the landing: %v\n%s", err, stdout)
	}
	if !strings.Contains(stderr, "taking over the lock of PID 2147483646") {
		t.Errorf("the take-over is not visible: %q", stderr)
	}
}

func TestPrLand_ReleasesTheLockAfterAnError(t *testing.T) {
	q := &queueFake{broken: map[string]bool{"30": true}}
	dir := useQueue(t, q)

	if _, _, err := execute(t, "pr", "land", "30", "--repo", "o/r"); err == nil {
		t.Fatal("an unreadable PR exited 0")
	}

	release, err := prland.AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("the lock outlived the command: %v", err)
	}
	release()
}

func TestPrLand_ReleasesTheLockAfterAQueue(t *testing.T) {
	q := &queueFake{broken: map[string]bool{"30": true}}
	dir := useQueue(t, q)

	if _, _, err := execute(t, "pr", "land", "30", "31", "--repo", "o/r"); !errors.Is(err, errNotLanded) {
		t.Fatalf("got %v", err)
	}

	release, err := prland.AcquireLock(dir, "o/r", func(int) bool { return true }, new(bytes.Buffer))
	if err != nil {
		t.Fatalf("the lock outlived the command: %v", err)
	}
	release()
}

func TestPrLand_AnUnreadablePRHasNoHeadToName(t *testing.T) {
	q := &queueFake{broken: map[string]bool{"12": true}}
	useQueue(t, q)

	stdout, _, _ := execute(t, "pr", "land", "12", "31", "--repo", "o/r")

	if !strings.Contains(stdout, "[NOT MERGED] #12:\n") || !strings.Contains(stdout, "    #12:\n") || !strings.Contains(stdout, "error: gh pr view: HTTP 502") {
		t.Errorf("output:\n%s", stdout)
	}
}

func TestPrLand_UnknownWaitBoundsTheWaitOnAnUncomputedMergeState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	saved := prLandOptions
	var slept time.Duration
	prLandOptions = func(repo, _ string) prland.Options {
		return prland.Options{
			Repo: repo,
			Run: func(_ context.Context, args ...string) ([]byte, error) {
				if args[1] == "view" {
					return []byte(`{"state":"OPEN","isDraft":false,"headRefOid":"abc1234567","headRefName":"feat/x","baseRefName":"main","mergeStateStatus":"UNKNOWN"}`), nil
				}
				return []byte(`[{"name":"test","bucket":"pass"}]`), nil
			},
			Untriaged: func(context.Context) ([]int, error) { return nil, nil },
			Sleep:     func(d time.Duration) { slept += d },
		}
	}
	t.Cleanup(func() { prLandOptions = saved })

	stdout, _, err := execute(t, "pr", "land", "30", "--repo", "o/r", "--wait", "--unknown-wait", "2m")
	if err == nil {
		t.Fatal("a refusal exited 0")
	}
	if slept != 2*time.Minute || !strings.Contains(stdout, "after waiting 2m0s") {
		t.Errorf("slept %s, output:\n%s", slept, stdout)
	}
}
