package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Medium is an install image the build trusts by its SHA256 alone.
type Medium struct {
	Name   string
	URL    string // where to download it; empty when only the owner can (WindowsISO)
	SHA256 string
}

// WindowsISO is Microsoft's multi-edition Windows 11 Arm64 ISO, version 26H2
// (build 26300), in English (United States), the language of the owner's
// Windows machines. Microsoft's download API refuses automation (lesson 481),
// so the owner supplies the file or the 24-hour link.
//
// The hash is the file's, measured 2026-10-18 on the ISO the page served for
// "English": volume label CCCOMA_A64FRE_EN-US_DV9 (consumer, Arm64, en-US).
// It is in the SHA256 table microsoft.com/software-download/windows11arm64
// publishes, but under "Arabic 64-bit": that table's language labels do not
// match its files, so a pin is checked against the file's own label, never
// read from the row. A new Windows release changes it.
var WindowsISO = Medium{
	Name:   "Windows 11 Arm64 26H2 multi-edition ISO (English, United States)",
	SHA256: "8992f2d2cfd2fa647e579e9eb31fc21f34da8333c29b6170fde17d2ed6623e9c",
}

// VirtioWin carries the ARM64 network and serial drivers and the guest agent.
var VirtioWin = Medium{
	Name:   "virtio-win 0.1.302",
	URL:    "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.302-1/virtio-win-0.1.302.iso",
	SHA256: "303f7ae40dad495d6ae474fdc571df58958a4dbc5c37a522d80f9a203867949d",
}

// Ensure leaves at dest a file that hashes to m.SHA256. A matching file
// already there is kept; otherwise src is copied (a local path) or downloaded
// (an http(s) URL). The copy is hashed as it is written and lands at dest only
// when it matches, so dest never holds an unverified file.
func Ensure(client *http.Client, dest string, m Medium, src string, out io.Writer) error {
	want := strings.ToLower(m.SHA256)
	if got, err := hashFile(dest); err == nil {
		if got == want {
			step(out, "%s: verified at %s", m.Name, dest)
			return nil
		}
		if src == "" {
			return fmt.Errorf("%s at %s has SHA256 %s, want %s: replace it, or pass its source", m.Name, dest, got, want)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else if src == "" {
		return fmt.Errorf("%s is missing: put it at %s, or pass its file or download link", m.Name, dest)
	}

	r, from, err := open(client, src)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".*.partial")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	step(out, "%s: fetching from %s", m.Name, from)
	h := sha256.New()
	_, cerr := io.Copy(io.MultiWriter(tmp, h), r)
	if err := errors.Join(cerr, tmp.Close()); err != nil {
		return fmt.Errorf("%s from %s: %w", m.Name, from, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("%s from %s has SHA256 %s, want %s: wrong file, language or release", m.Name, from, got, want)
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return err
	}
	step(out, "%s: verified at %s", m.Name, dest)
	return nil
}

// open reads src, a local path or an http(s) URL, and names it for progress
// without a URL's query: a signed download link carries its signature there.
func open(client *http.Client, src string) (io.ReadCloser, string, error) {
	u, err := url.Parse(src)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		f, err := os.Open(src)
		return f, src, err
	}
	from := u.Scheme + "://" + u.Host + u.Path
	resp, err := client.Get(src)
	if err != nil {
		// A *url.Error repeats the full URL; keep only its cause.
		if ue := (*url.Error)(nil); errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, from, fmt.Errorf("download from %s: %w", from, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, from, fmt.Errorf("download from %s: %s", from, resp.Status)
	}
	return resp.Body, from, nil
}

func hashFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
