//go:build !windows

package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// isMountPoint reports whether dir is the root of a mounted filesystem: its
// device differs from its parent's. A directory created as a mount point and
// never mounted shares its parent's device, which is the case this exists for.
func isMountPoint(dir string) (bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	dev := func(p string) (uint64, error) {
		info, err := os.Stat(p)
		if err != nil {
			return 0, err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return 0, fmt.Errorf("no device number for %s", p)
		}
		return uint64(st.Dev), nil //nolint:unconvert // int32 on darwin, uint64 on linux
	}
	here, err := dev(abs)
	if err != nil {
		return false, err
	}
	parent, err := dev(filepath.Dir(abs))
	if err != nil {
		return false, err
	}
	return here != parent, nil
}
