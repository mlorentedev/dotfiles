package sshtarget

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
)

// Retirement is what retiring the target would undo on a machine whose role
// was removed (SEC-014, #2388), read from the record without root. Only what
// an entry says the step created or replaced is undone, in an order that
// never leaves sshd listening on its stock config: the listener off and
// persisted first, then the drop-in, then the key line, the record last.
//
// The firewall and the power policy are not retired here: a firewall left
// behind keeps port 22 closed off the mesh (#2794), and the power policy may
// stay by decision (runbook, Rollback).
type Retirement struct {
	Present bool     // a record exists
	Pending []string // what a run changes, in order
	Notes   []string // what it leaves, and why
	dropIn  *Prior
	keys    []byte // authorized_keys with the key's lines taken out
}

// Converged reports whether a run would change nothing.
func (r Retirement) Converged() bool { return len(r.Pending) == 0 }

// InspectRetire plans the retirement. It runs no command as root.
func (t Target) InspectRetire() (Retirement, error) {
	var r Retirement
	if t.Paths.Record == "" {
		return r, nil
	}
	data, err := readOptional(t.Paths.Record)
	if err != nil || data == nil {
		return r, err
	}
	rec, err := readRecord(t.Paths.Record)
	if err != nil {
		return r, err
	}
	r.Present = true
	leave := func(res string, p *Prior) bool {
		if p != nil && p.State == PriorUnknown {
			r.Notes = append(r.Notes, res+": left, it was in place before the record and may be the owner's")
		}
		return p != nil && p.State != PriorUnknown
	}

	listening, err := serving(t.Service)
	if err != nil {
		return r, err
	}
	if leave(resService, rec.Listener) && listening {
		r.Pending = append(r.Pending, resService)
		listening = false
	}
	held := false
	if leave(resDropIn, rec.DropIn) {
		cur, err := readOptional(t.Paths.DropIn)
		if err != nil {
			return r, err
		}
		undone := cur == nil
		if rec.DropIn.State == PriorReplaced {
			undone = string(cur) == rec.DropIn.Content
		}
		switch {
		case undone:
		case listening:
			// Without it, an sshd the step did not start runs its stock
			// config, passwords included.
			held = true
			r.Notes = append(r.Notes, resDropIn+": held while sshd listens, which the step did not start: turn sshd off, then converge again")
		default:
			r.Pending, r.dropIn = append(r.Pending, resDropIn), rec.DropIn
		}
	}
	if leave(resKey, rec.Key) {
		keys, err := readOptional(t.Paths.AuthorizedKeys)
		if err != nil {
			return r, err
		}
		key := bytes.TrimSpace(t.Key)
		if rec.Key.Content != "" {
			key = []byte(rec.Key.Content)
		}
		switch {
		case hasKey(keys, key):
			r.Pending, r.keys = append(r.Pending, resKey), withoutKey(keys, key)
		case rec.Key.Content == "":
			// A record of a build that did not keep the line, and the repo's
			// key in no line: rotated since, or removed by the owner.
			r.Notes = append(r.Notes, resKey+": no line holds the repo's key, and the record does not say which key the step wrote: check "+t.Paths.AuthorizedKeys)
		}
	}
	// A held entry needs the record to be undone later; an unknown one never
	// will be, so the record goes once nothing is held.
	if !held {
		r.Pending = append(r.Pending, resRecord)
	}
	return r, nil
}

// Retire runs the plan. A step that fails stops the run and keeps the record,
// so the next converge resumes from what is left; nothing already undone is
// put back, since putting it back would reopen what the run closed.
func (t Target) Retire(r Retirement) error {
	if r.Converged() {
		return nil
	}
	if slices.Contains(r.Pending, resService) && (t.Session != "" || t.Sudoed) {
		return errors.New("retiring stops sshd, which would cut an ssh session this run may be in ($SSH_CONNECTION is set, or sudo stripped it): run the converge at the machine, as your user")
	}
	if err := t.Root.Ready(); err != nil {
		return err
	}
	for _, res := range r.Pending {
		var err error
		switch res {
		case resService:
			err = t.Service.Disable(t.Root)
		case resDropIn:
			var prev []byte
			if r.dropIn.State == PriorReplaced {
				prev = []byte(r.dropIn.Content)
			}
			err = t.restoreRoot(t.Paths.DropIn, prev)
		case resKey:
			err = t.writeKeys(r.keys)
		case resRecord:
			err = t.restoreRoot(t.Paths.Record, nil)
		}
		if err != nil {
			return fmt.Errorf("retire the %s: %w (the record stays, so the next converge resumes)", res, err)
		}
	}
	return nil
}

// serving asks whether sshd runs now or starts at boot, where the service can
// tell the two apart from Enabled's "both"; launchd's loaded job is both.
func serving(s Service) (bool, error) {
	if v, ok := s.(interface{ Serving() (bool, error) }); ok {
		return v.Serving()
	}
	return s.Enabled()
}

// withoutKey is keys with every line that authorizes key taken out, the
// others kept byte for byte.
func withoutKey(keys, key []byte) []byte {
	var out []byte
	for _, line := range bytes.SplitAfter(keys, []byte("\n")) {
		if !hasKey(line, key) {
			out = append(out, line...)
		}
	}
	return out
}
