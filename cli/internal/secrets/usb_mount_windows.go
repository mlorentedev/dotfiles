package secrets

import (
	"os"
	"path/filepath"
	"strings"
)

// isMountPoint reports whether dir is the root of a drive other than the
// system one (E:\), which is where VeraCrypt mounts a volume on Windows. A
// folder on a drive is not one, and neither is C:\.
func isMountPoint(dir string) (bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	vol := filepath.VolumeName(abs)
	root := vol != "" && strings.TrimRight(abs, `\/`) == vol
	return root && !strings.EqualFold(vol, os.Getenv("SystemDrive")), nil
}
