package doctor

import (
	"strconv"
	"strings"
)

// compareVersions compares two dot-separated numeric versions, returning -1 if
// a < b, 0 if equal, +1 if a > b. It reproduces the `sort -V` semantics
// doctor.sh used to test `actual >= min_version`: component-wise numeric
// comparison, with a missing component treated as 0 (so "1.7" == "1.7.0").
//
// Non-numeric leading text in a component is stripped before parsing, so a
// stray "v1" or "1-rc" degrades gracefully to its numeric prefix rather than
// erroring — the twins never claimed full semver, only a monotonic compare.
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, bi := component(as, i), component(bs, i)
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

// atLeast reports whether actual >= min under compareVersions.
func atLeast(actual, min string) bool {
	return compareVersions(actual, min) >= 0
}

// component returns the integer value of the i-th dotted field, or 0 when
// absent. It reads the leading run of digits, ignoring any suffix.
func component(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	digits := strings.TrimLeftFunc(parts[i], func(r rune) bool {
		return r < '0' || r > '9'
	})
	end := 0
	for end < len(digits) && digits[end] >= '0' && digits[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(digits[:end])
	return n
}
