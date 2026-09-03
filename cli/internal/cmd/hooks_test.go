package cmd

import (
	"strings"
	"testing"
)

// TestHooksInstallIsReachableFromRoot is the wiring assertion CLI-071's
// adversarial review taught us to write: every test in internal/hooks proves the
// FUNCTIONS, and none of them would notice if nobody called them. The reviewer
// demonstrated that gap on prtriage by deleting two invocations from doctor.Run
// and watching the suite stay green.
//
// So this asserts the path a user actually takes — `dotf hooks install` resolves
// from the root command — rather than that a constructor exists.
func TestHooksInstallIsReachableFromRoot(t *testing.T) {
	root := New("test", "")

	cmd, _, err := root.Find([]string{"hooks", "install"})
	if err != nil {
		t.Fatalf("`dotf hooks install` does not resolve: %v — is newHooksCmd still registered in root.go?", err)
	}
	if cmd.Name() != "install" {
		t.Fatalf("resolved %q, want the install subcommand — `hooks` alone resolving means the child is missing", cmd.Name())
	}
	if cmd.RunE == nil {
		t.Error("the subcommand resolves but does nothing")
	}

	// The two flags are the seam setup depends on: without them the setup
	// scripts cannot point the installer at a checkout that is not the deploy
	// mirror, which is exactly what setup-linux.sh does today.
	for _, flag := range []string{"source", "dotfiles-dir"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("--%s is missing; the setup scripts pass it", flag)
		}
	}
}

// TestHooksInstallRefusesArgs pins that the operand form is rejected. The shell
// twin took positional [src] [dotfilesDir], and a user carrying that habit over
// must get an error rather than have the arguments silently ignored.
func TestHooksInstallRefusesArgs(t *testing.T) {
	root := New("test", "")
	cmd, _, err := root.Find([]string{"hooks", "install"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if err := cmd.Args(cmd, []string{"/some/src"}); err == nil {
		t.Fatal("want positional arguments rejected: the twin accepted them and this does not")
	}
	if !strings.Contains(cmd.Use, "install") {
		t.Errorf("Use = %q", cmd.Use)
	}
}
