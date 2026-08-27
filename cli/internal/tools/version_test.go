package tools

import (
	"errors"
	"testing"
)

// One table for every branch of ProbeVersion: real banners (the first two are
// the ones that broke the setup scripts' "last token of the first line"
// parse), a version printed before an unrelated non-zero exit, and an absent
// tool.
func TestProbeVersion(t *testing.T) {
	cases := []struct {
		name string
		out  string
		err  error
		want string
	}{
		{name: "opencode banner line first", out: "OpenCode locked.\n1.16.2\n", want: "1.16.2"},
		{name: "hive prefixed name", out: "hive-vault 3.0.0\n", want: "3.0.0"},
		{name: "pi bare", out: "0.84.3\n", want: "0.84.3"},
		{name: "dotf prefixed", out: "dotf version 0.51.0\n", want: "0.51.0"},
		{name: "jq with a suffix", out: "jq-1.7.1\n", want: "1.7.1"},
		{name: "no version at all", out: "usage: thing [options]\n", want: ""},
		{name: "empty output", out: "", want: ""},
		{name: "version printed before an unrelated non-zero exit", out: "tool 2.3.4\nerror: telemetry endpoint unreachable\n", err: errors.New("exit status 1"), want: "2.3.4"},
		{name: "absent tool", err: errors.New("not found"), want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ProbeVersion("x", func(string, ...string) ([]byte, error) { return []byte(tc.out), tc.err })
			if got != tc.want {
				t.Errorf("ProbeVersion(%q, err=%v) = %q, want %q", tc.out, tc.err, got, tc.want)
			}
		})
	}
}
