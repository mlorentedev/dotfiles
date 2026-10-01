package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// #1884: two handoff-write runs at the same moment lost a thread. Each read
// MEMORY.md, computed its own update and renamed it over the file, so the last
// rename erased the other's thread, and both exited 0 having printed "wrote".
// The review measured it on the built binary: with 4 writers started together,
// 58 of 240 writes were lost.
//
// The writers here start together on purpose. Run one after another, they
// would pass with or without the lock, and so prove nothing.
func TestConcurrentWritesDoNotLoseAThread(t *testing.T) {
	const writers, rounds = 8, 5
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	for round := 0; round < rounds; round++ {
		memory := filepath.Join(t.TempDir(), "MEMORY.md")
		if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		start := make(chan struct{})
		errs := make([]error, writers)
		var wg sync.WaitGroup
		for w := 0; w < writers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				cmd := newMemCmd()
				cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", fmt.Sprintf("feat-w%d", w)})
				cmd.SetIn(bytes.NewBufferString(fmt.Sprintf("**Next action:** writer %d.\n", w)))
				cmd.SetOut(&bytes.Buffer{})
				cmd.SetErr(&bytes.Buffer{})
				<-start
				errs[w] = cmd.Execute()
			}(w)
		}
		close(start)
		wg.Wait()

		got, err := os.ReadFile(memory)
		if err != nil {
			t.Fatal(err)
		}
		for w := 0; w < writers; w++ {
			if errs[w] != nil {
				t.Errorf("round %d: writer %d failed: %v", round, w, errs[w])
			}
			// The writer exiting 0 is not the assertion: the lost update exited 0 too.
			if !strings.Contains(string(got), fmt.Sprintf("### thread: feat-w%d", w)) {
				t.Errorf("round %d: thread feat-w%d is missing after %d concurrent writes", round, w, writers)
			}
		}
	}
}

// One MEMORY.md is reached through the vault path and through the symlink under
// ~/.claude/projects/<key>/memory. Writers holding different paths to it must
// meet at one lock, or the lock guards nothing between them.
func TestTwoPathsToOneMemoryShareTheLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a symlinked directory needs privileges on Windows; junctions are not measured yet")
	}
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	vault := t.TempDir()
	memory := filepath.Join(vault, "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "memory")
	if err := os.Symlink(vault, link); err != nil {
		t.Fatal(err)
	}

	unlock, err := lockMemoryFile(memory)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	start := time.Now()
	if _, err := lockMemoryFileWithin(filepath.Join(link, "MEMORY.md"), 100*time.Millisecond); err == nil {
		t.Fatal("the symlinked path took the lock while the vault path held it")
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Error("the second writer gave up before its wait expired")
	}
}
