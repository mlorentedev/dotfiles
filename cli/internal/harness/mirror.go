package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	// Updated counts files written because their bytes or permission bits
	// differed, or they were absent; Unchanged counts converged files left
	// untouched. Updated == 0 on a re-run is the idempotence evidence a setup
	// run reports (#1266).
	Updated, Unchanged int
	// Targets are the manifest-declared files mirrored beside harness/.
	Targets []string
	// Missing are the declared targets the checkout lacks; non-empty implies
	// the returned error wraps ErrMissingTargets.
	Missing []string
	// Pruned are the leftovers removed from the pruned deploy-dir trees (or,
	// in a plan, that would be); Unpruned are orphans git never tracked, left
	// in place; PruneSkipped says why no orphan could be proven a leftover.
	// See ScanOrphans.
	Pruned, Unpruned []string
	PruneSkipped     string
	// Unreadable are deploy-dir entries the leftover scan could not read, so
	// they were not checked (Orphans.Unreadable).
	Unreadable []string
}

// Mirror copies the inputs the deploy-dir consumers read — the whole harness/
// tree, every file harness/manifest.json declares as an injection target, and
// the deploy-dir set (DeployDirFiles, DeployDirTrees) — from the checkout at
// repoRoot into deployDir, preserving relative paths. It replaces the bash+jq block setup-linux.sh carried and the block
// setup-windows.ps1 never had (WIN-007/#1288): `dotf doctor` reads
// model-map.json and model-pins.json from the deploy dir, so a Windows box
// failed both checks after every setup, with a remedy ("re-run setup") that
// could not clear them.
//
// Idempotent: a file whose bytes and permission bits already match is left
// untouched, mtime included. It prunes only the trees that hold nothing but
// checkout copies (PrunedDeployDirTrees), and there only a file the checkout's
// history deleted: #802's arm for generated copies. harness/ and the secrets
// stay with `dotf doctor --fix`, the semantic #802 settled for them.
//
// The target list is DERIVED from the manifest, never restated here: the day
// it was a hardcoded pair, a third target (#1176) needed a copy line nobody
// wrote.
func Mirror(repoRoot, deployDir string) (MirrorResult, error) {
	return mirror(repoRoot, deployDir, false)
}

// PlanMirror reports what Mirror would do without writing anything, the deploy
// dir included: Updated counts the files Mirror would write. It runs Mirror's
// own walk and comparison, so the plan and the apply cannot disagree.
func PlanMirror(repoRoot, deployDir string) (MirrorResult, error) {
	return mirror(repoRoot, deployDir, true)
}

func mirror(repoRoot, deployDir string, dryRun bool) (MirrorResult, error) {
	var res MirrorResult
	repoRoot, deployDir = filepath.Clean(repoRoot), filepath.Clean(deployDir)
	if sameDir(repoRoot, deployDir) {
		return res, ErrCheckoutIsDeployDir
	}

	targets, err := manifestTargets(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil {
		return res, err
	}

	ignored := IgnoredInCheckout(repoRoot, ExecGit)
	if err := mirrorTree(repoRoot, deployDir, "harness", ignored, dryRun, &res); err != nil {
		return res, err
	}
	for _, rel := range targets {
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if !isRegular(src) {
			res.Missing = append(res.Missing, rel)
			continue
		}
		if err := mirrorFile(src, filepath.Join(deployDir, filepath.FromSlash(rel)), dryRun, &res); err != nil {
			return res, err
		}
		res.Targets = append(res.Targets, rel)
	}
	if err := mirrorDeployDir(repoRoot, deployDir, ignored, dryRun, &res); err != nil {
		return res, err
	}
	orphans, err := ScanOrphans(repoRoot, deployDir, ExecGit)
	if err != nil {
		return res, err
	}
	res.Pruned, res.Unpruned, res.PruneSkipped, res.Unreadable = orphans.Deleted, orphans.Unknown, orphans.Skipped, orphans.Unreadable
	if !dryRun {
		if err := PruneOrphans(deployDir, orphans.Deleted); err != nil {
			return res, err
		}
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
		// A target is mirrored to the same path under the deploy dir, so one
		// that is absolute or climbs out with ".." would write outside it.
		if !filepath.IsLocal(filepath.FromSlash(t.File)) {
			return nil, fmt.Errorf("%s: target %q is not a path inside the checkout", ManifestFile, t.File)
		}
		seen[t.File] = true
		out = append(out, t.File)
	}
	return out, nil
}

// mirrorTree copies every regular file under <repoRoot>/<sub> to
// <deployDir>/<sub>, walking in a deterministic order.
func mirrorTree(repoRoot, deployDir, sub string, ignored map[string]bool, dryRun bool, res *MirrorResult) error {
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
		if ignored[filepath.ToSlash(rel)] {
			continue
		}
		if err := mirrorFile(src, filepath.Join(deployDir, rel), dryRun, res); err != nil {
			return err
		}
	}
	return nil
}

// mirrorFile writes src to dst only when bytes or permission bits differ,
// atomically (temp file in the destination dir, then rename), so a reader never
// sees a half-written registry and a converged file keeps its mtime. Under
// dryRun it stops after the comparison and only counts.
func mirrorFile(src, dst string, dryRun bool, res *MirrorResult) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stating %s: %w", src, err)
	}
	want, err := os.ReadFile(src) //nolint:gosec // paths derive from the checkout tree
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if have, err := os.ReadFile(dst); err == nil && bytes.Equal(have, want) { //nolint:gosec // same
		dstInfo, statErr := os.Stat(dst)
		if statErr == nil && dstInfo.Mode().Perm() == info.Mode().Perm() {
			res.Unchanged++
			return nil
		}
	}
	if dryRun {
		res.Updated++
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmpName, err := writeMirrorTemp(filepath.Dir(dst), want, info.Mode().Perm())
	if err != nil {
		return err
	}
	if err := replaceMirrorDestination(tmpName, dst); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	res.Updated++
	return nil
}

func writeMirrorTemp(dir string, content []byte, mode os.FileMode) (string, error) {
	tmp, err := os.CreateTemp(dir, ".mirror-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	return tmpName, nil
}

func replaceMirrorDestination(tmpName, dst string) error {
	restoreMode, restore, err := makeDestinationReplaceable(dst)
	if err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		if restore {
			_ = os.Chmod(dst, restoreMode)
		}
		return fmt.Errorf("installing %s: %w", dst, err)
	}
	return nil
}

func makeDestinationReplaceable(path string) (os.FileMode, bool, error) {
	if runtime.GOOS != "windows" {
		return 0, false, nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("stating destination %s: %w", path, err)
	}
	mode := info.Mode().Perm()
	if mode&0o200 != 0 {
		return mode, false, nil
	}
	if err := os.Chmod(path, mode|0o200); err != nil {
		return mode, false, fmt.Errorf("making destination replaceable %s: %w", path, err)
	}
	return mode, true, nil
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
