package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ManifestFile is the harness manifest, relative to the checkout root.
const ManifestFile = "harness/manifest.json"

// ErrCheckoutIsDeployDir is returned when the checkout and the deploy dir are
// the same directory: there is nothing to mirror, and copying a tree onto
// itself is not "unchanged", it is undefined.
var ErrCheckoutIsDeployDir = errors.New("the checkout is the deploy dir; nothing to mirror")

// ErrMissingTargets is returned (wrapped) when harness/manifest.json declares
// a target the checkout does not have. Everything else has already been
// mirrored by then — a broken declaration must not cost a machine the rest of
// its harness — but the gap is named, because skipping it silently is exactly
// how a drift check comes to evaluate a file the mirror never received
// (#1200) and print a remedy that cannot clear it.
var ErrMissingTargets = errors.New("harness/manifest.json declares a target the checkout does not have")

// MirrorResult is what one Mirror run did.
type MirrorResult struct {
	// Updated counts files written because their bytes differed or they were
	// absent; Unchanged counts files left untouched. Updated == 0 on a re-run
	// is the idempotence evidence a setup run reports (#1266).
	Updated, Unchanged int
	// Targets are the manifest-declared files mirrored beside harness/.
	Targets []string
	// Missing are the declared targets the checkout lacks; non-empty implies
	// the returned error wraps ErrMissingTargets.
	Missing []string
}

// Mirror copies the harness inputs the deploy-dir consumers read — the whole
// harness/ tree and every file harness/manifest.json declares as an injection
// target — from the checkout at repoRoot into deployDir, preserving relative
// paths. It replaces the bash+jq block setup-linux.sh carried and the block
// setup-windows.ps1 never had (WIN-007/#1288): `dotf doctor` reads
// model-map.json and model-pins.json from the deploy dir, so a Windows box
// failed both checks after every setup, with a remedy ("re-run setup") that
// could not clear them.
//
// Idempotent: a file whose bytes already match is left untouched, mtime
// included. It never prunes — `dotf doctor --fix` owns orphan removal, the
// semantic #802 settled for every mirror in this repository.
//
// The target list is DERIVED from the manifest, never restated here: the day
// it was a hardcoded pair, a third target (#1176) needed a copy line nobody
// wrote.
func Mirror(repoRoot, deployDir string) (MirrorResult, error) {
	var res MirrorResult
	repoRoot, deployDir = filepath.Clean(repoRoot), filepath.Clean(deployDir)
	if sameDir(repoRoot, deployDir) {
		return res, ErrCheckoutIsDeployDir
	}

	targets, err := manifestTargets(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil {
		return res, err
	}

	if err := mirrorTree(repoRoot, deployDir, "harness", &res); err != nil {
		return res, err
	}
	for _, rel := range targets {
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if !isRegular(src) {
			res.Missing = append(res.Missing, rel)
			continue
		}
		if err := mirrorFile(src, filepath.Join(deployDir, filepath.FromSlash(rel)), &res); err != nil {
			return res, err
		}
		res.Targets = append(res.Targets, rel)
	}
	if len(res.Missing) > 0 {
		return res, fmt.Errorf("%w: %v", ErrMissingTargets, res.Missing)
	}
	return res, nil
}

// manifestTargets reads `.targets[].file` — the same projection the bash block
// took with jq — in declaration order, de-duplicated.
func manifestTargets(path string) ([]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ManifestFile, err)
	}
	var m struct {
		Targets []struct {
			File string `json:"file"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ManifestFile, err)
	}
	seen := map[string]bool{}
	var out []string
	for _, t := range m.Targets {
		if t.File == "" || seen[t.File] {
			continue
		}
		seen[t.File] = true
		out = append(out, t.File)
	}
	return out, nil
}

// mirrorTree copies every regular file under <repoRoot>/<sub> to
// <deployDir>/<sub>, walking in a deterministic order.
func mirrorTree(repoRoot, deployDir, sub string, res *MirrorResult) error {
	root := filepath.Join(repoRoot, sub)
	if !isDir(root) {
		return fmt.Errorf("%s: not a directory in the checkout", filepath.ToSlash(sub))
	}
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking %s: %w", filepath.ToSlash(sub), err)
	}
	sort.Strings(files)
	for _, src := range files {
		rel, err := filepath.Rel(repoRoot, src)
		if err != nil {
			return err
		}
		if err := mirrorFile(src, filepath.Join(deployDir, rel), res); err != nil {
			return err
		}
	}
	return nil
}

// mirrorFile writes src's bytes to dst only when they differ, atomically
// (temp file in the destination dir, then rename), so a reader never sees a
// half-written registry and an identical file keeps its mtime.
func mirrorFile(src, dst string, res *MirrorResult) error {
	want, err := os.ReadFile(src) //nolint:gosec // paths derive from the checkout tree
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if have, err := os.ReadFile(dst); err == nil && bytes.Equal(have, want) { //nolint:gosec // same
		res.Unchanged++
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".mirror-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(want); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("installing %s: %w", dst, err)
	}
	res.Updated++
	return nil
}

func sameDir(a, b string) bool {
	if a == b {
		return true
	}
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fa, fb)
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isRegular(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
