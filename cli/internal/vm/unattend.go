package vm

import (
	"embed"
	"os"
	"path/filepath"
)

// unattend is what Windows Setup reads from the build's USB stick.
//
//go:embed windows/autounattend.xml windows/firstlogon.ps1
var unattend embed.FS

// WriteUnattend fills dir with the unattended install's files, replacing
// whatever an earlier build left there.
func WriteUnattend(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"autounattend.xml", "firstlogon.ps1"} {
		b, err := unattend.ReadFile("windows/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
