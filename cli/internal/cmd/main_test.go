package cmd

import (
	"os"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

// TestMain keeps every git these tests start from forking a background gc that
// outlives the test (#2803).
//
// With fakeDotfEnv set, the test binary is not running tests: `dotf update`'s
// tests installed it as the release dotf and run it as one (runAsReleaseDotf).
func TestMain(m *testing.M) {
	if os.Getenv(fakeDotfEnv) == "1" {
		os.Exit(runAsReleaseDotf(os.Args[1:]))
	}
	gittest.Quiet()
	os.Exit(m.Run())
}
