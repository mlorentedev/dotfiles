package spec

import (
	"os"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gittest"
)

// TestMain keeps every git these tests start from forking a background gc that
// outlives the test (#2803).
func TestMain(m *testing.M) {
	gittest.Quiet()
	os.Exit(m.Run())
}
