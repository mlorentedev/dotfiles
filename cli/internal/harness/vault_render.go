package harness

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// compareRender holds a record dir to what a refresh would write from its
// source dir: every file the same, and the record file stamped with the
// provenance it already carries. Line endings are compared as LF, so a vault
// checked out with CRLF does not read as an edit.
func (d *VaultDrift) compareRender(name, recDir, srcFile, from, sha string) {
	srcDir := filepath.Dir(srcFile)
	want, err := treeFiles(srcDir)
	if err == nil {
		file := filepath.Base(srcFile)
		want[file] = stampProvenance(normalizeLF(want[file]), from, sha)
	}
	var got map[string][]byte
	if err == nil {
		got, err = treeFiles(recDir)
	}
	if err != nil {
		d.Unread = append(d.Unread, fmt.Sprintf("harness/%s (%v)", name, err))
		return
	}
	if !sameTree(got, want) {
		d.Edited = append(d.Edited, name)
	}
}

// compareEnforced holds each enforced record to the section of the vault
// pattern it is cut from. These records carry no stamp, so this is the only
// comparison they get. A checkout without a manifest has none to compare.
func (d *VaultDrift) compareEnforced(repoRoot, vault string) {
	m, err := loadCheckManifest(repoRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		d.Unread = append(d.Unread, fmt.Sprintf("%s (%v)", ManifestFile, err))
		return
	}
	for _, e := range m.Enforced {
		name := "enforced/" + e.ID
		pat, slug, _ := strings.Cut(e.Source, "#")
		d.Checked++
		want, err := sectionOf(filepath.Join(vault, filepath.FromSlash(path.Join(m.VaultSubpath, pat))), slug)
		if err != nil {
			d.Stale = append(d.Stale, name+" (source gone)")
			continue
		}
		got, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(EnforcedDir), e.ID+".md")) //nolint:gosec // a record the manifest names
		if err != nil {
			d.Unread = append(d.Unread, fmt.Sprintf("harness/%s.md (%v)", name, err))
			continue
		}
		if string(normalizeLF(got)) != want {
			d.Edited = append(d.Edited, name)
		}
	}
}

// treeFiles maps each file under dir, by slash path, to its LF content. The
// dir's own dotfiles are skipped on both sides: a refresh never copies them
// (the shell's "$dir"* glob skipped them), and a .DS_Store Finder leaves in a
// record dir is not an edit.
func treeFiles(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if rel != "." && !strings.Contains(rel, "/") && strings.HasPrefix(rel, ".") {
			if e.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if e.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p) //nolint:gosec // a file under a record or vault dir
		if err != nil {
			return err
		}
		files[rel] = normalizeLF(b)
		return nil
	})
	return files, err
}

func sameTree(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || !bytes.Equal(v, w) {
			return false
		}
	}
	return true
}
