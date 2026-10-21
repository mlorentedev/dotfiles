package harness

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/fsmode"
)

// Skill deploy statuses.
const (
	SkillDeployed  = "deployed"  // written, or would be under a dry run
	SkillCurrent   = "current"   // already held exactly the render
	SkillPruned    = "pruned"    // a generated output whose record or target is gone
	SkillSkipped   = "skipped"   // the target's requires_command is not on PATH
	SkillUnmanaged = "unmanaged" // an unmarked copy shadows a record: never pruned, reported
)

// SkillOutcome is what DeploySkills did for one output.
type SkillOutcome struct {
	Agent   string
	Path    string // absolute; for a skipped target, its directory
	Status  string
	Reason  string // why a target was skipped
	Catalog bool   // the copilot catalog region, not a skill output
}

// generatedMarker is the provenance every rendered output carries, and the only
// proof that an output is ours to prune.
const generatedMarker = "generated: true"

// DeploySkills renders every committed skill record to each skills.deploy[]
// target under home, then prunes the generated outputs whose record is gone or
// no longer targets that agent. An output that already holds its render is not
// rewritten, so a converged machine reports every output current. A symlink at
// an output is replaced by a copy (BUG-100). An output a record targets is
// overwritten with its render whether or not it carries the marker, as the
// shell did: that path belongs to the record. An unmarked copy of a record
// that does not target the agent is reported, never removed. A record that
// cannot be read is an error, reported after every other output is deployed.
// Last, the copilot catalog region is rewritten (skills_catalog.go). This is
// the Go form of compile-harness.sh's deploy_skills and deploy_prune.
func DeploySkills(repoRoot, home string, has func(string) bool, dryRun bool) ([]SkillOutcome, error) {
	m, err := LoadSkills(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil || m == nil {
		return nil, err
	}
	recDir := filepath.Join(repoRoot, filepath.FromSlash(m.RecordDir))
	names, err := recordNames(recDir)
	if err != nil {
		return nil, err
	}
	var out []SkillOutcome
	var errs []error
	for _, t := range m.Deploy {
		base := filepath.Join(home, filepath.FromSlash(t.Dir))
		if t.RequiresCommand != "" && !has(t.RequiresCommand) {
			out = append(out, SkillOutcome{Agent: t.Agent, Path: base, Status: SkillSkipped, Reason: t.RequiresCommand + " not on PATH"})
			continue
		}
		for _, name := range names {
			o, err := deploySkill(t, recDir, name, base, m.VaultSubpath+"/"+name+"/SKILL.md", dryRun)
			if err != nil {
				errs = append(errs, fmt.Errorf("skill %s for %s: %w", name, t.Agent, err))
				continue
			}
			if o != nil {
				out = append(out, *o)
			}
		}
	}
	for _, t := range m.Deploy {
		pruned, err := pruneSkills(t, recDir, filepath.Join(home, filepath.FromSlash(t.Dir)), dryRun)
		out = append(out, pruned...)
		if err != nil {
			errs = append(errs, fmt.Errorf("prune %s: %w", t.Agent, err))
		}
	}
	cat, err := deployCatalog(m, recDir, names, home, dryRun)
	if cat != nil {
		out = append(out, *cat)
	}
	if err != nil {
		errs = append(errs, fmt.Errorf("catalog: %w", err))
	}
	return out, errors.Join(errs...)
}

// recordNames lists the record directories holding a SKILL.md, in byte order.
// The shell's glob sorted by the locale's collation, which orders hyphenated
// names differently on glibc and on macOS.
func recordNames(recDir string) ([]string, error) {
	entries, err := os.ReadDir(recDir)
	if err != nil {
		return nil, fmt.Errorf("no skill records at %s: %w", recDir, err)
	}
	var names []string
	for _, e := range entries {
		if fi, err := os.Stat(filepath.Join(recDir, e.Name(), "SKILL.md")); err == nil && fi.Mode().IsRegular() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// skillOutPath is where target t puts the render of record name: one file for
// a command or prompt, a directory holding SKILL.md for a skill.
func skillOutPath(t SkillDeployTarget, base, name string) string {
	if t.Render == RenderSkill {
		return filepath.Join(base, name, "SKILL.md")
	}
	return filepath.Join(base, name+".md")
}

// deploySkill renders one record for one target; nil means the record does not
// target that agent.
func deploySkill(t SkillDeployTarget, recDir, name, base, srcPath string, dryRun bool) (*SkillOutcome, error) {
	record, err := os.ReadFile(filepath.Join(recDir, name, "SKILL.md")) //nolint:gosec // a record under the manifest's record_dir
	if err != nil {
		return nil, err
	}
	targeted, err := SkillTargetsAgent(record, t.Agent)
	if err != nil || !targeted {
		return nil, err
	}
	outp := skillOutPath(t, base, name)
	changed := false
	if t.Render == RenderSkill {
		// The skill's directory is the unit: a symlinked one is unlinked, and
		// the record's auxiliary files are copied in beside the render.
		if changed, err = unlinkSymlink(filepath.Dir(outp), dryRun); err != nil {
			return nil, err
		}
		aux, err := syncAuxFiles(filepath.Join(recDir, name), filepath.Dir(outp), dryRun)
		if err != nil {
			return nil, err
		}
		changed = changed || aux
	}
	wrote, err := writeIfChanged(outp, []byte(RenderSkillRecord(t.Render, record, srcPath)), 0o644, dryRun)
	if err != nil {
		return nil, err
	}
	status := SkillCurrent
	if changed || wrote {
		status = SkillDeployed
	}
	return &SkillOutcome{Agent: t.Agent, Path: outp, Status: status}, nil
}

// unlinkSymlink removes a symlink at p, the shape the vault-symlink era left
// behind, so the deploy writes a regular copy instead of through the link.
func unlinkSymlink(p string, dryRun bool) (bool, error) {
	fi, err := os.Lstat(p)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	return true, os.Remove(p)
}

// syncAuxFiles copies a record's files other than SKILL.md into dst, keeping
// each file's mode, so a helper script stays executable. Top-level dotfiles are
// left out, as the shell's `cp -rf "$dir"*` left them out.
func syncAuxFiles(src, dst string, dryRun bool) (bool, error) {
	changed := false
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, ".") && !strings.ContainsRune(rel, filepath.Separator) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || rel == "SKILL.md" {
			return nil
		}
		fi, err := os.Stat(p)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(p) //nolint:gosec // a file under a committed record
		if err != nil {
			return err
		}
		wrote, err := writeIfChanged(filepath.Join(dst, rel), content, fi.Mode().Perm(), dryRun)
		changed = changed || wrote
		return err
	})
	return changed, err
}

// writeIfChanged writes content with mode perm unless p already holds exactly
// that, and reports whether it wrote. A symlink at p is replaced, not followed.
// The mode is compared as far as the OS expresses it (fsmode.Needs): Windows
// reports 0666 for any writable file, so a full comparison would rewrite every
// output on every run there.
func writeIfChanged(p string, content []byte, perm fs.FileMode, dryRun bool) (bool, error) {
	if fi, err := os.Lstat(p); err == nil && fi.Mode().IsRegular() && !modeNeeded(p, perm) {
		if current, err := os.ReadFile(p); err == nil && bytes.Equal(current, content) { //nolint:gosec // an output under home
			return false, nil
		}
	}
	if dryRun {
		return true, nil
	}
	if _, err := unlinkSymlink(p, false); err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(p, content, perm); err != nil {
		return false, err
	}
	return true, os.Chmod(p, perm)
}

// modeNeeded reports whether p's mode differs from perm; an unreadable mode
// counts as differing, so the write goes ahead.
func modeNeeded(p string, perm fs.FileMode) bool {
	needs, err := fsmode.Needs(p, perm)
	return err != nil || needs
}

// pruneSkills removes target t's generated outputs whose record is gone or no
// longer targets t.Agent. It runs for a target whose requires_command is
// absent too, so an uninstalled tool keeps no stale copy.
func pruneSkills(t SkillDeployTarget, recDir, base string, dryRun bool) ([]SkillOutcome, error) {
	entries, err := os.ReadDir(base)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []SkillOutcome
	var errs []error
	for _, e := range entries {
		name, outp, isOutput := pruneCandidate(t, base, e)
		if !isOutput {
			continue
		}
		record, recErr := os.ReadFile(filepath.Join(recDir, name, "SKILL.md")) //nolint:gosec // a record under the manifest's record_dir
		deployed, _ := os.ReadFile(skillOutPath(t, base, name))                //nolint:gosec // an output under home
		if !bytes.Contains(deployed, []byte(generatedMarker)) {
			if recErr == nil {
				out = append(out, SkillOutcome{Agent: t.Agent, Path: outp, Status: SkillUnmanaged})
			}
			continue
		}
		if recErr == nil {
			targeted, err := SkillTargetsAgent(record, t.Agent)
			if err != nil {
				errs = append(errs, fmt.Errorf("skill %s: %w", name, err))
			}
			if err != nil || targeted {
				continue
			}
		}
		if !dryRun {
			if err := os.RemoveAll(outp); err != nil {
				errs = append(errs, err)
				continue
			}
		}
		out = append(out, SkillOutcome{Agent: t.Agent, Path: outp, Status: SkillPruned})
	}
	return out, errors.Join(errs...)
}

// pruneCandidate names the record a directory entry would be the output of: a
// directory for a skill target, a .md file for a command or prompt one. Hidden
// entries are no record's output.
func pruneCandidate(t SkillDeployTarget, base string, e fs.DirEntry) (name, outp string, ok bool) {
	if strings.HasPrefix(e.Name(), ".") {
		return "", "", false
	}
	outp = filepath.Join(base, e.Name())
	fi, err := os.Stat(outp)
	if err != nil {
		return "", "", false
	}
	if t.Render == RenderSkill {
		return e.Name(), outp, fi.IsDir()
	}
	if !fi.Mode().IsRegular() || !strings.HasSuffix(e.Name(), ".md") {
		return "", "", false
	}
	return strings.TrimSuffix(e.Name(), ".md"), outp, true
}
