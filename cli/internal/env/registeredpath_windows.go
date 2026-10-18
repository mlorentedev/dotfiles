//go:build windows

package env

import (
	"errors"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// RegisteredPath is the PATH Windows hands a new process: the machine's
// directories, then the user's, each expanded. An installer registers where it
// placed a tool there (an MSI, Git for Windows, winget's portable packages when
// it cannot create links), but a running process keeps the PATH it started
// with. A step that installs a tool and then runs it reads this back, the same
// read setup-windows.ps1's Sync-SessionPath does. It never writes.
func RegisteredPath() ([]string, error) {
	var dirs []string
	for _, k := range []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`},
		{registry.CURRENT_USER, `Environment`},
	} {
		v, err := registryPath(k.root, k.path)
		if err != nil {
			return nil, err
		}
		for _, dir := range filepath.SplitList(v) {
			if dir != "" {
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs, nil
}

// registryPath reads one scope's Path, expanding a REG_EXPAND_SZ value: the
// stored %USERPROFILE%\... form is no directory until it is expanded. An
// absent key or value is an empty PATH, not an error.
func registryPath(root registry.Key, path string) (string, error) {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = k.Close() }()
	v, typ, err := k.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if typ == registry.EXPAND_SZ {
		return registry.ExpandString(v)
	}
	return v, nil
}
