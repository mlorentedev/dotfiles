package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

// fakeDotfEnv turns the test binary into a stand-in release dotf (TestMain, runAsReleaseDotf).
const fakeDotfEnv = "DOTF_CMD_TEST_FAKE_DOTF"

// runAsReleaseDotf is the release dotf `dotf update` installs in these tests: the test
// binary itself, so it runs on every OS. `--version` answers $FAKE_DOTF_VERSION;
// `converge` records its arguments and DOTFILES_REPO_DIR to $FAKE_CONVERGE_OUT
// and exits with $FAKE_CONVERGE_RC.
func runAsReleaseDotf(args []string) int {
	if len(args) > 0 && args[0] == "--version" {
		fmt.Println("dotf version " + os.Getenv("FAKE_DOTF_VERSION"))
		return 0
	}
	rec := fmt.Sprintf("args=%s\nrepo=%s\n", strings.Join(args, " "), os.Getenv("DOTFILES_REPO_DIR"))
	if err := os.WriteFile(os.Getenv("FAKE_CONVERGE_OUT"), []byte(rec), 0o600); err != nil {
		return 99
	}
	if os.Getenv("FAKE_CONVERGE_RC") == "3" {
		return 3
	}
	return 0
}

// pinnedCheckout is a clean clone of an upstream whose versions.conf pins
// DOTF_VERSION=0.70.0, with the running dotf at a temp path named as a release
// dotf is. It returns the checkout and a second clone that can push to the
// upstream.
func pinnedCheckout(t *testing.T) (checkout, writer string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	files, home := convergeFixture(t)
	root := t.TempDir()
	upstream := filepath.Join(root, "up.git")
	writer, checkout = filepath.Join(root, "w"), filepath.Join(root, "dotfiles")
	gitIn(t, root, "init", "-q", "--bare", upstream)
	gitIn(t, root, "clone", "-q", upstream, writer)
	if err := os.CopyFS(writer, os.DirFS(files)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(writer, env.CheckoutMarker), []byte("# pins\nDOTF_VERSION=0.70.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, writer, "add", "-A")
	gitIn(t, writer, "commit", "-q", "-m", "one")
	gitIn(t, writer, "push", "-q", "origin", "HEAD")
	gitIn(t, root, "clone", "-q", upstream, checkout)

	t.Setenv("DOTFILES_REPO_DIR", checkout)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	// A shell that sourced a stale versions.conf exports its own pin; update
	// must still install the upstream's.
	t.Setenv("DOTF_VERSION", "0.0.1")
	return checkout, writer
}

// release serves dotf 0.70.0 for this OS from an httptest server whose binary
// is the test binary (fakeDotf), with checksums.txt listing sum ("" = the real
// one). It points update at the server and at a running dotf under a temp dir,
// and returns that dotf's path and the file a converge records to.
func release(t *testing.T, sum string) (running, converged string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(self) //nolint:gosec // the test binary itself
	if err != nil {
		t.Fatal(err)
	}
	member, asset := "dotf", fmt.Sprintf("dotf_0.70.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archive := tarGz(t, member, body)
	if runtime.GOOS == "windows" {
		member, asset = "dotf.exe", fmt.Sprintf("dotf_0.70.0_windows_%s.zip", runtime.GOARCH)
		archive = zipOf(t, member, body)
	}
	if sum == "" {
		h := sha256.Sum256(archive)
		sum = hex.EncodeToString(h[:])
	}
	files := map[string][]byte{
		"/mlorentedev/dotfiles/releases/download/v0.70.0/" + asset:      archive,
		"/mlorentedev/dotfiles/releases/download/v0.70.0/checksums.txt": []byte(sum + "  " + asset + "\n"),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	running = filepath.Join(t.TempDir(), member)
	if err := os.WriteFile(running, []byte("dotf 0.69.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	base, exe := dotfReleaseBase, dotfExecutable
	t.Cleanup(func() { dotfReleaseBase, dotfExecutable = base, exe })
	dotfReleaseBase = srv.URL
	dotfExecutable = func() (string, error) { return running, nil }

	converged = filepath.Join(t.TempDir(), "converge.out")
	t.Setenv(fakeDotfEnv, "1")
	t.Setenv("FAKE_DOTF_VERSION", "0.70.0")
	t.Setenv("FAKE_CONVERGE_OUT", converged)
	return running, converged
}

func tarGz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// runUpdateAs runs `dotf update` as a binary reporting version.
func runUpdateAs(t *testing.T, version string) (stdout string, err error) {
	t.Helper()
	t.Cleanup(func() { dotfVersion = "dev" })
	c := New(version, "")
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"update"})
	err = c.Execute()
	return out.String(), err
}

// noInProcessConverge fails the test if update converges with the binary it
// is replacing.
func noInProcessConverge(t *testing.T) {
	t.Helper()
	fixture := convergeOptions
	t.Cleanup(func() { convergeOptions = fixture })
	convergeOptions = func() converge.Options {
		o := fixture()
		o.RunSetup = func(converge.Env) error { t.Error("update converged with the binary it replaced"); return nil }
		return o
	}
}

func lastStatus(t *testing.T) string {
	t.Helper()
	state, err := env.StateDir()
	if err != nil {
		t.Fatal(err)
	}
	rec, err := update.ReadRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	return rec.Status
}

// CLI-133: on a current checkout, a dotf off the pin is replaced in Go, from
// the release, where it runs, and the new binary converges as a child told
// the checkout. No installer script runs, on any OS.
func TestUpdate_InstallsThePinAndConvergesWithIt(t *testing.T) {
	checkout, _ := pinnedCheckout(t)
	running, converged := release(t, "")
	noInProcessConverge(t)

	stdout, err := runUpdateAs(t, "0.69.0")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, stdout)
	}
	if got, _ := os.ReadFile(running); bytes.Equal(got, []byte("dotf 0.69.0")) { //nolint:gosec // test path
		t.Fatalf("the running dotf was not replaced:\n%s", stdout)
	}
	rec, rerr := os.ReadFile(converged) //nolint:gosec // test path
	if rerr != nil {
		t.Fatalf("the installed dotf never converged: %v\n%s", rerr, stdout)
	}
	if want := "args=converge\nrepo=" + checkout + "\n"; string(rec) != want {
		t.Errorf("the new binary converged with %q, want %q", rec, want)
	}
	if !strings.Contains(stdout, "0.69.0 -> 0.70.0") || lastStatus(t) != "reinstalled" {
		t.Errorf("status %q, output:\n%s", lastStatus(t), stdout)
	}
}

// CLI-133 AC1: a diverged checkout still gets the pinned dotf; only the
// converge waits, and the run is a skip that exits 0.
func TestUpdate_InstallsThePinOnADivergedCheckout(t *testing.T) {
	checkout, writer := pinnedCheckout(t)
	for dir, file := range map[string]string{writer: "upstream.txt", checkout: "local.txt"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		gitIn(t, dir, "add", file)
		gitIn(t, dir, "commit", "-q", "-m", file)
	}
	gitIn(t, writer, "push", "-q", "origin", "HEAD")
	running, converged := release(t, "")
	noInProcessConverge(t)

	stdout, err := runUpdateAs(t, "0.69.0")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, stdout)
	}
	if got, _ := os.ReadFile(running); bytes.Equal(got, []byte("dotf 0.69.0")) { //nolint:gosec // test path
		t.Fatalf("a diverged checkout left the running dotf in place:\n%s", stdout)
	}
	if _, serr := os.Stat(converged); serr == nil {
		t.Error("a diverged checkout converged")
	}
	if !strings.Contains(stdout, "converge skipped") || lastStatus(t) != "diverged" {
		t.Errorf("status %q, output:\n%s", lastStatus(t), stdout)
	}
}

// A converge that fails in the installed binary fails the run as a converge,
// with the pin reported installed.
func TestUpdate_TheNewBinarysConvergeFailing(t *testing.T) {
	_, _ = pinnedCheckout(t)
	_, _ = release(t, "")
	t.Setenv("FAKE_CONVERGE_RC", "3")
	noInProcessConverge(t)

	stdout, err := runUpdateAs(t, "0.69.0")
	if err == nil {
		t.Fatalf("a failed converge must fail update:\n%s", stdout)
	}
	if !strings.Contains(stdout, "0.70.0 (the DOTF_VERSION pin) installed; its converge failed") || lastStatus(t) != "converge-failed" {
		t.Errorf("status %q, output:\n%s", lastStatus(t), stdout)
	}
}

// CLI-133 AC3: a release that fails verification installs nothing: the running
// binary is byte-identical and the run fails as install-failed.
func TestUpdate_ABadChecksumInstallsNothing(t *testing.T) {
	_, _ = pinnedCheckout(t)
	running, converged := release(t, strings.Repeat("0", 64))

	stdout, err := runUpdateAs(t, "0.69.0")
	if err == nil {
		t.Fatalf("a checksum mismatch must fail update:\n%s", stdout)
	}
	if got, _ := os.ReadFile(running); !bytes.Equal(got, []byte("dotf 0.69.0")) { //nolint:gosec // test path
		t.Errorf("the running dotf changed to %d bytes", len(got))
	}
	if _, serr := os.Stat(converged); serr == nil {
		t.Error("converged after a failed install")
	}
	if !strings.Contains(err.Error(), "0.70.0") || lastStatus(t) != "install-failed" {
		t.Errorf("status %q, error %v", lastStatus(t), err)
	}
}

// A dotf already at the pin installs nothing.
func TestUpdate_LeavesADotfAtThePinAlone(t *testing.T) {
	_, _ = pinnedCheckout(t)
	running, _ := release(t, "")

	stdout, err := runUpdateAs(t, "0.70.0")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, stdout)
	}
	if got, _ := os.ReadFile(running); !bytes.Equal(got, []byte("dotf 0.69.0")) { //nolint:gosec // test path
		t.Error("update replaced a dotf already at the pin")
	}
	if !strings.Contains(stdout, "no deploy needed") {
		t.Errorf("output = %q, want the current-checkout line", stdout)
	}
}

func TestPinnedDotfVersion(t *testing.T) {
	repo := t.TempDir()
	if _, err := pinnedDotfVersion(repo, ""); err == nil {
		t.Error("a checkout without versions.conf must report an error, not an empty pin")
	}
	if err := os.WriteFile(filepath.Join(repo, "versions.conf"), []byte("GO_VERSION=1.26.0\nDOTF_VERSION=0.70.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := pinnedDotfVersion(repo, ""); err != nil || got != "0.70.0" {
		t.Errorf("pinnedDotfVersion = %q, %v, want 0.70.0", got, err)
	}
}
