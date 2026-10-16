package spec

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// roundInfix is what separates a review-state file's stem from its round
// number in a kept round: review.md's rounds are review-round1.md,
// review-round2.md, and so on.
const roundInfix = "-round"

// RoundName is the name round n of the review-state file name is kept under.
func RoundName(name string, n int) string {
	ext := filepath.Ext(name)
	return fmt.Sprintf("%s%s%d%s", strings.TrimSuffix(name, ext), roundInfix, n, ext)
}

// parseRound reports which declared review-state file name is a kept round of,
// and its number. ok is false for any other name, so an author's own
// notes-round1.md is still scanned.
func parseRound(name string) (base string, n int, ok bool) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	i := strings.LastIndex(stem, roundInfix)
	if i < 0 {
		return "", 0, false
	}
	digits := stem[i+len(roundInfix):]
	// RoundName never writes a leading zero, so review-round01.md is not a
	// kept round: it is scanned like any file the author wrote.
	if digits == "" || digits[0] == '0' || strings.Trim(digits, "0123456789") != "" {
		return "", 0, false
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 1 {
		return "", 0, false
	}
	base = stem[:i] + ext
	if !isDeclaredReviewState(base) {
		return "", 0, false
	}
	return base, n, true
}

// RotateReviewRound keeps the slot's previous verdict before a new round
// overwrites it (#2264). review.md and the request it answered move to the
// next round number, so the dispositions an author wrote against a failed
// round still have the verdict they answer. It returns the kept review's
// name, or "" when the slot held no verdict, and an undo that puts both files
// back, for a launch that fails before the new round has started.
//
// The number is one past the highest round already kept, counting both files,
// so a round made by hand before this existed is never overwritten.
func RotateReviewRound(specDir string, slot ReviewSlot) (kept string, undo func() error, err error) {
	none := func() error { return nil }
	if _, err := os.Lstat(filepath.Join(specDir, slot.Review)); errors.Is(err, fs.ErrNotExist) {
		return "", none, nil
	} else if err != nil {
		return "", none, err
	}
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return "", none, err
	}
	next := 1
	for _, e := range entries {
		if base, n, ok := parseRound(e.Name()); ok && (base == slot.Review || base == slot.Request) && n >= next {
			next = n + 1
		}
	}
	var moved [][2]string // {from, to}, in the order they were moved
	undo = func() error {
		var errs []error
		for i := len(moved) - 1; i >= 0; i-- {
			errs = append(errs, os.Rename(moved[i][1], moved[i][0]))
		}
		return errors.Join(errs...)
	}
	for _, name := range []string{slot.Review, slot.Request} {
		from, to := filepath.Join(specDir, name), filepath.Join(specDir, RoundName(name, next))
		if _, err := os.Lstat(from); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return "", none, errors.Join(err, undo())
		}
		if err := os.Rename(from, to); err != nil {
			return "", none, errors.Join(err, undo())
		}
		moved = append(moved, [2]string{from, to})
	}
	return RoundName(slot.Review, next), undo, nil
}
