package tools

import (
	"errors"
	"testing"
)

// Banners captured from real tools. The first two are the ones that broke the
// "last token of the first line" parse in the setup scripts.
func TestProbeVersion_ExtractsTheSemverFromRealBanners(t *testing.T) {
	cases := []struct {
		name, out, want string
	}{
		{"opencode banner line first", "OpenCode locked.\n1.16.2\n", "1.16.2"},
		{"hive prefixed name", "hive-vault 3.0.0\n", "3.0.0"},
		{"pi bare", "0.84.3\n", "0.84.3"},
		{"dotf prefixed", "dotf version 0.51.0\n", "0.51.0"},
		{"jq with a suffix", "jq-1.7.1\n", "1.7.1"},
		{"no version at all", "usage: thing [options]\n", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ProbeVersion("x", func(string, ...string) ([]byte, error) { return []byte(tc.out), nil })
			if got != tc.want {
				t.Errorf("ProbeVersion(%q) = %q, want %q", tc.out, got, tc.want)
			}
		})
	}
}

func TestProbeVersion_UsesOutputEvenWhenTheToolExitsNonZero(t *testing.T) {
	got := ProbeVersion("x", func(string, ...string) ([]byte, error) {
		return []byte("tool 2.3.4\nerror: telemetry endpoint unreachable\n"), errors.New("exit status 1")
	})
	if got != "2.3.4" {
		t.Errorf("a version printed before an unrelated error must still count, got %q", got)
	}
}

func TestProbeVersion_AbsentToolIsEmpty(t *testing.T) {
	got := ProbeVersion("x", func(string, ...string) ([]byte, error) { return nil, errors.New("not found") })
	if got != "" {
		t.Errorf("absent tool must probe as empty, got %q", got)
	}
}
