package doctor

import (
	"bytes"
	"errors"
	"testing"
)

// A Cadence install set user-scope HOME=C:\SPB_Data on a Windows box, and git,
// ssh and half of dotf moved there while the rest stayed under USERPROFILE,
// with no signal (#2432). doctor names both values in one FAIL.
func TestCheckForeignHome(t *testing.T) {
	cases := []struct {
		name   string
		stored map[string]string // user-scope persistent variables
		getErr error
		want   Status
		needle string
	}{
		{"no user-scope HOME → PASS", nil, nil, StatusPass, "no persisted user-scope HOME"},
		{"HOME equal to USERPROFILE, spelled differently → PASS", map[string]string{"HOME": `c:/users/manu/`}, nil, StatusPass, "agrees with USERPROFILE"},
		{"HOME through %USERPROFILE% → PASS", map[string]string{"HOME": `%USERPROFILE%`}, nil, StatusPass, "agrees with USERPROFILE"},
		{"a foreign HOME → FAIL naming both", map[string]string{"HOME": `C:\SPB_Data`}, nil, StatusFail, `user-scope HOME is C:\SPB_Data but USERPROFILE is C:\Users\manu`},
		{"registry unreadable → WARN", nil, errors.New("access denied"), StatusWarn, "unreadable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(map[string]string{"USERPROFILE": `C:\Users\manu`}, nil, nil)
			sys.GOOS = "windows"
			sys.UserEnv = func(name string) (string, bool, error) {
				v, ok := tc.stored[name]
				return v, ok, tc.getErr
			}
			var buf bytes.Buffer
			checkForeignHome(sys, capture(&buf))
			if got := statusOfLine(buf.String(), tc.needle); got != tc.want {
				t.Fatalf("line mentioning %q: status %q, want %q\n%s", tc.needle, tagOf(got), tagOf(tc.want), buf.String())
			}
		})
	}

	t.Run("not Windows → no section", func(t *testing.T) {
		sys := newSys(nil, nil, nil)
		sys.GOOS = "linux"
		sys.UserEnv = func(string) (string, bool, error) { return "/elsewhere", true, nil }
		var buf bytes.Buffer
		checkForeignHome(sys, capture(&buf))
		if buf.Len() != 0 {
			t.Fatalf("want nothing off Windows, got\n%s", buf.String())
		}
	})
}
