package harness

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Outcome is WHY the gate decided what it decided.
//
// It is a closed vocabulary rather than a free-text reason because the record is
// a measurement surface: `features.json`, the agy/pi/opencode verification, and
// any future query all select on this field. A reason string is for a human; an
// outcome is for a filter, and a filter that matches on prose silently returns
// nothing the day the prose is reworded — the failure mode this repository has
// already catalogued in `check-roster-consistency.py`.
//
// Every value here is reachable from `dotf harness gate`. Deliberately absent is
// a "skill invocation" outcome: `Decide` has that branch, but the command
// returns on the skill paths before reaching it, so recording one would describe
// a state the gate cannot be in.
type Outcome string

const (
	// OutcomePayloadUnrecognised is stdin the gate could not parse at all.
	//
	// This is the single most diagnostic value in the vocabulary. A harness whose
	// field names do not match — agy is the open question — produces exactly this
	// and nothing else, and before this record existed it was indistinguishable
	// from a harness working perfectly: both allowed, and both wrote nothing.
	OutcomePayloadUnrecognised Outcome = "payload-unrecognised"
	// OutcomeSkillConsumed is a skill invocation, recorded in the ledger.
	OutcomeSkillConsumed Outcome = "skill-consumed"
	// OutcomeSkillUnnamed is the skill primitive with an unreadable argument.
	OutcomeSkillUnnamed Outcome = "skill-unnamed"
	// OutcomeNoRole is a call naming no persona — a main-thread call. Not an
	// error, and the pre-existing allow.
	OutcomeNoRole Outcome = "no-role"
	// OutcomeRoleUnresolved is a role that was ASKED FOR and did not resolve.
	// Enforcement is off for this call, which is the state AC5 names explicitly
	// because it is the one that looks identical to health from outside.
	OutcomeRoleUnresolved Outcome = "role-unresolved"
	// OutcomeAllow is a persona in scope with nothing outstanding.
	OutcomeAllow Outcome = "allow"
	// OutcomeWarn is allowed, with `enforce: warn` skills unconsumed.
	//
	// This value is the whole of AC6. A warn is emitted on stderr, and a
	// PreToolUse hook's stderr on exit 0 is not persisted anywhere the session
	// can be asked about afterwards — so before this record, a warn that fired
	// and a warn that never fired were the same observation.
	OutcomeWarn Outcome = "warn"
	// OutcomeBlock is a refused tool call.
	OutcomeBlock Outcome = "block"
)

// UnparsedScope is the ledger scope for a payload with no readable session.
//
// It has to go somewhere, and by definition it has no scope of its own. The
// leading underscore keeps it out of the space of real session ids — every
// harness observed here sends a UUID — so a genuine session can never land in
// this file and make an unparsed pile look like healthy traffic.
const UnparsedScope = "_unparsed"

// maxDecisionBytes caps one scope's journal before it rotates to a single `.1`
// generation, bounding a scope at twice this. The gate runs on EVERY tool call,
// so this file grows without an upper bound otherwise — and an unbounded log
// written by a hook is a way to fill a disk, which would then fail the very
// writes the fail-open discipline depends on being harmless.
const maxDecisionBytes = 1 << 20

// DecisionRecord is one gate decision, durably.
//
// IT CARRIES NO TOOL INPUT, and that is a security property rather than an
// omission. This file is durable, unencrypted, and nothing scans it. Tool inputs
// carry file contents, shell commands and credentials; a journal that logged
// them would be a secrets leak with a retention policy. The neutral `ToolCall`
// already drops them at the parse boundary, and this type is built to keep it
// that way: tool and skill are NAMES.
type DecisionRecord struct {
	Time    string `json:"ts"`
	Harness string `json:"harness"`
	Session string `json:"session,omitempty"`
	// AgentType is what the payload SAID was acting. Recorded separately from
	// RoleRequested and RoleResolved because the three can differ, and the
	// difference is the diagnosis: a `--role` override, a typo, or a record that
	// moved will show up as a resolved role that does not match the payload.
	AgentType string `json:"agent_type,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Tool      string `json:"tool,omitempty"`
	Skill     string `json:"skill,omitempty"`
	// RoleRequested is what the gate actually looked up, after the flag override.
	RoleRequested string `json:"role_requested,omitempty"`
	// RoleResolved is the persona it found. Empty with a `role-unresolved`
	// outcome is the case AC5 names; empty with `no-role` is a main-thread call.
	RoleResolved string   `json:"role_resolved,omitempty"`
	Outcome      Outcome  `json:"outcome"`
	Allowed      bool     `json:"allowed"`
	Reason       string   `json:"reason,omitempty"`
	Warned       []string `json:"warned,omitempty"`
	Missing      []string `json:"missing,omitempty"`
	// PayloadBytes is set only for an unrecognised payload. It separates "stdin
	// was empty" from "a real payload arrived and did not parse" — the two
	// causes of a silent gate, which need opposite fixes. The LENGTH, never the
	// content: the thing we could not parse is exactly the thing least safe to
	// write down.
	PayloadBytes int `json:"payload_bytes,omitempty"`
}

// DecisionPath is where one scope's journal lives.
//
// It reuses scopeKey, and therefore StatePath's collision digest, on purpose.
// Character-mapping alone collides (`a/b` and `a.b` both flatten to `a_b`), and
// on the consumption ledger that would open one session's gate with another's
// record. Here it would attribute one dispatch's decisions to another, which is
// worse in the specific way that matters: this file is the instrument, so a
// collision corrupts the measurement rather than announcing itself.
func DecisionPath(stateDir, scope string) string {
	return filepath.Join(stateDir, "gate", scopeKey(scope)+".decisions.jsonl")
}

// RecordDecision appends one decision.
//
// Errors are returned but every caller ignores them, matching RecordConsumed:
// losing a record costs a measurement, while failing here would block a session
// over a full disk. The gate's entire contract is that it never turns its own
// malfunction into a refused tool call.
func RecordDecision(path string, rec DecisionRecord) error {
	if rec.Time == "" {
		rec.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- path is built by DecisionPath from the state dir
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// ONE write of one line, on a fd opened O_APPEND. That is what keeps
	// concurrent writers from interleaving: the kernel makes the seek-to-end and
	// the write a single step for a regular file, and every record here is far
	// below the size where that stops holding. Building the line in memory first
	// is therefore load-bearing, not tidiness — two Writes could interleave.
	//
	// Concurrency is not hypothetical: the gate runs per tool call, and several
	// sessions and subagents on one machine write at once. Pinned by
	// TestConcurrentWritersLoseNoRecords.
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return err
	}

	// ROTATION HAPPENS AFTER THE WRITE, AND FROM OUR OWN FD. Doing it first —
	// stat the path, rename, then open — loses the record being written when
	// another writer rotates in between, and lets a file that is not full replace
	// the kept generation. Raised by the reviewer on #1435.
	//
	// fstat on the descriptor we just wrote answers about THAT file rather than
	// about whatever the path names now, and SameFile confirms the path still
	// refers to it before renaming.
	//
	// A RESIDUAL RACE REMAINS AND IS ACCEPTED. Between the SameFile check and the
	// rename, another writer can rotate, so two writers holding the same full file
	// can both proceed and the second may replace the generation the first just
	// kept. Measured — an assertion on the emergent property flaked.
	//
	// What it can cost is bounded to OLD history: the record being written is
	// already durable above, so nothing recent is lost, and a not-full file is no
	// longer a plausible replacement for a full one. Closing the window entirely
	// needs an advisory lock, and `syscall.Flock` does not exist on Windows —
	// this package compiles for it, and a Windows-only build break is a defect
	// this repository has already paid for once (#1075). A lock is not worth that
	// for a diagnostic journal whose worst case is losing records nobody has
	// asked about since they rotated.
	fi, err := f.Stat()
	if err != nil || fi.Size() < maxDecisionBytes {
		return nil //nolint:nilerr // a failed stat means only that rotation is skipped; the record is already durable
	}
	if cur, err := os.Stat(path); err == nil && os.SameFile(fi, cur) {
		_ = os.Rename(path, path+".1")
	}
	return nil
}

// LoadDecisions reads a scope's journal oldest-first.
//
// A malformed line is SKIPPED rather than failing the read. The writer appends
// from a hook that can be killed mid-write, so a torn final line is an expected
// state, and refusing to read the file because of it would lose every intact
// record before it — turning a partial answer into no answer.
func LoadDecisions(path string) ([]DecisionRecord, error) {
	f, err := os.Open(path) // #nosec G304 -- path is built by DecisionPath from the state dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var out []DecisionRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec DecisionRecord
		if json.Unmarshal(line, &rec) != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}
