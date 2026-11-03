package redact

import "regexp"

// Writer masks values the caller knows. The functions below mask what nobody
// declared: credentials recognisable by their form alone, in text a process
// printed on its own (a converge transcript, a daemon's log read back by
// doctor). One set of shapes serves every such reader, so a shape added for one
// protects them all (#1750).

var (
	// ansiEscape matches CSI and OSC sequences: colour, cursor and title codes.
	ansiEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)`)
	// tokenShapes each have two groups, the context before and after the
	// secret, which survive; the secret between them is masked.
	tokenShapes = []*regexp.Regexp{
		regexp.MustCompile(`()\bgh[pousr]_[A-Za-z0-9]{30,}()`),
		regexp.MustCompile(`()\bgithub_pat_[A-Za-z0-9_]{20,}()`),
		regexp.MustCompile(`()\bsk-[A-Za-z0-9_-]{20,}()`),
		regexp.MustCompile(`()\bAKIA[0-9A-Z]{16}\b()`),
		regexp.MustCompile(`()\bxox[abprs]-[A-Za-z0-9-]{10,}()`),
		regexp.MustCompile(`()\bAGE-SECRET-KEY-1[0-9A-Z]{20,}()`),
		regexp.MustCompile(`()\bnpm_[A-Za-z0-9]{36}\b()`),
		regexp.MustCompile(`()\bpypi-AgEI[A-Za-z0-9_-]{20,}()`),
		regexp.MustCompile(`(?i)(\b(?:proxy-)?authorization:\s*(?:bearer|basic|token)\s+)[^\s'"]+()`),
		regexp.MustCompile(`(?i)(\bbearer\s+)[A-Za-z0-9._~+/=-]{16,}()`),
		// URL userinfo: scheme://user:password@host keeps the user, masks the password.
		regexp.MustCompile(`(\b[A-Za-z][A-Za-z0-9+.-]*://[^/\s:@]+:)[^@\s/]+(@)`),
	}
)

// StripANSI removes terminal escape sequences, which would otherwise split a
// credential so no shape matches it.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// MaskShapes replaces every credential-shaped run in s with [REDACTED],
// keeping the text around it. Strip escapes first.
func MaskShapes(s string) string {
	for _, re := range tokenShapes {
		s = re.ReplaceAllString(s, "${1}[REDACTED]${2}")
	}
	return s
}

// Patterns returns the escape pattern and every shape, for a caller that must
// not cut a stream inside a match. The slice is a fresh copy.
func Patterns() []*regexp.Regexp {
	return append([]*regexp.Regexp{ansiEscape}, tokenShapes...)
}
