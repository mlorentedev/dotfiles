package cmd

import (
	"reflect"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

func TestSelectToolsForRolesFiltersOnlyNoArgRuns(t *testing.T) {
	cat := tools.Catalog{Tools: []tools.Tool{
		{Name: "common"},
		{Name: "ts-bridge", Roles: []string{"ts-bridge-client"}},
	}}

	got, err := selectToolsForRoles(cat, "", nil)
	if err != nil || !reflect.DeepEqual(toolNames(got), []string{"common"}) {
		t.Fatalf("no-role selection = %v, %v", toolNames(got), err)
	}
	got, err = selectToolsForRoles(cat, "", []string{"ts-bridge-client"})
	if err != nil || !reflect.DeepEqual(toolNames(got), []string{"common", "ts-bridge"}) {
		t.Fatalf("bridge selection = %v, %v", toolNames(got), err)
	}
	got, err = selectToolsForRoles(cat, "ts-bridge", nil)
	if err != nil || !reflect.DeepEqual(toolNames(got), []string{"ts-bridge"}) {
		t.Fatalf("explicit selection = %v, %v", toolNames(got), err)
	}
}

func toolNames(list []tools.Tool) []string {
	out := make([]string, 0, len(list))
	for _, tool := range list {
		out = append(out, tool.Name)
	}
	return out
}
