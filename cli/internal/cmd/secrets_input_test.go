package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// terminalLines makes stdin a terminal whose hidden reads return lines, one per
// call, then the given terminator: io.EOF (Ctrl-D) or a "\x1a" line (Ctrl-Z then
// Enter on Windows).
func terminalLines(t *testing.T, lines []string, windowsEOF bool) {
	t.Helper()
	oldTerm, oldRead := stdinIsTerminal, readPassword
	t.Cleanup(func() { stdinIsTerminal, readPassword = oldTerm, oldRead })
	stdinIsTerminal = func() bool { return true }
	i := 0
	readPassword = func() ([]byte, error) {
		if i < len(lines) {
			i++
			return []byte(lines[i-1]), nil
		}
		if windowsEOF && i == len(lines) {
			i++
			return []byte("\x1a"), nil
		}
		return nil, io.EOF
	}
}

func readValue(t *testing.T, isFile bool) (string, string, error) {
	t.Helper()
	c := &cobra.Command{}
	var errOut bytes.Buffer
	c.SetErr(&errOut)
	v, err := readSecretValue(c, isFile)
	return v, errOut.String(), err
}

// A file secret can be entered at the terminal, hidden, across several lines.
// It used to be refused with advice to write the secret to a file on disk
// (`dotf secrets set <id> < file`), which is exactly where a secret must not go.
// Hit on 2026-09-23 rotating STRIPE_BACKUP_CODE, a one-line code exposed as a file.
func TestReadSecretValueTakesAHiddenMultiLineFileSecret(t *testing.T) {
	cases := []struct {
		name       string
		lines      []string
		windowsEOF bool
		want       string
	}{
		{"one line stays as typed", []string{"PLANTED-code"}, false, "PLANTED-code"},
		{"several lines keep blanks and end with a newline", []string{"PLANTED-a", "", "PLANTED-b"}, false, "PLANTED-a\n\nPLANTED-b\n"},
		{"Ctrl-Z then Enter ends input on Windows", []string{"PLANTED-a", "PLANTED-b"}, true, "PLANTED-a\nPLANTED-b\n"},
		{"Ctrl-Z inside a line is data, not the end of input", []string{"PLANTED-a\x1aPLANTED-b"}, false, "PLANTED-a\x1aPLANTED-b"},
		{"nothing entered is empty, which the callers refuse", nil, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			terminalLines(t, tc.lines, tc.windowsEOF)
			got, prompt, err := readValue(t, true)
			if err != nil {
				t.Fatalf("a file secret must be enterable at the terminal: %v", err)
			}
			if got != tc.want {
				t.Errorf("value = %q, want %q", got, tc.want)
			}
			if !strings.Contains(prompt, "Ctrl-D") || strings.Contains(prompt, "PLANTED") {
				t.Errorf("the prompt must say how to finish and must not echo the value: %q", prompt)
			}
		})
	}
}
