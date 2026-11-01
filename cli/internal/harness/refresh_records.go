package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// refreshWrites is what a refresh will change, collected before any of it is
// applied so a failed check leaves the checkout untouched.
type refreshWrites struct {
	ops []refreshOp
}

type refreshOp struct {
	log string
	// One of: a whole file (dst, content), a record dir copied from the vault
	// (dst, src, stamp, from, sha), or a stale record to drop (dst, drop).
	dst, content          string
	src, stamp, from, sha string
	drop                  bool
}

func (w *refreshWrites) file(dst, content, log string) {
	w.ops = append(w.ops, refreshOp{log: log, dst: dst, content: content})
}

// records plans the record dirs of one kind: each vault dir holding file is
// validated against the schema and replaces its record whole, and a record
// whose vault dir is gone is dropped. A vault dir without the file is not a
// record and leaves its old record in place, as the shell did.
func (w *refreshWrites) records(repoRoot, vault string, k *recordKind, file string) error {
	vaultDir := filepath.Join(vault, filepath.FromSlash(k.VaultSubpath))
	recDir := filepath.Join(repoRoot, filepath.FromSlash(k.RecordDir))
	if !isDir(vaultDir) {
		return fmt.Errorf("refresh needs the vault dir %s", vaultDir)
	}
	schema := orDefault(k.Schema, "harness/"+strings.ToLower(strings.TrimSuffix(file, ".md"))+"-frontmatter.schema.json")
	required, err := schemaRequired(filepath.Join(repoRoot, filepath.FromSlash(schema)))
	if err != nil {
		return fmt.Errorf("schema %s: %w", schema, err)
	}
	names, err := visibleDirs(vaultDir)
	if err != nil {
		return err
	}
	for _, name := range names {
		src := filepath.Join(vaultDir, name)
		if _, err := os.Stat(filepath.Join(src, file)); err != nil {
			continue
		}
		if msg := frontmatterError(filepath.Join(src, file), required); msg != "" {
			return errors.New(msg)
		}
		sha, err := fileSHA16(filepath.Join(src, file))
		if err != nil {
			return err
		}
		w.ops = append(w.ops, refreshOp{
			log: fmt.Sprintf("record: %s/%s", k.RecordDir, name),
			dst: filepath.Join(recDir, name), src: src, stamp: file, from: path.Join(k.VaultSubpath, name, file), sha: sha,
		})
	}
	stale, err := visibleDirs(recDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, name := range stale {
		if !isDir(filepath.Join(vaultDir, name)) {
			w.ops = append(w.ops, refreshOp{log: "dropped stale record: " + path.Join(k.RecordDir, name),
				dst: filepath.Join(recDir, name), drop: true})
		}
	}
	return nil
}

// apply performs the planned writes in order and returns their log lines.
func (w *refreshWrites) apply() ([]string, error) {
	var log []string
	for _, op := range w.ops {
		var err error
		switch {
		case op.drop:
			err = os.RemoveAll(op.dst)
		case op.src != "":
			err = replaceRecordDir(op)
		default:
			err = writeKeepingMode(op.dst, []byte(op.content))
		}
		if err != nil {
			return log, err
		}
		log = append(log, op.log)
	}
	return log, nil
}

// replaceRecordDir replaces a record dir with a copy of its vault dir, then
// stamps the record file with where it came from and the sha of the source it
// was copied from (HARNESS-069). The sha is of the vault source, not of the
// stamped record: it says "this is what I was refreshed from".
func replaceRecordDir(op refreshOp) error {
	if err := os.RemoveAll(op.dst); err != nil {
		return err
	}
	entries, err := os.ReadDir(op.src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(op.dst, 0o755); err != nil {
		return err
	}
	// The shell copied "$dir"*, a glob that skips the dir's own dotfiles.
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if err := copyEntry(filepath.Join(op.src, e.Name()), filepath.Join(op.dst, e.Name())); err != nil {
			return err
		}
	}
	rec := filepath.Join(op.dst, op.stamp)
	raw, err := os.ReadFile(rec) //nolint:gosec // the record just copied
	if err != nil {
		return err
	}
	return writeKeepingMode(rec, stampProvenance(raw, op.from, op.sha))
}

// stampProvenance inserts the generated_* fields after the opening fence. Like
// the awk it replaces, it reads lines, so the output ends in a newline.
func stampProvenance(raw []byte, from, sha string) []byte {
	var b strings.Builder
	done := false
	for _, l := range recordLines(raw) {
		b.WriteString(l + "\n")
		if !done && fenceLine.MatchString(l) {
			fmt.Fprintf(&b, "generated: true\ngenerated_from: %s\ngenerated_sha: %s\n", from, sha)
			done = true
		}
	}
	return []byte(b.String())
}

// copyEntry copies a file, a symlink or a directory tree, keeping file modes:
// a skill's helper script stays executable.
func copyEntry(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case fi.IsDir():
		if err := os.MkdirAll(dst, fi.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyEntry(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	b, err := os.ReadFile(src) //nolint:gosec // a file under a vault record dir
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, fi.Mode().Perm())
}

// writeKeepingMode writes data to p, keeping the mode of the file it replaces.
func writeKeepingMode(p string, data []byte) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(p); err == nil {
		mode = fi.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, mode)
}

// visibleDirs lists the subdirectories of dir that a shell `*/` glob matches:
// not hidden, in byte order.
func visibleDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !isDir(filepath.Join(dir, e.Name())) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}
