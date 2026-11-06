package harness

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/fsread"
)

// VaultDrift is how the committed harness records compare with the vault
// sources they were refreshed from. Each record carries `generated_from` (the
// vault path) and `generated_sha` (the first 16 hex of the source's sha256), so
// a vault edit nobody refreshed shows as a sha that no longer matches (#1545).
// The stamp sees only the vault side, so each record whose stamp matches is also
// compared with the render of its source: a record edited without its source
// is reverted by the next refresh (#1545), and Edited names it.
type VaultDrift struct {
	Checked   int
	Stale     []string // kind/name, with " (source gone)" when the source is missing
	Edited    []string // kind/name of a record that differs from the render of its unchanged source
	Unstamped []string // records carrying no provenance, which cannot be compared
	Unread    []string // records directories that could not be read
	Missing   []string // kind/name of a vault record a refresh would write that the checkout lacks
	Excluded  []string // kind/name of a record the manifest excludes that is still committed
	Unused    []string // kind/name of a manifest exclusion the vault has no dir for
	Invalid   []string // a vault record or exclusion a refresh refuses, with the reason
}

// HasDrift reports whether any record failed to match or could not be compared.
func (d VaultDrift) HasDrift() bool {
	return len(d.Stale)+len(d.Edited)+len(d.Unstamped)+len(d.Unread)+len(d.Missing)+len(d.Excluded)+len(d.Unused)+len(d.Invalid) > 0
}

// Lines renders the comparison as check lines, problems first.
func (d VaultDrift) Lines() []CheckLine {
	var out []CheckLine
	for _, n := range d.Missing {
		out = append(out, CheckLine{Problem: true, Text: "[MISSING] harness/" + n + " is a vault record no refresh has committed",
			Detail: []string{"      -> run dotf harness refresh in a worktree and commit it, or exclude it in the manifest with the reason"}})
	}
	for _, n := range d.Excluded {
		out = append(out, CheckLine{Problem: true, Text: "[EXCLUDED] harness/" + n + " is excluded in the manifest but still committed",
			Detail: []string{"      -> run dotf harness refresh in a worktree, which drops it, and commit the removal"}})
	}
	for _, n := range d.Unused {
		out = append(out, CheckLine{Problem: true, Text: "[UNUSED] the manifest excludes " + n + ", which the vault no longer has",
			Detail: []string{"      -> remove the exclusion: its reason no longer applies to anything"}})
	}
	for _, msg := range d.Invalid {
		out = append(out, CheckLine{Problem: true, Text: "[INVALID] " + msg,
			Detail: []string{"      -> fix it at the source; a refresh refuses it, so committing a record cannot clear this"}})
	}
	for _, n := range d.Stale {
		out = append(out, CheckLine{Problem: true, Text: "[STALE] harness/" + n + " differs from its vault source",
			Detail: []string{"      -> if the vault is the newer side, run dotf harness refresh in a worktree; if the record is, carry the edit to the vault"}})
	}
	for _, n := range d.Edited {
		out = append(out, CheckLine{Problem: true, Text: "[EDITED] harness/" + n + " differs from the render of its vault source, which has not changed",
			Detail: []string{"      -> carry the edit to the vault source, then run dotf harness refresh; a refresh reverts an edit made only to the record"}})
	}
	for _, n := range d.Unstamped {
		out = append(out, CheckLine{Problem: true, Text: "[UNSTAMPED] harness/" + n + " carries no generated_from/generated_sha, so it cannot be compared with the vault"})
	}
	for _, u := range d.Unread {
		out = append(out, CheckLine{Problem: true, Text: "[ERROR] cannot read " + u + ", so its records were not compared with the vault"})
	}
	if !d.HasDrift() {
		out = append(out, CheckLine{Text: fmt.Sprintf("[check] OK -> %d committed records match their vault source", d.Checked)})
	}
	return out
}

// CompareWithVault compares every skill and agent record under repoRoot with
// its source under vault.
func CompareWithVault(repoRoot, vault string) VaultDrift {
	var d VaultDrift
	for _, kind := range []struct{ sub, file string }{{"skills", "SKILL.md"}, {"agents", "AGENT.md"}} {
		records, err := recordFiles(filepath.Join(repoRoot, "harness", kind.sub), kind.file)
		if err != nil {
			d.Unread = append(d.Unread, fmt.Sprintf("harness/%s (%v)", kind.sub, err))
		}
		for _, rec := range records {
			name := kind.sub + "/" + filepath.Base(filepath.Dir(rec))
			from, sha := recordProvenance(rec)
			if from == "" || sha == "" {
				// Every record a refresh writes carries both; one without them
				// cannot be compared, and skipping it would hide its drift.
				d.Unstamped = append(d.Unstamped, name)
				continue
			}
			d.Checked++
			got, err := fileSHA16(filepath.Join(vault, filepath.FromSlash(from)))
			switch {
			case err != nil:
				d.Stale = append(d.Stale, name+" (source gone)")
			case got != sha:
				d.Stale = append(d.Stale, name)
			default:
				d.compareRender(name, filepath.Dir(rec), filepath.Join(vault, filepath.FromSlash(from)), from, sha)
			}
		}
	}
	d.compareEnforced(repoRoot, vault)
	d.compareCoverage(repoRoot, vault)
	sort.Strings(d.Missing)
	sort.Strings(d.Stale)
	sort.Strings(d.Edited)
	sort.Strings(d.Unstamped)
	return d
}

// compareCoverage walks the vault side, which the stamps cannot see (#3098).
// A vault record that a refresh would write and the checkout lacks is Missing,
// and one a refresh would refuse is Invalid, since committing cannot clear it.
// A committed record the manifest excludes is Excluded, since a refresh drops
// it, and an exclusion naming no vault dir is Unused. A checkout without a
// manifest has no record kinds to walk.
func (d *VaultDrift) compareCoverage(repoRoot, vault string) {
	m, err := loadCheckManifest(repoRoot)
	if err != nil {
		return // compareEnforced already reported an unreadable manifest
	}
	for _, kind := range []struct {
		k         *recordKind
		sub, file string
	}{{m.Skills, "skills", "SKILL.md"}, {m.Agents, "agents", "AGENT.md"}} {
		if kind.k != nil {
			d.coverKind(repoRoot, vault, kind.k, kind.sub, kind.file)
		}
	}
	sort.Strings(d.Unused)
}

func (d *VaultDrift) coverKind(repoRoot, vault string, k *recordKind, sub, file string) {
	vaultDir := filepath.Join(vault, filepath.FromSlash(k.VaultSubpath))
	recDir := filepath.Join(repoRoot, filepath.FromSlash(k.RecordDir))
	names, err := visibleDirs(vaultDir)
	if err != nil {
		d.Unread = append(d.Unread, fmt.Sprintf("vault %s (%v)", k.VaultSubpath, err))
		return
	}
	schema := k.schema(file)
	required, err := schemaRequired(filepath.Join(repoRoot, filepath.FromSlash(schema)))
	if err != nil {
		d.Unread = append(d.Unread, fmt.Sprintf("%s (%v)", schema, err))
		return
	}
	for name := range k.Exclude {
		if !isDir(filepath.Join(vaultDir, name)) {
			d.Unused = append(d.Unused, sub+"/"+name)
		}
	}
	for _, name := range names {
		src := filepath.Join(vaultDir, name, file)
		_, notRecord := os.Stat(src)
		_, notCommitted := os.Stat(filepath.Join(recDir, name, file))
		excluded, err := k.excluded(name)
		switch {
		case err != nil:
			d.Invalid = append(d.Invalid, err.Error())
		case excluded:
			if notCommitted == nil {
				d.Excluded = append(d.Excluded, sub+"/"+name)
			}
		case notRecord != nil:
		case frontmatterError(src, required) != "":
			d.Invalid = append(d.Invalid, frontmatterError(src, required))
		case notCommitted != nil:
			d.Missing = append(d.Missing, sub+"/"+name)
		}
	}
}

// recordFiles is filepath.Glob(dir/*/file) without its blind spot: Glob drops
// I/O errors, so an unreadable records directory read as holding none and the
// check passed over it (#2797). A dir that does not exist holds no records;
// any other failure to read it, or to check one of its entries, is returned.
func recordFiles(dir, file string) ([]string, error) {
	found, unreadable, err := fsread.ChildFiles(dir, file)
	if err != nil {
		return nil, err
	}
	errs := make([]error, len(unreadable))
	for i, pe := range unreadable {
		errs[i] = pe
	}
	return found, errors.Join(errs...)
}

// recordProvenance reads generated_from and generated_sha from a record's
// frontmatter.
func recordProvenance(path string) (from, sha string) {
	f, err := os.Open(path) //nolint:gosec // a record path under the checkout
	if err != nil {
		return "", ""
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	fences := 0
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			fences++
			if fences == 2 {
				break
			}
			continue
		}
		if fences != 1 {
			continue
		}
		if v, ok := strings.CutPrefix(line, "generated_from:"); ok {
			from = strings.TrimSpace(v)
		} else if v, ok := strings.CutPrefix(line, "generated_sha:"); ok {
			sha = strings.TrimSpace(v)
		}
	}
	return from, sha
}

// fileSHA16 is the refresh's sha of a source file: the first 16 hex of its
// sha256, over the bytes as they are.
func fileSHA16(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // a vault path named by a committed record
	if err != nil {
		return "", err
	}
	return sha16(string(b)), nil
}

func sha16(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}
