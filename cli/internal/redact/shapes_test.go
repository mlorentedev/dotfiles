package redact

import (
	"strings"
	"testing"
)

func TestMaskShapes(t *testing.T) {
	// Built at run time so the source holds no literal a secret scanner stops on.
	ghToken := "gh" + "p_" + strings.Repeat("a", 36)
	secret := "pa55" + strings.Repeat("w", 12)
	cases := []struct{ name, in, want string }{
		{"github token", "token " + ghToken + " used", "token [REDACTED] used"},
		{"bearer header", "Authorization: Bearer " + secret, "Authorization: Bearer [REDACTED]"},
		{"url userinfo keeps the user", "https://bob:" + secret + "@host/x", "https://bob:[REDACTED]@host/x"},
		{"plain text untouched", "Listening on 127.0.0.1:8087", "Listening on 127.0.0.1:8087"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskShapes(tc.in); got != tc.want {
				t.Errorf("MaskShapes(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// An escape sequence inside a token hides it from every shape; stripping
// first is what lets the mask see it.
func TestStripANSIExposesSplitToken(t *testing.T) {
	split := "gh" + "p_" + strings.Repeat("a", 18) + "\x1b[0m" + strings.Repeat("a", 18)
	if got := MaskShapes(split); !strings.Contains(got, "aaaa") {
		t.Fatalf("precondition: the split token should escape MaskShapes, got %q", got)
	}
	if got := MaskShapes(StripANSI(split)); got != "[REDACTED]" {
		t.Errorf("MaskShapes(StripANSI(split)) = %q, want [REDACTED]", got)
	}
}

func TestPatternsIsACopy(t *testing.T) {
	p := Patterns()
	p[0] = nil
	if Patterns()[0] == nil {
		t.Fatal("Patterns returned the shared slice; a caller can overwrite a shape")
	}
}
