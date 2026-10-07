package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/prland"
)

func TestPrLand_ARefusalListsTheReasonsAndExitsNonZero(t *testing.T) {
	saved := prLandOptions
	prLandOptions = func(string, string) prland.Options {
		return prland.Options{
			Run: func(_ context.Context, args ...string) ([]byte, error) {
				if args[1] == "view" {
					return []byte(`{"state":"OPEN","isDraft":false,"headRefOid":"abc1234567","headRefName":"feat/x","baseRefName":"main","mergeStateStatus":"BEHIND"}`), nil
				}
				return []byte(`[{"name":"test","bucket":"pass"}]`), nil
			},
			Untriaged: func(context.Context) ([]int, error) { return nil, nil },
		}
	}
	t.Cleanup(func() { prLandOptions = saved })

	stdout, _, err := execute(t, "pr", "land", "#30")
	if err == nil {
		t.Fatal("a refusal exited 0")
	}
	if !strings.Contains(stdout, "[NOT MERGED] #30 at abc1234") || !strings.Contains(stdout, "merge state is BEHIND") {
		t.Errorf("refusal output:\n%s", stdout)
	}
}

func TestPrLand_RejectsANonNumber(t *testing.T) {
	if _, _, err := execute(t, "pr", "land", "feat/x"); err == nil {
		t.Fatal("a non-number was accepted")
	}
}
