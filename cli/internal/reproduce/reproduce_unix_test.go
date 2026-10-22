//go:build unix

package reproduce

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A check that leaves a child holding its output must still end at the
// timeout: killing the check does not close a pipe its child inherited.
func TestRunTimesOutPastAChildHoldingTheOutput(t *testing.T) {
	// The repository is made on the test goroutine, and the goroutine only
	// sends: t.Fatal elsewhere would hang the test instead of failing it.
	dir := newRepo(t)
	type outcome struct {
		res Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		got, err := Run(context.Background(), Options{Dir: dir, Command: []string{"sh", "-c", "sleep 30 & sleep 30"}, Timeout: 100 * time.Millisecond})
		done <- outcome{got, err}
	}()
	select {
	case o := <-done:
		if o.err != nil {
			t.Fatalf("Run: %v", o.err)
		}
		if got := o.res; got.State != CouldNotRun {
			t.Errorf("State = %q, want %q", got.State, CouldNotRun)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Run waited on the child past the timeout")
	}
}

// A target that cannot be put back is an error, loud, never a result.
func TestRunFailsLoudlyWhenTheProbeCannotBeReverted(t *testing.T) {
	dir := newRepo(t)
	probe := probeFor(t, dir, "two\n")
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	_, err := Run(context.Background(), Options{Dir: dir, Command: []string{"chmod", "0555", dir}, Probe: probe})
	if !errors.Is(err, ErrNotReverted) {
		t.Fatalf("err = %v, want ErrNotReverted", err)
	}
}

// A restore that git reports as done but that does not bring the bytes
// back is an error too: here a smudge filter added after the checkout makes
// HEAD's version differ from the file that was found.
func TestRunFailsLoudlyWhenTheRestoredBytesDiffer(t *testing.T) {
	dir := newRepo(t)
	probe := probeFor(t, dir, "two\n")
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "attributes"), []byte("a.txt filter=upper\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "config", "filter.upper.smudge", "tr a-z A-Z")
	_, err := Run(context.Background(), Options{Dir: dir, Command: []string{"git", "--version"}, Probe: probe})
	if !errors.Is(err, ErrNotReverted) {
		t.Fatalf("err = %v, want ErrNotReverted", err)
	}
}
