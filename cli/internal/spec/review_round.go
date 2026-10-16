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
	if digits == "" || strings.Trim(digits, "0123456789") != "" {
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
// name, or "" when the slot held no verdict.
//
// The number is one past the highest round already kept, counting both files,
// so a round made by hand before this existed is never overwritten.
func RotateReviewRound(specDir string, slot ReviewSlot) (string, error) {
	if _, err := os.Lstat(filepath.Join(specDir, slot.Review)); errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return "", err
	}
	next := 1
	for _, e := range entries {
		if base, n, ok := parseRound(e.Name()); ok && (base == slot.Review || base == slot.Request) && n >= next {
			next = n + 1
		}
	}
	kept := RoundName(slot.Review, next)
	if err := os.Rename(filepath.Join(specDir, slot.Review), filepath.Join(specDir, kept)); err != nil {
		return "", err
	}
	request := filepath.Join(specDir, slot.Request)
	if _, err := os.Lstat(request); errors.Is(err, fs.ErrNotExist) {
		return kept, nil
	} else if err != nil {
		return kept, err
	}
	return kept, os.Rename(request, filepath.Join(specDir, RoundName(slot.Request, next)))
}
