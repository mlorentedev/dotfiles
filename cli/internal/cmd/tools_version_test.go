package cmd

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// The two banners that broke the setup scripts' token parse, plus the absent
// case: a shell caller tests stdout and the exit status, both must be right.
func TestToolsVersionCmd(t *testing.T) {
	orig := toolsVersionRunner
	t.Cleanup(func() { toolsVersionRunner = orig })
	toolsVersionRunner = func(name string, _ ...string) ([]byte, error) {
		switch name {
		case "hive":
			return []byte("hive-vault 3.0.0\n"), nil
		case "opencode":
			return []byte("OpenCode locked.\n1.16.2\n"), nil
		}
		return nil, errors.New("not found")
	}

	cases := []struct {
		name    string
		tool    string
		want    string
		wantErr bool
	}{
		{name: "prefixed banner", tool: "hive", want: "3.0.0"},
		{name: "banner line before the number", tool: "opencode", want: "1.16.2"},
		{name: "absent tool: empty stdout, exit error", tool: "nope", want: "", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newToolsVersionCmd()
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{tc.tool})
			err := c.Execute()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got := strings.TrimSpace(out.String()); got != tc.want {
				t.Errorf("stdout = %q, want %q", got, tc.want)
			}
		})
	}
}

// `dotf tools version` probes through tools.ProbeRunner, so a tool that crashes
// is not read at a version its warnings name (BUG-141).
func TestToolsVersion_DropsAFailedRunsStderr(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	out, err := toolsVersionRunner("sh", "-c", "echo 'ansible-core 2.21.5 supports Python up to 3.14' >&2; exit 3")
	if err == nil || len(out) != 0 {
		t.Errorf("a crash read as output %q (err %v), want none", out, err)
	}
}
