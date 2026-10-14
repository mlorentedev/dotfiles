package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func execute(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := New("dev", "")
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errBuf.String(), err
}

func TestRootCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutSub string
	}{
		{
			name:       "help exits clean and prints usage",
			args:       []string{"--help"},
			wantErr:    false,
			wantOutSub: "Usage:",
		},
		{
			name:       "no args prints help instead of erroring",
			args:       []string{},
			wantErr:    false,
			wantOutSub: "Available Commands:",
		},
		{
			name:       "version prints the version",
			args:       []string{"version"},
			wantErr:    false,
			wantOutSub: "dotf version",
		},
		{
			name:    "unknown subcommand fails",
			args:    []string{"bogus"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := execute(t, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			combined := stdout + stderr
			if tt.wantOutSub != "" && !strings.Contains(combined, tt.wantOutSub) {
				t.Errorf("output %q does not contain %q", combined, tt.wantOutSub)
			}
		})
	}
}

// `dotf --version` and `-v` print the very line `dotf version` does, to stdout:
// install.{sh,ps1} regex that line for the installed semver.
func TestRootVersionFlag(t *testing.T) {
	run := func(args ...string) (string, string) {
		t.Helper()
		c := New("1.2.3", "abc")
		var out, errBuf bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&errBuf)
		c.SetArgs(args)
		if err := c.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return out.String(), errBuf.String()
	}
	want, _ := run("version")
	if want != "dotf version 1.2.3\n" {
		t.Fatalf("dotf version = %q", want)
	}
	for _, flag := range []string{"--version", "-v"} {
		if out, errOut := run(flag); out != want || errOut != "" {
			t.Errorf("dotf %s: stdout %q, stderr %q; want stdout %q", flag, out, errOut, want)
		}
	}
}
