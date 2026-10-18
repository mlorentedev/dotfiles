package doctor

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// checkSkillRecordsAgainstVault compares each committed harness record with the
// vault source it was refreshed from. The vault is the source of record for
// skills and agents; `compile-harness.sh --refresh` copies it into the
// committed harness/{skills,agents}/<name>/ records and stamps `generated_from`
// (the vault path) and `generated_sha` (the first 16 hex of the source's
// sha256). `--check` validates the records offline and never reads the vault,
// so a vault edit nobody refreshed left main's record stale with nothing
// saying so: on 2026-10-13 msi's checkout carried the refreshed teardown skill
// as an "uncommitted change" while main was the stale side.
//
// WARN, not FAIL: either side can be the older one. A vault edit waits for a
// refresh PR, and a machine whose vault is not pulled yet sees the records
// ahead of it.
func checkSkillRecordsAgainstVault(sys *System, rep *Report) {
	repo := resolveRepoDir(sys)
	if repo == "" {
		return
	}
	// The resolver, not a literal default: a vault guessed at ~/Projects/knowledge
	// would compare against whatever happens to live there (ADR-025).
	vault := sys.vaultPath()
	if vault == "" || !isDir(vault) {
		rep.Skip("no vault resolves on this machine (VAULT_PATH, machine.json): committed harness records not compared with their source")
		return
	}

	var stale, unstamped, unread []string
	checked := 0
	for _, kind := range []struct{ sub, file string }{{"skills", "SKILL.md"}, {"agents", "AGENT.md"}} {
		records, err := recordFiles(filepath.Join(repo, "harness", kind.sub), kind.file)
		if err != nil {
			unread = append(unread, fmt.Sprintf("harness/%s (%v)", kind.sub, err))
		}
		for _, rec := range records {
			name := kind.sub + "/" + filepath.Base(filepath.Dir(rec))
			from, sha := recordProvenance(rec)
			if from == "" || sha == "" {
				// Every record --refresh writes carries both; one without them
				// cannot be compared, and skipping it would hide its drift.
				unstamped = append(unstamped, name)
				continue
			}
			checked++
			got, err := fileSHA16(filepath.Join(vault, filepath.FromSlash(from)))
			switch {
			case err != nil:
				stale = append(stale, name+" (source gone)")
			case got != sha:
				stale = append(stale, name)
			}
		}
	}

	if len(unread) > 0 {
		rep.Warn(fmt.Sprintf("cannot read %s, so those records were not compared with the vault", strings.Join(unread, ", ")))
	}
	if len(unstamped) > 0 {
		sort.Strings(unstamped)
		rep.Warn(fmt.Sprintf("%d committed harness record(s) carry no generated_from/generated_sha, so they cannot be compared with the vault: %s "+
			"(run `compile-harness.sh --refresh` in a worktree and open a PR)", len(unstamped), strings.Join(unstamped, ", ")))
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		rep.Warn(fmt.Sprintf("%d committed harness record(s) differ from their vault source: %s. "+
			"If the vault is the newer side, run `compile-harness.sh --refresh` in a worktree and open a PR; "+
			"if the records are, pull the vault", len(stale), strings.Join(stale, ", ")))
		return
	}
	if checked > 0 && len(unstamped) == 0 && len(unread) == 0 {
		rep.Pass(fmt.Sprintf("%d committed records match their vault source", checked))
	}
}

// recordFiles is filepath.Glob(dir/*/file) without its blind spot: Glob drops
// I/O errors, so an unreadable records directory read as holding none and the
// check passed over it (#2797). A dir that does not exist holds no records;
// any other failure to read it, or to check one of its entries, is returned.
func recordFiles(dir, file string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var found []string
	var errs []error
	for _, e := range entries {
		p := filepath.Join(dir, e.Name(), file)
		switch _, err := os.Lstat(p); {
		case err == nil:
			found = append(found, p)
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
		default:
			errs = append(errs, err)
		}
	}
	return found, errors.Join(errs...)
}

// recordProvenance reads generated_from and generated_sha from a record's
// frontmatter.
func recordProvenance(path string) (from, sha string) {
	f, err := os.Open(path)
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

// fileSHA16 is compile-harness.sh's sha_of: the first 16 hex of the sha256.
func fileSHA16(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:16], nil
}
