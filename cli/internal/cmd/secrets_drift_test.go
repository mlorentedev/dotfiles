package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// stubLister is a fixed inventory for the drift command.
type stubLister struct {
	items   []secrets.ItemSummary
	folders []string
}

func (s stubLister) ListItems() ([]secrets.ItemSummary, error) { return s.items, nil }
func (s stubLister) ListFolders() ([]string, error)            { return s.folders, nil }

func runDrift(t *testing.T, l secrets.BWLister) (string, error) {
	t.Helper()
	useTempRegistry(t, `
version: 1
secrets:
  - {id: DOCKERHUB_TOKEN, plane: app, backend: bw, bw: {item: dockerhub, field: PAT, folder: Dotfiles/apps}, expose: {env: DOCKERHUB_TOKEN}}
`)
	old := bwLister
	bwLister = l
	t.Cleanup(func() { bwLister = old })
	c := newSecretsDriftCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(nil)
	err := c.Execute()
	return out.String(), err
}

// AC8, the half no test held: a finding makes the command exit non-zero, so a
// hook or CI step can gate on it — and each line names the registry id to edit.
func TestDriftExitsNonZeroOnAFindingAndNamesTheSecret(t *testing.T) {
	out, err := runDrift(t, stubLister{
		items:   []secrets.ItemSummary{{Name: "dockerhub", Fields: []string{"PAT"}}},
		folders: []string{"Dotfiles/apps"},
	})
	if err == nil {
		t.Fatalf("a misfiled item must fail the command:\n%s", out)
	}
	if !strings.Contains(out, "item-misfiled") || !strings.Contains(out, "[DOCKERHUB_TOKEN]") {
		t.Errorf("the finding must be printed with its registry id:\n%s", out)
	}
}

func TestDriftExitsZeroOnAMatchingStore(t *testing.T) {
	out, err := runDrift(t, stubLister{
		items:   []secrets.ItemSummary{{Name: "dockerhub", Folder: "Dotfiles/apps", Fields: []string{"PAT"}}},
		folders: []string{"Dotfiles/apps"},
	})
	if err != nil {
		t.Fatalf("a matching store must exit 0: %v\n%s", err, out)
	}
}
