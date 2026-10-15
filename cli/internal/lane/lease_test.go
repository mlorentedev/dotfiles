package lane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

var leaseNow = time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)

func loadLease(t *testing.T, path string) *worktree.Metadata {
	t.Helper()
	meta, err := worktree.LoadMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	return meta
}

// A worktree lane next creates carries the lease dotf worktree add writes, with
// the issue and the agent it was started for, and the lease is there before the
// agent's command is typed (#2547 AC1).
func TestNext_TheNewWorktreeCarriesTheLeaseBeforeTheAgentStarts(t *testing.T) {
	b := newFake(t, "pi")
	o := opts(t)
	if out, err := Handoff(o, b, queue); err != nil || !out.Started {
		t.Fatalf("Handoff = %+v, %v", out, err)
	}
	meta := loadLease(t, b.created)
	if meta == nil {
		t.Fatal("no lease written")
	}
	want := worktree.Metadata{Creator: "pi", Issue: 7, CreatedAt: leaseNow, LeaseExpiresAt: leaseNow.Add(24 * time.Hour), ReapOK: true}
	if *meta != want {
		t.Errorf("lease = %+v, want %+v", *meta, want)
	}
	if !b.leased["w1:p1"] {
		t.Error("the agent was typed before the lease existed")
	}
	excl, err := os.ReadFile(filepath.Join(o.GitDir, "info", "exclude"))
	if err != nil || !strings.Contains(string(excl), worktree.MetadataFileName+"\n") {
		t.Errorf("exclude = %q, %v", excl, err)
	}
}

// A rerun keeps the lease it finds, so created_at does not move, and the
// exclude line is not appended twice (#2547 AC2).
func TestNext_ARerunKeepsTheLeaseItFinds(t *testing.T) {
	idle := t.TempDir()
	first := leaseNow.Add(-3 * time.Hour)
	if err := worktree.SaveMetadata(idle, worktree.Metadata{Creator: "claude", Issue: 7, CreatedAt: first, LeaseExpiresAt: first.Add(24 * time.Hour), ReapOK: true}); err != nil {
		t.Fatal(err)
	}
	o := opts(t)
	for range 2 {
		b := &fakeBackend{agent: "claude", existing: map[string]Worktree{"fix/first-thing": {Path: idle, Branch: "fix/first-thing"}}}
		if _, err := Handoff(o, b, queue); err != nil {
			t.Fatal(err)
		}
	}
	if meta := loadLease(t, idle); meta == nil || !meta.CreatedAt.Equal(first) {
		t.Errorf("lease = %+v, want created_at %v kept", meta, first)
	}
	excl, _ := os.ReadFile(filepath.Join(o.GitDir, "info", "exclude"))
	if n := strings.Count(string(excl), worktree.MetadataFileName); n != 1 {
		t.Errorf("exclude holds the lease file %d times: %q", n, excl)
	}
}

// A worktree another workspace already shows starts no second agent, but a
// missing lease is still written: a first run that failed between creating the
// worktree and leasing it is repaired by the rerun. A dry run writes nothing.
func TestNext_AShownWorktreeWithoutALeaseIsLeasedButNotRestarted(t *testing.T) {
	for _, dry := range []bool{false, true} {
		shown := t.TempDir()
		b := &fakeBackend{agent: "claude", existing: map[string]Worktree{"fix/first-thing": {Path: shown, Branch: "fix/first-thing", Open: true}}}
		o := opts(t)
		o.DryRun = dry
		if out, err := Handoff(o, b, queue); err != nil || out.Started {
			t.Fatalf("dry=%v: Handoff = %+v, %v", dry, out, err)
		}
		if got := loadLease(t, shown) != nil; got == dry {
			t.Errorf("dry=%v: lease written = %v", dry, got)
		}
	}
}

// A backend that answers without a path gets no agent started: a worktree the
// lane cannot lease is one sweep would never reap.
func TestNext_ABackendAnswerWithoutAPathStartsNothing(t *testing.T) {
	b := newFake(t, "claude")
	b.created = ""
	if _, err := Handoff(opts(t), b, queue); err == nil || !strings.Contains(err.Error(), "no worktree path") {
		t.Errorf("want a refusal, got %v", err)
	}
	if _, typed := b.typed["w1:p1"]; typed {
		t.Error("an agent was started in a worktree with no lease")
	}
}

// What the lane writes is what dotf worktree list reads: the lane worktree is
// listed with its lease and issue, not as one with no dotf metadata (#2547 AC3).
func TestNext_WorktreeListReadsTheLaneLease(t *testing.T) {
	b := newFake(t, "agy")
	if _, err := Handoff(opts(t), b, queue); err != nil {
		t.Fatal(err)
	}
	git := &worktree.MockGitRunner{PorcelainOutput: "worktree /repos/dotfiles\nHEAD aaa\nbranch refs/heads/main\n\nworktree " + b.created + "\nHEAD bbb\nbranch refs/heads/fix/first-thing\n"}
	list, err := worktree.ListWithRunner("/repos/dotfiles", git, leaseNow.Add(time.Hour))
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if got := list[1]; got.Metadata == nil || got.Metadata.Issue != 7 || got.Metadata.Creator != "agy" || got.State != worktree.StateActive {
		t.Errorf("listed %+v (metadata %+v)", got, got.Metadata)
	}
}
