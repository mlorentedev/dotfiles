// Package bunenv drops the variables that point a Bun program at an inspector.
// Claude Code is a Bun program that runs its own inspector and exports
// BUN_INSPECT to the shells it starts; a Bun child (pi, opencode) that inherits
// it tries to bind the same port and exits before doing any work (#3615).
package bunenv

import "strings"

// InspectorPrefix starts every variable Bun reads to start or reach an
// inspector: BUN_INSPECT itself and its BUN_INSPECT_* companions. A prefix
// rather than a list, so a companion Bun adds later is dropped too.
const InspectorPrefix = "BUN_INSPECT"

// WithoutInspector returns environ without the inspector variables. Keys
// compare upper-cased, because Windows environment names are case-insensitive.
func WithoutInspector(environ []string) []string {
	clean := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if upper != InspectorPrefix && !strings.HasPrefix(upper, InspectorPrefix+"_") {
			clean = append(clean, entry)
		}
	}
	return clean
}
