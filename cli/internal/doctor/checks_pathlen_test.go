package doctor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// windowsPath builds a ';'-joined PATH of n distinct entries, each repeated
// copies times, so a test controls both the length and the duplicates.
func windowsPath(n, copies int) string {
	var entries []string
	for c := 0; c < copies; c++ {
		for i := 0; i < n; i++ {
			entries = append(entries, fmt.Sprintf(`C:\hostedtoolcache\windows\tool%03d\bin`, i))
		}
	}
	return strings.Join(entries, ";")
}

// cmd.exe resolves nothing from a PATH past 8191 chars, wherever the entry
// sits, so every mise .cmd shim fails with "'mise' is not recognized" while
// pwsh still finds mise.exe (#3285, lesson 574). Doctor names the length,
// once, instead of leaving a confusing failure per tool.
func TestCheckCmdPathLength(t *testing.T) {
	cases := []struct {
		name, goos, path string
		want, absent     string
		fails            bool
	}{
		{"under the limit", "windows", windowsPath(100, 1), "[ OK ] PATH is 3899 chars, under cmd.exe's 8191", "", false},
		{"over, with duplicates", "windows", windowsPath(100, 3),
			"PATH is 11699 chars, past cmd.exe's 8191: cmd.exe resolves nothing from it, so every mise shim reports 'mise' is not recognized. 200 of its 300 entries repeat an earlier one", "", true},
		{"over, no duplicates", "windows", windowsPath(250, 1), "PATH is 9749 chars, past cmd.exe's 8191", "repeat an earlier one", true},
		{"same spelling differs only in case and a trailing slash", "windows",
			windowsPath(220, 1) + `;C:\HOSTEDTOOLCACHE\windows\tool000\bin\`, "1 of its 221 entries repeat an earlier one", "", true},
		{"not Windows", "linux", strings.Repeat("/x", 6000), "", "PATH is", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sys := newSys(map[string]string{"PATH": tc.path}, nil, nil)
			sys.GOOS = tc.goos
			var buf bytes.Buffer
			rep := capture(&buf)
			checkCmdPathLength(sys, rep)
			out := buf.String()
			if tc.want != "" && !strings.Contains(out, tc.want) {
				t.Errorf("want %q in\n%s", tc.want, out)
			}
			if tc.absent != "" && strings.Contains(out, tc.absent) {
				t.Errorf("did not want %q in\n%s", tc.absent, out)
			}
			if got := rep.Failures() > 0; got != tc.fails {
				t.Errorf("failed = %v, want %v\n%s", got, tc.fails, out)
			}
		})
	}
}
