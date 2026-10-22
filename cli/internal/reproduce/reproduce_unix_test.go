//go:build unix

package reproduce

import (
	"context"
	"testing"
	"time"
)

// A check that leaves a child holding its output must still end at the
// timeout: killing the check does not close a pipe its child inherited.
func TestRunTimesOutPastAChildHoldingTheOutput(t *testing.T) {
	done := make(chan Result, 1)
	go func() {
		got, err := Run(context.Background(), Options{Dir: newRepo(t), Command: []string{"sh", "-c", "sleep 30 & sleep 30"}, Timeout: 100 * time.Millisecond})
		if err != nil {
			t.Errorf("Run: %v", err)
		}
		done <- got
	}()
	select {
	case got := <-done:
		if got.State != CouldNotRun {
			t.Errorf("State = %q, want %q", got.State, CouldNotRun)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Run waited on the child past the timeout")
	}
}
