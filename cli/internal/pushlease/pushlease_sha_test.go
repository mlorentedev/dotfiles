package pushlease

import (
	"strings"
	"testing"
)

// --expect takes a SHA-1 (40 hex) or a SHA-256 (64 hex) object name. Every
// push fixture is a SHA-1 repository, so the 64-hex half of the shape is
// pinned here (a gap the HARNESS-203 first signature named).
func TestFullSHA_AcceptsBothObjectFormatsAndNothingElse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"sha1", strings.Repeat("a", 40), true},
		{"sha256", strings.Repeat("b", 64), true},
		{"sha1 short by one", strings.Repeat("a", 39), false},
		{"between the formats", strings.Repeat("a", 41), false},
		{"sha256 short by one", strings.Repeat("b", 63), false},
		{"sha256 long by one", strings.Repeat("b", 65), false},
		{"not hex", strings.Repeat("g", 40), false},
		{"upper case before Prepare lowers it", strings.Repeat("A", 64), false},
	}
	for _, c := range cases {
		if got := fullSHA.MatchString(c.in); got != c.want {
			t.Errorf("%s: fullSHA.MatchString(%d chars) = %v, want %v", c.name, len(c.in), got, c.want)
		}
	}
}
