package sshtarget

import (
	"errors"
	"strings"
	"testing"
)

// Only launchctl's 113 means launchd does not know sshd. Any other failure is
// a read that did not happen: taken as "not loaded", the step would plan an
// enable and a bootstrap as root over a service it never read (#2731).
func TestLaunchdEnabledReadsOnlyExit113AsNotLoaded(t *testing.T) {
	probe := []string{"launchctl", "print", "system/com.openssh.sshd"}
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"loaded":  {nil, true},
		"unknown": {exitErr(113), false},
	} {
		got, err := Launchd{Run: answering(t, probe, "", tc.err)}.Enabled()
		if err != nil || got != tc.want {
			t.Errorf("%s: Enabled() = %v, %v; want %v", name, got, err, tc.want)
		}
	}
	for name, failure := range map[string]error{
		"another exit status": exitErr(5),
		"no exit status":      errors.New("launchctl: no such file or directory"),
	} {
		_, err := Launchd{Run: answering(t, probe, "Bad request.", failure)}.Enabled()
		if err == nil || !strings.Contains(err.Error(), "system/com.openssh.sshd") {
			t.Errorf("%s: Enabled() = %v, want an error naming the service", name, err)
		}
	}
}
