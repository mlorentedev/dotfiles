package shellsafe

import (
	"testing"
)

func TestBash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple string", "hello", "'hello'"},
		{"string with spaces", "hello world", "'hello world'"},
		{"string with single quotes", "hello 'world'", "'hello '\\''world'\\'''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bash(tt.input)
			if got != tt.expected {
				t.Errorf("Bash(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPowerShell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple string", "hello", "'hello'"},
		{"string with spaces", "hello world", "'hello world'"},
		{"string with single quotes", "hello 'world'", "'hello ''world'''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PowerShell(tt.input)
			if got != tt.expected {
				t.Errorf("PowerShell(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
