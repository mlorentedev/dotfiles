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
// and has no writer. Converging the store is `reconcile` (CLI-080), which needs
// this report to exist first — you cannot safely converge what you cannot see.

// Drift kinds. Strings rather than an enum because they are printed, grouped and
// grepped by operators, and a stable spelling is the contract.
const (
	DriftFolderMissing = "folder-missing"
	DriftItemMissing   = "item-missing"
	DriftItemMisfiled  = "item-misfiled"
	// DriftItemAmbiguous: several items share the declared name. Bitwarden allows
	// it and the reader refuses to choose, so the declaration resolves nothing.
	DriftItemAmbiguous = "item-ambiguous"
	// DriftItemFolderUnknown: the item is filed in a folder the store's folder
	// list does not carry, so its placement cannot be judged either way.
	DriftItemFolderUnknown = "item-folder-unknown"
	DriftFieldMissing      = "field-missing"
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
	// Decl is the declaration the finding was raised against — coordinates only,
	// like everything else here. It is what lets reconcile turn a finding into an
	// operation without re-deriving the comparison, so the two commands cannot
	// disagree about what has drifted.
	Decl BWDecl
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
	// From is the declared source of the value, for reconcile (CLI-080); nil for
	// most declarations.
	From *BWFrom
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
				Dormant: dormant, From: s.BW.From,
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
				Dormant: dormant, From: s.BW.From,
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
	w := newDriftWalk(items, folders)
	for _, d := range decls {
		w.check(d)
	}
	sortFindings(w.folderF)
	sortFindings(w.itemF)
	sortFindings(w.fieldF)
	return append(append(w.folderF, w.itemF...), w.fieldF...)
}

// driftWalk is LayoutDrift's state: the store as indexed once, and the findings
// so far in their three fix-order buckets. Split into one method per question so
// each stays small enough to read whole (CLI-078 review round 3).
type driftWalk struct {
	folderSet              map[string]bool
	byName                 map[string]ItemSummary
	count                  map[string]int
	seen                   map[string]bool
	folderF, itemF, fieldF []LayoutFinding
}

func newDriftWalk(items []ItemSummary, folders []string) *driftWalk {
	w := &driftWalk{
		folderSet: make(map[string]bool, len(folders)),
		byName:    make(map[string]ItemSummary, len(items)),
		count:     make(map[string]int, len(items)),
		seen:      map[string]bool{},
	}
	for _, f := range folders {
		w.folderSet[f] = true
	}
	for _, it := range items {
		w.byName[it.Name] = it
		w.count[it.Name]++
	}
	return w
}

// once reports whether (kind, key) is new, and records it. One finding per
// distinct problem, not per declaration: seven X_* vars naming one absent item
// is one absent item.
func (w *driftWalk) once(kind, key string) bool {
	k := kind + "\x00" + key
	if w.seen[k] {
		return false
	}
	w.seen[k] = true
	return true
}

func (w *driftWalk) check(d BWDecl) {
	w.checkFolder(d)
	it, ok := w.resolveItem(d)
	if !ok {
		// Its placement and fields cannot be checked, and saying so per var
		// would bury the one fact that matters.
		return
	}
	w.checkPlacement(d, it)
	w.checkField(d, it)
}

func (w *driftWalk) checkFolder(d BWDecl) {
	if d.Folder == "" || w.folderSet[d.Folder] || !w.once(DriftFolderMissing, d.Folder) {
		return
	}
	w.folderF = append(w.folderF, LayoutFinding{
		Kind: DriftFolderMissing, Secret: d.Secret, Item: d.Item, Decl: d,
		Detail: fmt.Sprintf("no folder named %q exists; `dotf secrets set` would CREATE one rather than reuse an existing folder", d.Folder),
	})
}

// resolveItem returns the one item the declaration names, or reports why there
// is none to judge.
//
// Several items with the declared name: byName holds only one of them, so judging
// its folder or fields would be judging an arbitrary item — possibly a personal
// one — while the real one sits correct. The reader refuses the name outright, so
// say exactly that and judge neither.
func (w *driftWalk) resolveItem(d BWDecl) (ItemSummary, bool) {
	if n := w.count[d.Item]; n > 1 {
		if w.once("item", d.Item) {
			w.itemF = append(w.itemF, LayoutFinding{
				Kind: DriftItemAmbiguous, Secret: d.Secret, Item: d.Item, Decl: d,
				Detail: fmt.Sprintf("the name matches %d items, so the reader refuses it; rename or remove all but one", n),
			})
		}
		return ItemSummary{}, false
	}
	it, ok := w.byName[d.Item]
	if !ok && w.once("item", d.Item) {
		detail := "declared but not in the vault"
		if d.Dormant {
			detail += " (dormant declaration: nothing reads it yet, so migrating this secret would fail)"
		}
		w.itemF = append(w.itemF, LayoutFinding{Kind: DriftItemMissing, Secret: d.Secret, Item: d.Item, Decl: d, Detail: detail})
	}
	return it, ok
}

// checkPlacement reports an item outside its declared folder.
//
// A declaration with no folder states no placement, so there is nothing for the
// item to be misfiled against. The taxonomy covers the app and infra planes; the
// personal plane's is deferred (#586), and reading "" as "must be unfoldered"
// would report every personal item filed by hand — and hand reconcile an
// instruction to unfile it.
//
// Keyed on the item alone because the registry guarantees one declared folder per
// item (checkOneFolderPerItem). Without that rule two declarations could pull one
// item two ways, and reconcile would move it back and forth forever.
//
// An item whose folder id the list cannot name is neither in the declared folder
// nor out of it as far as this walk can tell, so it gets its own finding. Read as
// "" it was reported unfoldered, and reconcile planned a move against a folder
// list it had just shown to be stale.
func (w *driftWalk) checkPlacement(d BWDecl, it ItemSummary) {
	if d.Folder == "" {
		return
	}
	if it.FolderUnresolved {
		if w.once(DriftItemMisfiled, d.Item) {
			w.itemF = append(w.itemF, LayoutFinding{
				Kind: DriftItemFolderUnknown, Secret: d.Secret, Item: d.Item, Decl: d,
				Detail: fmt.Sprintf("is filed in a folder the store's folder list does not carry, so whether it is in %s cannot be told; the list is stale", d.Folder),
			})
		}
		return
	}
	if it.Folder == d.Folder || !w.once(DriftItemMisfiled, d.Item) {
		return
	}
	where := it.Folder
	if where == "" {
		where = "(no folder)"
	}
	w.itemF = append(w.itemF, LayoutFinding{
		Kind: DriftItemMisfiled, Secret: d.Secret, Item: d.Item, Decl: d,
		Detail: fmt.Sprintf("is in %s, declared %s", where, d.Folder),
	})
}

// checkField reports a declared field the item does not carry — and a declaration
// that names no field at all, which the reader refuses just the same
// (fieldFromItem("") errors). That second case used to pass as converged: a
// target counted and never checked (CLI-078 review round 3).
func (w *driftWalk) checkField(d BWDecl, it ItemSummary) {
	if hasField(it, d.Field) || !w.once(DriftFieldMissing, d.Item+"\x00"+d.Field) {
		return
	}
	detail := fmt.Sprintf("field %q is declared for %s but the item does not carry it", d.Field, d.Var)
	if d.Field == "" {
		detail = fmt.Sprintf("no field is declared for %s, and the reader refuses an empty field, so this target resolves nothing; declare the field that carries it", d.Var)
	}
	w.fieldF = append(w.fieldF, LayoutFinding{Kind: DriftFieldMissing, Secret: d.Secret, Item: d.Item, Decl: d, Detail: detail})
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
		// fieldFromItem refuses an empty field, so an item never carries it.
		// This returned true once, and certified a target the reader cannot
		// resolve (CLI-078 review round 3).
		return false
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
// a personal vault legitimately holds things this repo does not manage — 164 of
// 187 on 2026-09-23, counting items whose name no declaration uses. (An earlier
// "161 of 185" subtracted every declared NAME, including three with no item yet;
// an absent declared item does not make a present one managed.) The number is the one honest answer to "how much of my store does
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
