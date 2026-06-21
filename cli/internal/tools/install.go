package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Result classifies what Install did, for an honest human message and tests.
type Result int

const (
	// Skipped: the tool is already present at or above the pinned version, so
	// nothing was downloaded (never downgrade — REFACTOR-011/013).
	Skipped Result = iota
	// Installed: the tool was absent and the pinned binary was placed.
	Installed
	// Upgraded: a below-pin binary was replaced with the pinned one.
	Upgraded
)

func (r Result) String() string {
	switch r {
	case Installed:
		return "installed"
	case Upgraded:
		return "upgraded"
	default:
		return "skipped"
	}
}

// Fetcher downloads url into the local file destPath. It is the seam that lets
// tests exercise the verify/reconcile logic with no network: the default
// (HTTPFetch) performs a real GET, and a non-200 — including a 404 for a missing
// release asset, the offline case — is an error.
type Fetcher func(url, destPath string) error

// HTTPFetch is the production Fetcher: GET url, stream the body to destPath, and
// error on any non-200 so a missing asset or checksums file aborts cleanly.
func HTTPFetch(url, destPath string) error {
	resp, err := http.Get(url) //nolint:gosec // url is built from the pinned catalog, not user input
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", destPath, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	return nil
}

// Installer fetches, verifies (sha256 vs the release checksums), and places a
// catalog tool's release binary in Dest. It is the Go consolidation of the
// install-dotf.{sh,ps1} bootstrap pattern, generalised from a single CLI to any
// github-release tool in packages.json (CLI-029 PR-B). Unlike install-dotf, sops
// ships raw (un-archived) binaries, so there is no extraction step.
type Installer struct {
	GOOS, GOARCH string    // target platform; default runtime.GOOS/GOARCH
	Dest         string    // install dir; default ~/.local/bin
	BaseURL      string    // release host; default https://github.com (tests override)
	Fetch        Fetcher   // download seam; default HTTPFetch
	Out          io.Writer // progress sink; default os.Stdout
	// CurrentVersion reports the installed version of the named tool, or "" when
	// absent/unparseable. The reconcile seam — default runs <Dest>/<bin> --version.
	CurrentVersion func(name string) string
}

func (in *Installer) defaults() {
	if in.GOOS == "" {
		in.GOOS = runtime.GOOS
	}
	if in.GOARCH == "" {
		in.GOARCH = runtime.GOARCH
	}
	if in.BaseURL == "" {
		in.BaseURL = "https://github.com"
	}
	if in.Fetch == nil {
		in.Fetch = HTTPFetch
	}
	if in.Out == nil {
		in.Out = os.Stdout
	}
	if in.CurrentVersion == nil {
		in.CurrentVersion = in.installedVersion
	}
}

// Install converges the tool to its pinned version: a no-op when already at or
// above the pin, otherwise download → verify sha256 → place + chmod. A failure
// at any step leaves Dest untouched (the binary is staged in a temp dir and only
// moved into place after verification passes).
func (in *Installer) Install(t Tool) (Result, error) {
	in.defaults()

	if t.Source.Type != "github-release" {
		return Skipped, fmt.Errorf("%s: unsupported source type %q", t.Name, t.Source.Type)
	}
	asset := t.AssetName(in.GOOS, in.GOARCH)
	if asset == "" {
		return Skipped, fmt.Errorf("%s: no release asset for %s/%s", t.Name, in.GOOS, in.GOARCH)
	}
	sumsName := t.ChecksumsName(in.GOARCH)
	if sumsName == "" {
		return Skipped, fmt.Errorf("%s: no checksums file declared — refusing to install unverified", t.Name)
	}

	switch decideAction(in.CurrentVersion(t.Name), t.Version) {
	case actionSkip:
		fmt.Fprintf(in.Out, "%s %s already installed; skipping\n", t.Name, t.Version)
		return Skipped, nil
	case actionUpgrade:
		return in.fetchVerifyPlace(t, asset, sumsName, Upgraded)
	default: // actionInstall
		return in.fetchVerifyPlace(t, asset, sumsName, Installed)
	}
}

// fetchVerifyPlace runs the download → verify → place pipeline and returns res
// on success. Everything happens in a temp dir; Dest is written only after the
// checksum matches, so a mismatch or download error never leaves a partial or
// unverified binary behind.
func (in *Installer) fetchVerifyPlace(t Tool, asset, sumsName string, res Result) (Result, error) {
	tmp, err := os.MkdirTemp("", "dotf-tools-")
	if err != nil {
		return Skipped, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	assetPath := filepath.Join(tmp, asset)
	sumsPath := filepath.Join(tmp, sumsName)
	if err := in.Fetch(in.releaseURL(t.Source.Repo, t.Version, asset), assetPath); err != nil {
		return Skipped, fmt.Errorf("%s: %w", t.Name, err)
	}
	if err := in.Fetch(in.releaseURL(t.Source.Repo, t.Version, sumsName), sumsPath); err != nil {
		return Skipped, fmt.Errorf("%s: %w", t.Name, err)
	}
	if err := verifyChecksum(assetPath, sumsPath, asset); err != nil {
		return Skipped, fmt.Errorf("%s: %w", t.Name, err)
	}

	dst := filepath.Join(in.Dest, binFilename(t.Name, in.GOOS))
	if err := placeBinary(assetPath, dst); err != nil {
		return Skipped, fmt.Errorf("%s: %w", t.Name, err)
	}
	fmt.Fprintf(in.Out, "%s %s %s to %s\n", t.Name, t.Version, res, dst)
	return res, nil
}

// releaseURL builds the GitHub release download URL: <base>/<repo>/releases/download/v<version>/<file>.
func (in *Installer) releaseURL(repo, version, file string) string {
	return fmt.Sprintf("%s/%s/releases/download/v%s/%s", strings.TrimRight(in.BaseURL, "/"), repo, version, file)
}

// installedVersion is the default CurrentVersion: run <Dest>/<bin> --version and
// pull out the first dotted-numeric token. Absent binary or any failure → "".
func (in *Installer) installedVersion(name string) string {
	bin := filepath.Join(in.Dest, binFilename(name, in.GOOS))
	if _, err := os.Stat(bin); err != nil {
		return ""
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	if m := semverRE.Find(out); m != nil {
		return string(m)
	}
	return ""
}

var semverRE = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)

// binFilename is the on-disk command name: the catalog name plus ".exe" on
// Windows. The release asset (e.g. sops-v3.13.1.linux.amd64) is renamed to this
// so the tool resolves on PATH as `sops`.
func binFilename(name, goos string) string {
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}

// placeBinary copies src to dst (creating Dest), 0755 so it is executable. A
// copy (not rename) because the temp dir may be on a different filesystem.
func placeBinary(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o755)
}

// verifyChecksum confirms assetPath's sha256 matches the entry for assetName in
// the GNU-coreutils-format manifest at sumsPath ("<hex>  <filename>" lines, the
// same format dotf and sops both ship). A missing entry or a mismatch is an
// error — the security gate of the whole installer.
func verifyChecksum(assetPath, sumsPath, assetName string) error {
	want, err := expectedChecksum(sumsPath, assetName)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(assetPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s (want %s, got %s)", assetName, want, got)
	}
	return nil
}

// expectedChecksum returns the recorded hash for assetName, or an error when the
// manifest does not list it (an unverifiable asset must not be installed).
func expectedChecksum(sumsPath, assetName string) (string, error) {
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("%s not listed in %s", assetName, filepath.Base(sumsPath))
}

// action is the reconcile decision for one tool.
type action int

const (
	actionSkip    action = iota // present and >= pin: leave it (never downgrade)
	actionInstall               // absent: place the pinned binary
	actionUpgrade               // present but below pin: replace with the pinned binary
)

// decideAction encodes the reconcile policy: the pin is a MINIMUM, not an exact
// version. Install when absent; upgrade when the installed version is below the
// pin; skip when it is at or above the pin so a newer install is never
// downgraded (the REFACTOR-011/013 lesson that "presence is not convergence" and
// an exact-match reconcile would wrongly downgrade). An unparseable installed
// version compares as below the pin, so a broken install is re-fetched.
func decideAction(installed, pin string) action {
	if installed == "" {
		return actionInstall
	}
	if atLeast(installed, pin) {
		return actionSkip
	}
	return actionUpgrade
}

// compareVersions compares dot-separated numeric versions: -1 if a < b, 0 if
// equal, +1 if a > b, with a missing component treated as 0 ("3.13" == "3.13.0")
// and non-numeric leading text stripped. It mirrors doctor.compareVersions
// (unexported there); consolidating both into a shared package is a tracked
// follow-up, deferred here to keep PR-B from touching cli/internal/doctor.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, bi := component(as, i), component(bs, i)
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

func atLeast(actual, min string) bool { return compareVersions(actual, min) >= 0 }

// component returns the integer value of the i-th dotted field (leading digits
// only), or 0 when absent.
func component(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	digits := strings.TrimLeftFunc(parts[i], func(r rune) bool { return r < '0' || r > '9' })
	end := 0
	for end < len(digits) && digits[end] >= '0' && digits[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(digits[:end])
	return n
}
