package sshtarget

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotElevated is what Elevated.Ready returns in a process without
// administrator rights. Converge reports the step skipped with its plan, as it
// does for ErrNoTerminal on macOS and Linux.
var ErrNotElevated = errors.New("administrator rights are needed and this process has none: run dotf converge from an elevated terminal (Run as administrator)")

// highIntegritySID is the mandatory label of an elevated token. `whoami
// /groups` names it in the user's language but always prints the SID, so the
// SID is what Ready reads.
const highIntegritySID = "S-1-16-12288"

// The SIDs that own the administrators' authorized keys, as the ps1 granted
// them: BUILTIN\Administrators and SYSTEM.
const (
	sidAdministrators = "S-1-5-32-544"
	sidSystem         = "S-1-5-18"
)

// Elevated is Root on Windows. sudo caches one password for every command
// after it; UAC has no such cache, and a prompt per command could not be
// collected by a batch. So the apply runs only in a process that is already
// elevated: Ready checks the token, Run runs the command as it is, and files
// are written by the process itself rather than through `install`.
type Elevated struct {
	Exec Runner
}

func (e Elevated) Ready() error {
	out, err := e.Exec("whoami", "/groups")
	if err != nil {
		return fmt.Errorf("whoami /groups: %w: %s", err, bytes.TrimSpace(out))
	}
	if !bytes.Contains(out, []byte(highIntegritySID)) {
		return ErrNotElevated
	}
	return nil
}

func (e Elevated) Run(argv ...string) ([]byte, error) { return e.Exec(argv...) }

// Install writes data to path, replacing it in one rename. A file under
// %ProgramData%\ssh inherits that directory's ACL, which sshd accepts.
func (e Elevated) Install(path string, data []byte) error {
	return replaceFile(path, data, nil)
}

// InstallAdminOnly is Install for a file sshd reads only when Administrators
// and SYSTEM alone may touch it: administrators_authorized_keys. The ACL is
// set on the temporary file before the rename, so the file is never readable
// by anyone else, even for a moment.
func (e Elevated) InstallAdminOnly(path string, data []byte) error {
	return replaceFile(path, data, func(tmp string) error {
		return asRoot(e, "icacls", tmp, "/setowner", "*"+sidAdministrators)
	}, func(tmp string) error {
		return asRoot(e, "icacls", tmp, "/inheritance:r",
			"/grant:r", "*"+sidAdministrators+":F", "/grant:r", "*"+sidSystem+":F")
	})
}

func (e Elevated) Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// replaceFile writes data next to path, runs prepare on the temporary file,
// and renames it over path.
func replaceFile(path string, data []byte, prepare ...func(tmp string) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	for _, p := range prepare {
		if p == nil {
			continue
		}
		if err := p(tmp); err != nil {
			return err
		}
	}
	return os.Rename(tmp, path)
}
