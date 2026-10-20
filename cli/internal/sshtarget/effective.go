package sshtarget

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// proven are the directives of the drop-in whose effect the apply checks. They
// are the ones that keep SSH key-only and open to the mesh: a drop-in sshd
// never read leaves every one of them at sshd's default.
var proven = []string{
	"authenticationmethods",
	"passwordauthentication",
	"kbdinteractiveauthentication",
	"permitrootlogin",
	"allowusers",
}

// checkEffective compares what `sshd -T` says sshd would run with against the
// drop-in. `sshd -t` proves the config parses; an Include that matches no file
// parses too, so only the dump shows the drop-in was read (#2763).
//
// The expected values come from the drop-in itself, so the template stays the
// single source: a directive added to `proven` needs no second list. Keywords
// are lowercased by sshd, and a value is compared by its fields, case
// ignored, because AllowUsers prints one line per pattern and the main config
// may add patterns of its own.
func checkEffective(dropIn, effective []byte, path string) error {
	want, got := dumpFields(dropIn), dumpFields(effective)
	for _, key := range proven {
		fields, ok := want[key]
		if !ok {
			continue
		}
		for _, f := range fields {
			if !contains(got[key], f) {
				return fmt.Errorf("sshd runs with %s %s, not %q: it did not read the drop-in %s (an Include that matches no file passes `sshd -t`)",
					key, shown(got[key]), f, path)
			}
		}
	}
	return nil
}

// dumpFields maps each lowercased keyword to every field it carries across all
// its lines.
func dumpFields(b []byte) map[string][]string {
	out := map[string][]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// sshd also reads `Keyword=value`; the dump never prints it.
		if i := strings.IndexAny(line, " \t="); i > 0 && line[i] == '=' {
			line = line[:i] + " " + line[i+1:]
		}
		f := strings.Fields(line)
		if len(f) < 2 || strings.HasPrefix(f[0], "#") {
			continue
		}
		k := strings.ToLower(f[0])
		out[k] = append(out[k], f[1:]...)
	}
	return out
}

func contains(have []string, want string) bool {
	for _, h := range have {
		if strings.EqualFold(h, want) {
			return true
		}
	}
	return false
}

func shown(have []string) string {
	if len(have) == 0 {
		return "unset"
	}
	return strings.Join(have, " ")
}
