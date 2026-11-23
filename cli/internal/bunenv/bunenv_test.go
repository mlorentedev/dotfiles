package bunenv

import (
	"slices"
	"testing"
)

func TestWithoutInspector_DropsTheInspectorVariablesAndKeepsEverythingElse(t *testing.T) {
	got := WithoutInspector([]string{
		"BUN_INSPECT=ws://127.0.0.1:9221/x", "bun_inspect_connect_to=ws://y", "Bun_Inspect_Notify=unix:///n",
		"BUN_INSPECTOR=keep", "BUN_INSTALL=/b", "NAN_API_KEY=k", "PATH=/p", "NOEQUALS",
	})
	want := []string{"BUN_INSPECTOR=keep", "BUN_INSTALL=/b", "NAN_API_KEY=k", "PATH=/p", "NOEQUALS"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
