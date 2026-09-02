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
