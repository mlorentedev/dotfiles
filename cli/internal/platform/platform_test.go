package platform

import "testing"

func TestSupports(t *testing.T) {
	tests := []struct {
		name      string
		platforms []string
		goos      string
		want      bool
	}{
		{"absent means every OS", nil, "darwin", true},
		{"a listed OS", []string{"linux", "darwin"}, "darwin", true},
		{"an unlisted OS", []string{"linux", "darwin"}, "windows", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Supports(tt.platforms, tt.goos); got != tt.want {
				t.Errorf("Supports(%v, %q) = %v, want %v", tt.platforms, tt.goos, got, tt.want)
			}
		})
	}
}

func TestUnknown(t *testing.T) {
	if got := Unknown([]string{"linux", "darwin", "windows"}); got != "" {
		t.Errorf("every known platform: got %q", got)
	}
	// A misspelt platform makes a declaration unsupported everywhere, and an
	// unsupported entry is reported as a skip, so nothing else would say so.
	if got := Unknown([]string{"linux", "macos"}); got != "macos" {
		t.Errorf("a misspelt platform: got %q, want %q", got, "macos")
	}
}

func TestValidKey(t *testing.T) {
	for key, want := range map[string]bool{
		"linux": true, "darwin/arm64": true, "windows/amd64": true,
		"macos": false, "linux/x64": false, "darwin/arm64/extra": false, "": false,
	} {
		if got := ValidKey(key); got != want {
			t.Errorf("ValidKey(%q) = %v, want %v", key, got, want)
		}
	}
}
