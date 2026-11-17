package harness

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestDecide_ExemptsReadOnlyDiscoveryTools(t *testing.T) {
	p := gatePersona(SkillBinding{ID: "test-driven-development", Enforce: EnforceBlock})

	for _, tool := range []string{"Read", "Grep", "Glob", "LS", "read_file", "grep_search", "find_by_name", "list_dir"} {
		res := Decide(GateInput{
			Persona:  p,
			Call:     ToolCall{Tool: tool},
			Consumed: map[string]bool{},
		})
		if res.Decision != Allow {
			t.Errorf("read-only tool %q was blocked (%s); discovery tools must be exempt to prevent session deadlock", tool, res.Reason)
		}
	}

	// Mutating tool must still be blocked when blocking skill is unconsumed.
	mut := Decide(GateInput{
		Persona:  p,
		Call:     ToolCall{Tool: "Edit"},
		Consumed: map[string]bool{},
	})
	if mut.Decision != Block {
		t.Errorf("mutating tool Edit got %v, want Block", mut.Decision)
	}
}

func TestStickyPersona_BindsOncePerBranchAndResetsOnBranchSwitch(t *testing.T) {
	dir := t.TempDir()
	path := PersonaStatePath(dir, "sess-sticky-1")
	if path == "" {
		t.Fatal("PersonaStatePath returned empty for valid scope")
	}

	if got := LoadStickyPersona(path, "feat/101-alpha"); got != "" {
		t.Fatalf("initial LoadStickyPersona = %q, want empty", got)
	}

	if err := BindStickyPersona(path, "builder", "feat/101-alpha"); err != nil {
		t.Fatalf("BindStickyPersona: %v", err)
	}
	if got := LoadStickyPersona(path, "feat/101-alpha"); got != "builder" {
		t.Fatalf("LoadStickyPersona = %q, want builder", got)
	}

	// Attempting to bind a second persona ("architect") on the same branch when
	// editing docs/adr/ for ## Knowledge must NOT thrash the sticky persona.
	if err := BindStickyPersona(path, "architect", "feat/101-alpha"); err != nil {
		t.Fatalf("second BindStickyPersona: %v", err)
	}
	if got := LoadStickyPersona(path, "feat/101-alpha"); got != "builder" {
		t.Fatalf("sticky persona thrashed to %q on same branch, want builder", got)
	}

	// Switching the lane to a new branch resets the sticky binding.
	if got := LoadStickyPersona(path, "fix/102-bug"); got != "" {
		t.Fatalf("LoadStickyPersona on new branch = %q, want empty", got)
	}
	if err := BindStickyPersona(path, "reviewer", "fix/102-bug"); err != nil {
		t.Fatalf("BindStickyPersona on new branch: %v", err)
	}
	if got := LoadStickyPersona(path, "fix/102-bug"); got != "reviewer" {
		t.Fatalf("LoadStickyPersona on new branch = %q, want reviewer", got)
	}
}

func TestAtomicWriteFile_ConcurrentWritesOnWindowsDoNotFail(t *testing.T) {
	dir := t.TempDir()
	path := StatePath(dir, "sess-concurrent")

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := RecordConsumed(path, "skill"); err != nil {
				errs <- err
			}
			dpath := filepath.Join(dir, "gate", "concurrent.dispatch.json")
			if err := RecordDispatch(dpath, "worker", "reviewer"); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent state write failed: %v", err)
	}
	consumed := LoadConsumed(path)
	if !consumed["skill"] {
		t.Fatalf("LoadConsumed(%s) = %v, want skill=true", path, consumed)
	}
}
