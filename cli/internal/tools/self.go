package tools

import (
	"fmt"
	"path/filepath"
)

// DotfRelease is dotf's own release at version, as goreleaser publishes it and
// install.sh / install.ps1 download it: a .tar.gz holding `dotf` on Linux and
// macOS, a .zip holding `dotf.exe` on Windows, verified against checksums.txt.
// It is not a packages.json entry: converge installs the catalog, and dotf is
// what runs converge (CLI-133).
func DotfRelease(version string) Tool {
	return Tool{
		Name:    "dotf",
		Version: version,
		Source: Source{
			Type: "github-archive",
			Repo: "mlorentedev/dotfiles",
			Asset: map[string]string{
				"linux":   "dotf_{version}_linux_{goarch}.tar.gz",
				"darwin":  "dotf_{version}_darwin_{goarch}.tar.gz",
				"windows": "dotf_{version}_windows_{goarch}.zip",
			},
			Member:    map[string]string{"linux": "dotf", "darwin": "dotf", "windows": "dotf.exe"},
			Checksums: "checksums.txt",
		},
	}
}

// InstallSelf installs release t at target, the path of the running binary,
// through the same download, checksum, extract, probe and place pipeline as a
// catalog tool. It installs whatever version t names, a lower one included: the
// pin is the contract. A target not named t.Name (t.Name.exe on Windows) is
// refused rather than installing a second binary beside it.
func (in *Installer) InstallSelf(t Tool, target string) error {
	in.defaults()
	if want := binFilename(t.Name, in.GOOS); filepath.Base(target) != want {
		return fmt.Errorf("%s: the running binary %s is not named %s; not replacing it", t.Name, target, want)
	}
	asset := t.AssetName(in.GOOS, in.GOARCH)
	if asset == "" {
		return fmt.Errorf("%s: no release asset for %s/%s", t.Name, in.GOOS, in.GOARCH)
	}
	in.Dest = filepath.Dir(target)
	_, err := in.fetchVerifyPlace(t, asset, t.ChecksumsName(in.GOARCH), Installed)
	return err
}
