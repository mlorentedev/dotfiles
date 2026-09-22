package secrets

import (
	"fmt"
	"sort"
)

// Layout drift: does the store look the way the registry says it does?
//
// The registry has always been the mapping SSOT, and until now nothing compared
// it to the vault. Measured 2026-09-21 against a live store of 185 items: 169 of
// them (91%) sat unfoldered, the two folders that did exist were named
// `Dotfiles/apps` and `Dotfiles/infra` while every declaration said `apps` and
// `infra`, and two declared items did not exist at all.
//
// That last mismatch is not cosmetic. `BWPut.ResolveFolder` matches a folder by
// EXACT name and CREATES one when it finds none, so the next `dotf secrets set`
// provisioning an app-plane secret would have made a second folder called `apps`
// beside `Dotfiles/apps` and split the managed items across both. A declaration
// nobody checks is not a declaration; it is a comment that happens to be YAML.
//
// Read-only, by construction: this file computes findings from two inventories
// and has no writer. Converging the store is CLI-078's `reconcile`, which needs
// this report to exist first — you cannot safely converge what you cannot see.

// Drift kinds. Strings rather than an enum because they are printed, grouped and
// grepped by operators, and a stable spelling is the contract.
const (
	DriftFolderMissing = "folder-missing"
	DriftItemMissing   = "item-missing"
	DriftItemMisfiled  = "item-misfiled"
	DriftFieldMissing  = "field-missing"
)

// LayoutFinding is one disagreement between the declaration and the store.
//
// It carries coordinates and never a value — the same contract as ItemSummary,
// because these strings travel into a terminal, a CI log and a transcript.
type LayoutFinding struct {
	Kind   string
	Secret string // registry id, so the reader knows which line to edit
	Item   string
	Detail string
}

// BWDecl is one declared Bitwarden target, flattened per env var: a multi-var
// secret declares one item and several fields, and a missing field is a finding
// about that var, not about the secret as a whole.
type BWDecl struct {
	Secret string
	Var    string
	Item   string
	Field  string
	Folder string
	// Dormant marks a declaration the read path does not use yet: the registry
	// carries a `bw:` block for age-backed secrets too, as the migration target.
	//
	// Checking those is the entire reason this exists. GITHUB_PERSONAL_ACCESS_TOKEN
	// and RELEASE_TOKEN declared `github-cli-pat` and `github-release-pat`, neither
	// of which had ever been created — invisible, because a dormant declaration
	// resolves nothing and so breaks nothing, right up until the migration that
	// needs it. Meanwhile the live age source held a token GitHub answers 401 to
	// and `dotf secrets verify` reported OK, because resolving and working are
	// different claims. Measured 2026-09-21.
	Dormant bool
}

// BWDeclarations flattens every secret carrying a `bw:` block into its declared
// targets — INCLUDING the age-backed ones, whose block is dormant (see Dormant).
//
// Per-var field overrides are resolved here rather than by the caller: the seven
// X_* vars share one item and differ only by field, and a comparison that read
// the secret-level field would report six phantom mismatches.
//
// A file-exposed secret contributes one declaration under its file var. Its bw
// target is declared exactly like an env secret's; only the consumer contract
// differs. Walking expose.env alone left eight of them — KUBECONFIG, SSH_KEY, the
// recovery codes — never compared against the store.
func (r *Registry) BWDeclarations() []BWDecl {
	var out []BWDecl
	for i := range r.Secrets {
		s := &r.Secrets[i]
		if s.BW == nil {
			continue
		}
		dormant := s.Backend != BackendBW
		if f := s.Expose.File; f != nil {
			out = append(out, BWDecl{
				Secret: s.ID, Var: f.Var,
				Item: s.BW.Item, Field: s.BW.Field, Folder: s.BW.Folder,
				Dormant: dormant,
			})
			continue
		}
		for _, v := range s.Expose.Env.Vars {
			field := v.Field
			if field == "" {
				field = s.BW.Field
			}
			out = append(out, BWDecl{
				Secret: s.ID, Var: v.Name,
				Item: s.BW.Item, Field: field, Folder: s.BW.Folder,
				Dormant: dormant,
			})
		}
	}
	return out
}

// LayoutDrift compares declarations against the store's shape.
//
// Findings are ordered folder → item → field, which is the order they must be
// fixed in: a misfiled item cannot be placed into a folder that does not exist,
// and a missing field cannot be checked on an item that is absent. Reporting them
// interleaved would read as more problems than there are.
func LayoutDrift(decls []BWDecl, items []ItemSummary, folders []string) []LayoutFinding {
	folderSet := make(map[string]bool, len(folders))
	for _, f := range folders {
		folderSet[f] = true
	}
	byName := make(map[string]ItemSummary, len(items))
	for _, it := range items {
		byName[it.Name] = it
	}

	var folderF, itemF, fieldF []LayoutFinding
	// One finding per distinct problem, not per declaration: seven X_* vars
	// naming one absent item is one absent item.
	seenFolder, seenItem, seenMisfiled := map[string]bool{}, map[string]bool{}, map[string]bool{}

	for _, d := range decls {
		if d.Folder != "" && !folderSet[d.Folder] && !seenFolder[d.Folder] {
			seenFolder[d.Folder] = true
			folderF = append(folderF, LayoutFinding{
				Kind: DriftFolderMissing, Secret: d.Secret, Item: d.Item,
				Detail: fmt.Sprintf("no folder named %q exists; `dotf secrets set` would CREATE one rather than reuse an existing folder", d.Folder),
			})
		}

		it, ok := byName[d.Item]
		if !ok {
			if !seenItem[d.Item] {
				seenItem[d.Item] = true
				detail := "declared but not in the vault"
				if d.Dormant {
					detail += " (dormant declaration: nothing reads it yet, so migrating this secret would fail)"
				}
				itemF = append(itemF, LayoutFinding{
					Kind: DriftItemMissing, Secret: d.Secret, Item: d.Item,
					Detail: detail,
				})
			}
			// Its fields cannot be checked, and saying so per var would bury
			// the one fact that matters.
			continue
		}

		// A declaration with no folder states no placement, so there is nothing
		// for the item to be misfiled against. The taxonomy covers the app and
		// infra planes; the personal plane's is deferred (#586), and reading "" as
		// "must be unfoldered" would report every personal item filed by hand —
		// and hand reconcile an instruction to unfile it.
		if d.Folder != "" && it.Folder != d.Folder && !seenMisfiled[d.Item] {
			seenMisfiled[d.Item] = true
			where := it.Folder
			if where == "" {
				where = "(no folder)"
			}
			itemF = append(itemF, LayoutFinding{
				Kind: DriftItemMisfiled, Secret: d.Secret, Item: d.Item,
				Detail: fmt.Sprintf("is in %s, declared %s", where, d.Folder),
			})
		}

		if !hasField(it, d.Field) {
			fieldF = append(fieldF, LayoutFinding{
				Kind: DriftFieldMissing, Secret: d.Secret, Item: d.Item,
				Detail: fmt.Sprintf("field %q is declared for %s but the item does not carry it", d.Field, d.Var),
			})
		}
	}

	sortFindings(folderF)
	sortFindings(itemF)
	sortFindings(fieldF)
	return append(append(folderF, itemF...), fieldF...)
}

// hasField reports whether the item carries the declared field.
//
// It MIRRORS fieldFromItem, which is the function that actually resolves a value,
// and the mirroring is the whole correctness of this check: that function answers
// `notes` from the native note and `username`/`password` from the typed login
// block, consulting the custom-field list only for everything else. A checker that
// knew about custom fields alone reports every login- and note-sourced declaration
// as missing — measured, DOCKERHUB_USERNAME resolves perfectly well and was the
// first thing this reported absent.
//
// Mirroring a function is normally the defect (lesson 279). Here the alternative
// is worse: the resolver's dispatch is a three-way switch on a string, it is
// private, and exporting a "would this resolve?" predicate would put a value-
// resolving code path one refactor away from a report that must never hold one.
// The duplication is deliberate, small, and named — and the test below pins the
// two together.
func hasField(it ItemSummary, field string) bool {
	if field == "" {
		// No field declared means the item itself is the target; nothing to check.
		return true
	}
	switch field {
	case "notes":
		return it.HasNotes
	case "username":
		return it.HasUsername
	case "password":
		// No HasPassword exists, by design (see ItemSummary): the password is never
		// decoded, so login presence is the strongest honest answer.
		return it.HasLogin
	}
	for _, f := range it.Fields {
		if f == field {
			return true
		}
	}
	return false
}

func sortFindings(f []LayoutFinding) {
	sort.Slice(f, func(i, j int) bool {
		if f[i].Item != f[j].Item {
			return f[i].Item < f[j].Item
		}
		return f[i].Detail < f[j].Detail
	})
}

// UnmanagedItems returns the vault items no declaration names, sorted.
//
// Reported as a COUNT by the command rather than a list, and never as a finding:
// a personal vault legitimately holds things this repo does not manage — 161 of
// 185 here. The number is the one honest answer to "how much of my store does
// dotfiles govern?", and inventing a finding per unmanaged item would turn a
// health report into noise nobody reads.
func UnmanagedItems(decls []BWDecl, items []ItemSummary) []string {
	declared := make(map[string]bool, len(decls))
	for _, d := range decls {
		declared[d.Item] = true
	}
	var out []string
	for _, it := range items {
		if !declared[it.Name] {
			out = append(out, it.Name)
		}
	}
	sort.Strings(out)
	return out
}
