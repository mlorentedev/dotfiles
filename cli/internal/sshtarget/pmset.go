package sshtarget

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// PmsetProfile is the AC power profile of a macOS SSH target (owner decision
// on #2356, 2026-10-13): never sleep or spin the disk down, wake for network
// access, restart after a power loss, and keep Power Nap's maintenance wakes.
// displaysleep is left alone, because a lit display keeps nothing reachable.
// Battery is never touched: a laptop that ignores sleep on battery drains to a
// hard power-off.
var PmsetProfile = []PmsetSetting{
	{"sleep", "0"},
	{"disksleep", "0"},
	{"womp", "1"},
	{"autorestart", "1"},
	{"powernap", "1"},
}

// PmsetSetting is one `pmset -c <key> <value>` pair.
type PmsetSetting struct{ Key, Value string }

// Pmset is the power policy on macOS. It reads the AC section of `pmset -g
// custom`, which needs no root, and sets only what differs. A key the hardware
// does not report on AC (womp on a Mac without Ethernet, say) is not this
// machine's to set and never makes the plan pending.
type Pmset struct {
	Run Runner
}

func (p Pmset) Differs() (bool, error) {
	pending, _, err := p.pending()
	return len(pending) > 0, err
}

// Apply sets the differing keys in one `pmset -c` call. The undo sets them
// back to the values read before it, in one call too.
func (p Pmset) Apply(root Root) (func() error, error) {
	noop := func() error { return nil }
	pending, prev, err := p.pending()
	if err != nil || len(pending) == 0 {
		return noop, err
	}
	set, back := []string{"pmset", "-c"}, []string{"pmset", "-c"}
	for _, s := range pending {
		set = append(set, s.Key, s.Value)
		back = append(back, s.Key, prev[s.Key])
	}
	undo := func() error { return asRoot(root, back...) }
	if err := asRoot(root, set...); err != nil {
		// pmset writes its keys one by one, so a failure may leave some set.
		return undo, err
	}
	return undo, nil
}

// Prove reads the AC section back and fails naming every key that still
// differs from the profile.
func (p Pmset) Prove() error {
	pending, cur, err := p.pending()
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	var got []string
	for _, s := range pending {
		got = append(got, fmt.Sprintf("%s %s (want %s)", s.Key, cur[s.Key], s.Value))
	}
	return fmt.Errorf("pmset still reports on AC power: %s", strings.Join(got, ", "))
}

// pending is the profile's settings this machine reports with another value,
// and every AC value it reports.
func (p Pmset) pending() ([]PmsetSetting, map[string]string, error) {
	out, err := p.Run("pmset", "-g", "custom")
	if err != nil {
		return nil, nil, fmt.Errorf("pmset -g custom: %w: %s", err, bytes.TrimSpace(out))
	}
	cur, err := parsePmsetAC(out)
	if err != nil {
		return nil, nil, err
	}
	var pending []PmsetSetting
	for _, s := range PmsetProfile {
		if v, ok := cur[s.Key]; ok && v != s.Value {
			pending = append(pending, s)
		}
	}
	return pending, cur, nil
}

// parsePmsetAC reads the "AC Power:" section of `pmset -g custom`. A section
// header is a line with no leading space; a setting is an indented line, its
// name the first field and its value the second. Only one-word names are read,
// which every profile key is: "Sleep On Power Button 1" is skipped, and so is
// any annotation pmset prints after a value.
func parsePmsetAC(out []byte) (map[string]string, error) {
	vals := map[string]string{}
	inAC, seen := false, false
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' {
			inAC = strings.TrimSpace(line) == "AC Power:"
			seen = seen || inAC
			continue
		}
		f := strings.Fields(line)
		if !inAC || len(f) < 2 || !isPmsetKey(f[0]) {
			continue
		}
		vals[f[0]] = f[1]
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !seen {
		return nil, fmt.Errorf("pmset -g custom reports no AC Power section: %s", bytes.TrimSpace(out))
	}
	return vals, nil
}

func isPmsetKey(name string) bool {
	for _, s := range PmsetProfile {
		if s.Key == name {
			return true
		}
	}
	return false
}
