package agent

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// scriptedBackend answers a fixed response per `pool:model` entry and records
// the order it was asked in. `seen` is the load-bearing half: the claim AC2
// makes is about which entries were ATTEMPTED, and a test that only checked the
// final record would pass on a dispatcher that walked the whole chain and
// reported the last success.
type scriptedBackend struct {
	byEntry map[string]Response
	seen    []string
}

func (b *scriptedBackend) Dispatch(_ context.Context, req Request) Response {
	key := req.Pool + ":" + req.Model
	b.seen = append(b.seen, key)
	if r, ok := b.byEntry[key]; ok {
		return r
	}
	return Response{Status: StatusPoolUnavailable}
}

// fixedClock advances a fixed step per reading, so duration_ms is an exact
// expectation rather than the vacuous `>= 0` that cannot fail.
func fixedClock(step time.Duration) func() time.Time {
	base := time.Unix(0, 0)
	var n int64
	return func() time.Time {
		t := base.Add(time.Duration(n) * step)
		n++
		return t
	}
}

func TestDispatch_ChainWalk(t *testing.T) {
	tests := []struct {
		name       string
		tier       string
		chain      []string
		script     map[string]Response
		wantStatus Status
		wantPool   string
		wantModel  string
		wantExit   int
		// wantRecExit is the record's own exit field. Kept separate from
		// wantExit so a path that forgets to set it cannot borrow the
		// derived value and look correct.
		wantRecExit int
		wantSeen    []string
	}{
		{
			name:  "pool unavailable advances to the next entry",
			tier:  "mid",
			chain: []string{"claude:sonnet", "nan:deepseek-v4-flash"},
			script: map[string]Response{
				"claude:sonnet":         {Status: StatusPoolUnavailable},
				"nan:deepseek-v4-flash": {Status: StatusOK, Output: "answered"},
			},
			wantStatus:  StatusOK,
			wantPool:    "nan",
			wantModel:   "deepseek-v4-flash",
			wantExit:    0,
			wantRecExit: 0,
			wantSeen:    []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		},
		{
			name:  "task failed does not advance",
			tier:  "mid",
			chain: []string{"claude:sonnet", "nan:deepseek-v4-flash"},
			script: map[string]Response{
				"claude:sonnet":         {Status: StatusTaskFailed, Exit: 2, Output: "boom"},
				"nan:deepseek-v4-flash": {Status: StatusOK, Output: "must never be reached"},
			},
			wantStatus:  StatusTaskFailed,
			wantPool:    "claude",
			wantModel:   "sonnet",
			wantExit:    1,
			wantRecExit: 2,
			wantSeen:    []string{"claude:sonnet"},
		},
		{
			name:  "every entry unavailable is chain exhausted, not a task failure",
			tier:  "mid",
			chain: []string{"claude:sonnet", "nan:deepseek-v4-flash", "nan:mimo-v2.5"},
			script: map[string]Response{
				"claude:sonnet":         {Status: StatusPoolUnavailable},
				"nan:deepseek-v4-flash": {Status: StatusPoolUnavailable},
				"nan:mimo-v2.5":         {Status: StatusPoolUnavailable},
			},
			wantStatus:  StatusChainExhausted,
			wantExit:    3,
			wantRecExit: 3,
			wantSeen:    []string{"claude:sonnet", "nan:deepseek-v4-flash", "nan:mimo-v2.5"},
		},
		{
			name:  "an unrecognised backend status is a task failure, never an advance",
			tier:  "mid",
			chain: []string{"claude:sonnet", "nan:deepseek-v4-flash"},
			script: map[string]Response{
				"claude:sonnet": {Status: Status("who knows"), Exit: 9},
			},
			wantStatus:  StatusTaskFailed,
			wantPool:    "claude",
			wantModel:   "sonnet",
			wantExit:    1,
			wantRecExit: 9,
			wantSeen:    []string{"claude:sonnet"},
		},
		{
			name:  "a malformed chain entry fails closed rather than being skipped",
			tier:  "mid",
			chain: []string{"claude-sonnet", "nan:deepseek-v4-flash"},
			script: map[string]Response{
				"nan:deepseek-v4-flash": {Status: StatusOK, Output: "must never be reached"},
			},
			wantStatus:  StatusTaskFailed,
			wantExit:    1,
			wantRecExit: 1,
			wantSeen:    nil,
		},
		{
			name:        "dry run reports the resolved route and exits zero",
			tier:        "mid",
			chain:       []string{"claude:sonnet"},
			script:      map[string]Response{"claude:sonnet": {Status: StatusDryRun}},
			wantStatus:  StatusDryRun,
			wantPool:    "claude",
			wantModel:   "sonnet",
			wantExit:    0,
			wantRecExit: 0,
			wantSeen:    []string{"claude:sonnet"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			be := &scriptedBackend{byEntry: tc.script}
			rec := Dispatch(context.Background(), Options{
				Tier:    tc.tier,
				Role:    "reviewer",
				Task:    "check this",
				Chain:   tc.chain,
				Timeout: time.Minute,
				Now:     fixedClock(5 * time.Millisecond),
			}, be)

			if rec.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", rec.Status, tc.wantStatus)
			}
			if rec.Pool != tc.wantPool {
				t.Errorf("pool = %q, want %q", rec.Pool, tc.wantPool)
			}
			if rec.Model != tc.wantModel {
				t.Errorf("model = %q, want %q", rec.Model, tc.wantModel)
			}
			if got := ExitCode(rec.Status); got != tc.wantExit {
				t.Errorf("exit = %d, want %d", got, tc.wantExit)
			}
			// The record's own `exit` field, not just the derived one. They are
			// two different things and a caller reads the field: a record saying
			// `"status":"task_failed","exit":0` contradicts itself, and the
			// process exiting 1 beside it does not repair the record.
			if rec.Exit != tc.wantRecExit {
				t.Errorf("rec.Exit = %d, want %d (status %q)", rec.Exit, tc.wantRecExit, rec.Status)
			}
			if strings.Join(be.seen, ",") != strings.Join(tc.wantSeen, ",") {
				t.Errorf("attempted %v, want %v", be.seen, tc.wantSeen)
			}
		})
	}
}

// AC5. The rule is behavioural and unconditional, so it is asserted against a
// two-entry top chain that the map does not currently declare: a test that only
// used today's single-entry `chains.top` would pass on a dispatcher with no rule
// at all, and would go quiet the day someone added a fallback to the map.
func TestDispatch_TopTierNeverDegrades(t *testing.T) {
	be := &scriptedBackend{byEntry: map[string]Response{
		"claude:opus":           {Status: StatusPoolUnavailable},
		"nan:deepseek-v4-flash": {Status: StatusOK, Output: "a mid-tier answer"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:    TierTop,
		Role:    "architect",
		Task:    "decide",
		Chain:   []string{"claude:opus", "nan:deepseek-v4-flash"},
		Timeout: time.Minute,
		Now:     fixedClock(5 * time.Millisecond),
	}, be)

	if rec.Status != StatusEscalated {
		t.Errorf("status = %q, want %q", rec.Status, StatusEscalated)
	}
	if got := ExitCode(rec.Status); got == 0 {
		t.Errorf("exit = 0; the top tier being unservable must not read as success")
	}
	if len(be.seen) != 1 || be.seen[0] != "claude:opus" {
		t.Errorf("attempted %v; the top tier must never reach a second entry", be.seen)
	}
	if strings.Contains(rec.Output, "mid-tier answer") {
		t.Errorf("output carries a mid-tier answer: %q", rec.Output)
	}
}

func TestDispatch_RecordsDurationAndAttempts(t *testing.T) {
	be := &scriptedBackend{byEntry: map[string]Response{
		"claude:sonnet":         {Status: StatusPoolUnavailable},
		"nan:deepseek-v4-flash": {Status: StatusOK, Output: "answered"},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier:    "mid",
		Chain:   []string{"claude:sonnet", "nan:deepseek-v4-flash"},
		Timeout: time.Minute,
		Now:     fixedClock(5 * time.Millisecond),
	}, be)

	if rec.DurationMS != 5 {
		t.Errorf("duration_ms = %d, want 5 (clock steps 5ms per reading)", rec.DurationMS)
	}
	if len(rec.Attempts) != 2 {
		t.Fatalf("attempts = %d, want 2 — a skipped entry that leaves no trace is undiagnosable", len(rec.Attempts))
	}
	if rec.Attempts[0].Pool != "claude" || rec.Attempts[0].Status != StatusPoolUnavailable {
		t.Errorf("attempts[0] = %+v, want the claude entry reported unavailable", rec.Attempts[0])
	}
	if rec.Attempts[1].Model != "deepseek-v4-flash" || rec.Attempts[1].Status != StatusOK {
		t.Errorf("attempts[1] = %+v, want the nan entry reported ok", rec.Attempts[1])
	}
}

func TestDispatch_OutputIsCappedAndSaysSo(t *testing.T) {
	be := &scriptedBackend{byEntry: map[string]Response{
		"claude:sonnet": {Status: StatusOK, Output: strings.Repeat("x", OutputCap+100)},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier: "mid", Chain: []string{"claude:sonnet"}, Timeout: time.Minute,
		Now: fixedClock(time.Millisecond),
	}, be)

	if len(rec.Output) != OutputCap {
		t.Errorf("output length = %d, want %d", len(rec.Output), OutputCap)
	}
	if !rec.Truncated {
		t.Error("truncated = false; a capped output that does not say so reads as a complete short answer")
	}
}

// The cap counts bytes and a model's output is not ASCII, so the cut can land
// inside a rune.
//
// The fixture's rune WIDTH is the whole test. An ASCII one cannot see the bug at
// all — every byte is a boundary. Neither can a two-byte one: OutputCap is
// 65536, which is even, so a run of "é" always divides exactly and the cut lands
// on a boundary by arithmetic rather than by the code being correct. That
// version of this test passed against the byte-slicing cap it was written to
// catch. A three-byte rune is the smallest that misses: 65536 mod 3 == 1, so the
// cut lands one byte into a rune and something must step it back.
func TestDispatch_OutputIsCutOnARuneBoundary(t *testing.T) {
	multibyte := strings.Repeat("€", OutputCap) // 3 bytes each
	be := &scriptedBackend{byEntry: map[string]Response{
		"claude:sonnet": {Status: StatusOK, Output: multibyte},
	}}

	rec := Dispatch(context.Background(), Options{
		Tier: "mid", Chain: []string{"claude:sonnet"}, Timeout: time.Minute,
		Now: fixedClock(time.Millisecond),
	}, be)

	if !rec.Truncated {
		t.Fatal("truncated = false on an output twice the cap")
	}
	if len(rec.Output) > OutputCap {
		t.Errorf("output length = %d, exceeds the cap %d", len(rec.Output), OutputCap)
	}
	if !utf8.ValidString(rec.Output) {
		t.Error("output is not valid UTF-8: the cut split a rune, so the record reports corruption " +
			"where it meant to report a limit")
	}
	if strings.ContainsRune(rec.Output, utf8.RuneError) {
		t.Error("output carries U+FFFD; an orphan byte was kept rather than the cut being moved back")
	}
}

// The timeout is required per dispatch (ADR-032 §2), and "required" has to mean
// the backend actually receives a deadline — a parsed-but-inert flag is the
// silent lie. PR C adds the kill; this asserts the deadline exists at all.
func TestDispatch_BackendReceivesADeadline(t *testing.T) {
	var hadDeadline bool
	be := backendFunc(func(ctx context.Context, _ Request) Response {
		_, hadDeadline = ctx.Deadline()
		return Response{Status: StatusOK}
	})

	Dispatch(context.Background(), Options{
		Tier: "mid", Chain: []string{"claude:sonnet"}, Timeout: 30 * time.Second,
		Now: fixedClock(time.Millisecond),
	}, be)

	if !hadDeadline {
		t.Error("backend context carried no deadline; the per-dispatch timeout is inert")
	}
}

type backendFunc func(context.Context, Request) Response

func (f backendFunc) Dispatch(ctx context.Context, req Request) Response { return f(ctx, req) }
