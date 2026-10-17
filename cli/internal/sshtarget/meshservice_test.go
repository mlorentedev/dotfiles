package sshtarget

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func answering(t *testing.T, want []string, out string, err error) Runner {
	return func(argv ...string) ([]byte, error) {
		if !slices.Equal(argv, want) {
			t.Fatalf("ran %v, want only %v", argv, want)
		}
		return []byte(out), err
	}
}

func TestMeshAtBootOnDarwinReadsTheTailscaledDaemon(t *testing.T) {
	probe := []string{"launchctl", "print", "system/com.tailscale.tailscaled"}
	if note := MeshAtBootFor("darwin", answering(t, probe, "state = running\n", nil))(); note != "" {
		t.Errorf("a loaded tailscaled daemon → %q, want no note", note)
	}
	note := MeshAtBootFor("darwin", answering(t, probe, "Could not find service", errors.New("exit status 113")))()
	if !strings.Contains(note, "only once someone logs in") || !strings.Contains(note, "FileVault") {
		t.Errorf("the app without the daemon → %q, want it to say a reboot needs a login", note)
	}
}

func TestMeshAtBootOnLinuxReadsTheSystemdUnit(t *testing.T) {
	probe := []string{"systemctl", "is-enabled", "tailscaled"}
	cases := []struct {
		out   string
		err   error
		state string // "" when no note is due
	}{
		{"enabled\n", nil, ""},
		{"disabled\n", errors.New("exit status 1"), "tailscaled is disabled"},
		{"masked\n", errors.New("exit status 1"), "tailscaled is masked"},
		{"Failed to get unit file state for tailscaled.service: No such file or directory\n", errors.New("exit status 1"), "tailscaled is not installed"},
		{"", errors.New("exit status 1"), "tailscaled is not installed"},
	}
	for _, c := range cases {
		note := MeshAtBootFor("linux", answering(t, probe, c.out, c.err))()
		if c.state == "" {
			if note != "" {
				t.Errorf("%q → %q, want no note", c.out, note)
			}
			continue
		}
		if !strings.HasPrefix(note, c.state) || !strings.Contains(note, "systemctl enable --now tailscaled") {
			t.Errorf("%q → %q, want %q and the command that fixes it", c.out, note, c.state)
		}
	}
}

func TestMeshAtBootIsNotReadOnOtherOSes(t *testing.T) {
	if f := MeshAtBootFor("windows", nil); f != nil {
		t.Error("windows reads the mesh service; it has no reader yet")
	}
}

// Inspect carries the note into the plan, which never makes it pending: the
// target reports the mesh service and does not converge it.
func TestInspectCarriesTheMeshNoteWithoutMakingItPending(t *testing.T) {
	tgt := newMachine(t).target()
	tgt.MeshAtBoot = func() string { return "after a reboot the node rejoins the mesh only once someone logs in" }
	p, err := tgt.Inspect()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.MeshAtBoot, "someone logs in") {
		t.Errorf("plan MeshAtBoot = %q", p.MeshAtBoot)
	}
	if slices.ContainsFunc(p.Pending, func(r string) bool { return strings.Contains(r, "mesh") || strings.Contains(r, "tailscale") }) {
		t.Errorf("the mesh service made the plan pending: %v", p.Pending)
	}
}
