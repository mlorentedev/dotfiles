package secrets

import (
	"strings"
	"testing"
)

func envCount(env []string, name string) (n int, last string) {
	for _, kv := range env {
		if k, v, _ := strings.Cut(kv, "="); strings.EqualFold(k, name) {
			n, last = n+1, v
		}
	}
	return n, last
}

// #1008: the session `dotf secrets backup` acquires reaches the bw child alone,
// and replaces an ambient one rather than racing it.
func TestSessionEnv_ReplacesAnAmbientSession(t *testing.T) {
	got := sessionEnv([]string{"PATH=/bin", "BW_SESSION=ambient", "bw_session=lower"}, "acquired")
	if n, v := envCount(got, "BW_SESSION"); n != 1 || v != "acquired" {
		t.Errorf("BW_SESSION entries = %d (last %q), want exactly the acquired one: %v", n, v, got)
	}
	if n, _ := envCount(got, "PATH"); n != 1 {
		t.Errorf("PATH dropped: %v", got)
	}
}

// With no acquired session the child inherits the environment untouched, so an
// operator who prefixed BW_SESSION keeps working as before.
func TestSessionEnv_NoSessionKeepsTheAmbientOne(t *testing.T) {
	base := []string{"PATH=/bin", "BW_SESSION=ambient"}
	got := sessionEnv(base, "")
	if n, v := envCount(got, "BW_SESSION"); n != 1 || v != "ambient" {
		t.Errorf("got %v, want the ambient session kept", got)
	}
}

// The password reaches `bw unlock` through one named variable, never argv, and a
// stale session in the parent cannot short-circuit the unlock.
func TestUnlockEnv_CarriesThePasswordInItsOwnVariable(t *testing.T) {
	got := unlockEnv([]string{"PATH=/bin", "BW_SESSION=stale"}, []byte("hunter2"))
	if n, v := envCount(got, bwUnlockPasswordVar); n != 1 || v != "hunter2" {
		t.Errorf("%s entries = %d (%q), want one carrying the password", bwUnlockPasswordVar, n, v)
	}
	if n, _ := envCount(got, "BW_SESSION"); n != 0 {
		t.Errorf("a stale BW_SESSION reached bw unlock: %v", got)
	}
}
