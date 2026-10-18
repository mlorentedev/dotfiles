package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
)

// pr land holds a PR to the same registry as triage-queue: the loader it is
// handed must run when land asks what is untriaged, and a loader error must
// stop it before any queue is computed.
func TestPrLandOptions_UntriagedReadsTheRegistryItIsGiven(t *testing.T) {
	var asked int
	loader := func(context.Context) (prtriage.Registry, error) {
		asked++
		return prtriage.Registry{}, errors.New("registry unreadable")
	}

	_, err := prLandOptions("o/r", loader).Untriaged(t.Context())
	if err == nil || !strings.Contains(err.Error(), "registry unreadable") {
		t.Fatalf("want the loader's error, got %v", err)
	}
	if asked != 1 {
		t.Errorf("want the loader asked once, got %d", asked)
	}
}
