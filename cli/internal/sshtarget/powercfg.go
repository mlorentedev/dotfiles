package sshtarget

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PowercfgSetting is one AC value of the active power scheme, by powercfg's
// subgroup and setting aliases.
type PowercfgSetting struct {
	Sub, Setting string
	Value        uint64
}

// PowercfgProfile is what a Windows SSH target keeps on AC power, the values
// the runbook applied by hand: never sleep or hibernate on idle, and do
// nothing when the lid closes. Battery (the DC index) is never touched: a
// laptop that ignores its lid on battery drains to a hard power-off.
var PowercfgProfile = []PowercfgSetting{
	{"SUB_SLEEP", "STANDBYIDLE", 0},
	{"SUB_SLEEP", "HIBERNATEIDLE", 0},
	{"SUB_BUTTONS", "LIDACTION", 0},
}

// Powercfg is the power policy on Windows. It reads `powercfg /q` without
// elevation and writes the AC index of the active scheme, then activates the
// scheme again so the change takes effect.
type Powercfg struct {
	Run Runner
}

func (p Powercfg) Differs() (bool, error) {
	pending, _, err := p.pending()
	return len(pending) > 0, err
}

// Apply sets each differing AC value; the undo sets back the value it read.
func (p Powercfg) Apply(root Root) (func() error, error) {
	noop := func() error { return nil }
	pending, prev, err := p.pending()
	if err != nil || len(pending) == 0 {
		return noop, err
	}
	var set []PowercfgSetting
	undo := func() error {
		for i := len(set) - 1; i >= 0; i-- {
			s := set[i]
			if err := setAC(root, s.Sub, s.Setting, prev[s.Setting]); err != nil {
				return err
			}
		}
		return activate(root)
	}
	for _, s := range pending {
		if err := setAC(root, s.Sub, s.Setting, s.Value); err != nil {
			return undo, err
		}
		set = append(set, s)
	}
	if err := activate(root); err != nil {
		return undo, err
	}
	return undo, nil
}

// Prove reads every value back and names the ones that still differ.
func (p Powercfg) Prove() error {
	pending, cur, err := p.pending()
	if err != nil || len(pending) == 0 {
		return err
	}
	var got []string
	for _, s := range pending {
		got = append(got, fmt.Sprintf("%s %#x (want %#x)", s.Setting, cur[s.Setting], s.Value))
	}
	return fmt.Errorf("powercfg still reports on AC power: %s", strings.Join(got, ", "))
}

func (p Powercfg) pending() ([]PowercfgSetting, map[string]uint64, error) {
	cur := map[string]uint64{}
	var pending []PowercfgSetting
	for _, s := range PowercfgProfile {
		out, err := p.Run("powercfg", "/q", "SCHEME_CURRENT", s.Sub, s.Setting)
		if err != nil {
			return nil, nil, fmt.Errorf("powercfg /q SCHEME_CURRENT %s %s: %w: %s", s.Sub, s.Setting, err, bytes.TrimSpace(out))
		}
		ac, err := parsePowercfgAC(out)
		if err != nil {
			return nil, nil, fmt.Errorf("powercfg /q SCHEME_CURRENT %s %s: %w", s.Sub, s.Setting, err)
		}
		cur[s.Setting] = ac
		if ac != s.Value {
			pending = append(pending, s)
		}
	}
	return pending, cur, nil
}

var powercfgHex = regexp.MustCompile(`:\s*0x([0-9a-fA-F]+)\s*$`)

// parsePowercfgAC reads the AC index of one setting from `powercfg /q`. The
// labels are in the system's language, so it reads none of them: the AC and
// DC indexes are always the last two hexadecimal values, AC first. A range
// setting lists its minimum, maximum and increment before them, in hex too.
func parsePowercfgAC(out []byte) (uint64, error) {
	var vals []uint64
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		m := powercfgHex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v, err := strconv.ParseUint(m[1], 16, 64)
		if err != nil {
			return 0, err
		}
		vals = append(vals, v)
	}
	if len(vals) < 2 {
		return 0, fmt.Errorf("no AC and DC index in the output: %s", bytes.TrimSpace(out))
	}
	return vals[len(vals)-2], nil
}

func setAC(root Root, sub, setting string, v uint64) error {
	return asRoot(root, "powercfg", "/setacvalueindex", "SCHEME_CURRENT", sub, setting, strconv.FormatUint(v, 10))
}

// activate makes Windows read the changed scheme; until then the old values
// stay in force.
func activate(root Root) error {
	return asRoot(root, "powercfg", "/setactive", "SCHEME_CURRENT")
}
