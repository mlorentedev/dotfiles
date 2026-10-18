package sshtarget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

// RecordVersion is the record's format. A build refuses a record of a later
// version rather than read it wrong or overwrite it.
const RecordVersion = 1

// Record is what the step found before it first changed each resource,
// written as root and readable by the user (SEC-014, #2388). Retiring the
// target when the role is removed undoes only what an entry names: a machine
// hardened by hand, which has no record, is never stripped, and a listener
// the step did not start is never stopped.
//
// An entry is written once, by the first apply that sees the resource, and no
// later apply rewrites it; only the retirement removes the file.
type Record struct {
	Version  int       `json:"version"`
	Dotf     string    `json:"dotf"`    // the build that first wrote it
	Written  time.Time `json:"written"` // when the last entry was added
	Key      *Prior    `json:"authorized_key,omitempty"`
	DropIn   *Prior    `json:"sshd_drop_in,omitempty"`
	Listener *Prior    `json:"sshd_listener,omitempty"`
}

// Prior is one resource as the step first found it.
type Prior struct {
	State string `json:"state"`
	// Content is the drop-in that was there, when State is PriorReplaced, and
	// the key line the step wrote, when the key is PriorCreated: the repo's
	// key may have been rotated by the time the role is removed.
	Content string `json:"content,omitempty"`
}

// The states of a Prior, from the step's side.
const (
	// PriorCreated: absent (the listener: not enabled) before the step, or
	// a drop-in carrying DropInMarker, so the step's own. Retiring removes or
	// disables it.
	PriorCreated = "created"
	// PriorReplaced: the drop-in was there with other content, kept in
	// Content. Retiring puts it back.
	PriorReplaced = "replaced"
	// PriorUnknown: already in its declared state when the step first
	// recorded it, on a machine converged before the record existed, so
	// nothing can tell whether the step or the owner put it there. Retiring
	// leaves it and says so.
	PriorUnknown = "unknown"
)

// readRecord reads the record at path; a missing one is the zero Record.
func readRecord(path string) (Record, error) {
	var r Record
	data, err := readOptional(path)
	if err != nil || data == nil {
		return r, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("%s: unreadable record: %w", path, err)
	}
	if r.Version < 1 || r.Version > RecordVersion {
		return r, fmt.Errorf("%s: record version %d, this build reads %d", path, r.Version, RecordVersion)
	}
	return r, nil
}

// fill adds an entry for each resource the record lacks, from what Inspect
// found before any change, and reports whether it added one.
func (r *Record) fill(p Plan, pending func(string) bool) bool {
	added := false
	set := func(e **Prior, prior Prior) {
		if *e == nil {
			*e, added = &prior, true
		}
	}
	set(&r.Key, onlyIf(pending(resKey), Prior{State: PriorCreated, Content: string(p.key)}))
	switch {
	case ownDropIn(p.prevDrop):
		// A render of the step's own file, from this build or an earlier
		// one: restoring it on retirement would put dotf's file back.
		set(&r.DropIn, Prior{State: PriorCreated})
	case !pending(resDropIn):
		set(&r.DropIn, Prior{State: PriorUnknown})
	case p.prevDrop == nil:
		set(&r.DropIn, Prior{State: PriorCreated})
	default:
		set(&r.DropIn, Prior{State: PriorReplaced, Content: string(p.prevDrop)})
	}
	set(&r.Listener, onlyIf(pending(resService), Prior{State: PriorCreated}))
	return added
}

// DropInMarker opens every drop-in the step has rendered
// (ssh/sshd-workstation.conf).
const DropInMarker = "# Managed by `dotf converge --only workstation-target`"

func ownDropIn(drop []byte) bool { return bytes.HasPrefix(drop, []byte(DropInMarker)) }

// onlyIf is created when the step is about to create the resource, and
// unknown when it found it already in place.
func onlyIf(pending bool, created Prior) Prior {
	if pending {
		return created
	}
	return Prior{State: PriorUnknown}
}

// writeRecord installs data at path as root, creating its directory first:
// /var/db/dotf on a Mac exists only once something of dotf's has written there.
func writeRecord(root Root, path string, data []byte) error {
	if _, ok := root.(fileRoot); !ok {
		if err := asRoot(root, "install", "-d", "-m", "0755", filepath.Dir(path)); err != nil {
			return err
		}
	}
	return installRoot(root, path, data)
}
