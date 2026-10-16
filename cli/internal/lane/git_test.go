package lane

import "testing"

func TestForgeGit_MergedComesFromTheForgeAlone(t *testing.T) {
	forge := &fakeForge{prs: map[string]PRInfo{
		"fix/merged": {Number: 1, State: PRMerged},
		"fix/open":   {Number: 2, State: PROpen},
	}}
	g := ForgeGit{Forge: CachedForge(forge)}
	for branch, want := range map[string]bool{"fix/merged": true, "fix/open": false, "fix/none": false, "": false} {
		if got, err := g.IsPRMerged("/repo", branch); err != nil || got != want {
			t.Errorf("IsPRMerged(%q) = %v, %v; want %v", branch, got, err, want)
		}
	}
}

// One cache feeds the listing and Status: a branch costs one request however
// many readers ask, and wrapping a cache again adds no layer.
func TestCachedForge_IsSharedNotStacked(t *testing.T) {
	forge := &fakeForge{}
	shared := CachedForge(forge)
	for _, r := range []PRSource{shared, CachedForge(shared), ForgeGit{Forge: shared}.Forge} {
		_, _ = r.PRFor("fix/a")
	}
	if n := forge.calls["fix/a"]; n != 1 {
		t.Errorf("forge asked %d times, want 1", n)
	}
}
