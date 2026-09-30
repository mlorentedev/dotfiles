package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// internalRef matches references that mean something only inside this
// repository: ticket and decision ids (ADR-036, CLI-067), issue numbers (#1381)
// and the migration jargon "twin". Help text is read by people who have none of
// that context, so it names behaviour instead. Code comments keep the ids.
var internalRef = regexp.MustCompile(`\b(ADR|GUARD|CLI|OPS|BUG|HARNESS|SEC|AI|TOOL|WIN|SDD|MEMORY|REFACTOR|TEST|CI|SKILL|DOC|INFRA|SPEC|APP|HIVE)-[0-9]+|#[0-9]{2,5}\b|\btwins?\b`)

// TestHelpTextHasNoInternalReferences walks every command and checks each piece
// of text `--help` prints: the short and long descriptions, the examples and
// every flag's usage line.
func TestHelpTextHasNoInternalReferences(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		check := func(field, text string) {
			for _, line := range strings.Split(text, "\n") {
				if m := internalRef.FindString(line); m != "" {
					t.Errorf("%q %s mentions %q: %s", c.CommandPath(), field, m, strings.TrimSpace(line))
				}
			}
		}
		check("Short", c.Short)
		check("Long", c.Long)
		check("Example", c.Example)
		c.LocalFlags().VisitAll(func(f *pflag.Flag) { check("flag --"+f.Name, f.Usage) })
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(New("dev", ""))
}
