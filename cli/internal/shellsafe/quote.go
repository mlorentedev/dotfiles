package shellsafe

import (
	"strings"
)

// Bash renders one POSIX single-quoted argument safely.
// It wraps the value in single quotes and escapes existing single quotes
// by closing the string, appending an escaped single quote, and reopening.
func Bash(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// PowerShell renders one PowerShell single-quoted argument safely.
// It wraps the value in single quotes and escapes existing single quotes
// by doubling them, which is the PowerShell standard.
func PowerShell(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
