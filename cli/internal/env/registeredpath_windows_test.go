//go:build windows

package env

import (
	"strings"
	"testing"
)

// Every Windows machine's registered PATH holds System32, expanded: a read
// that missed the machine scope, or returned %SystemRoot% unexpanded, would
// not find it.
func TestRegisteredPath_ReadsTheMachineScopeExpanded(t *testing.T) {
	dirs, err := RegisteredPath()
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		if strings.Contains(dir, "%") {
			t.Errorf("unexpanded entry %q", dir)
		}
	}
	for _, dir := range dirs {
		if strings.EqualFold(strings.TrimRight(dir, `\`), `C:\Windows\System32`) {
			return
		}
	}
	t.Errorf("no System32 in the registered PATH: %v", dirs)
}
