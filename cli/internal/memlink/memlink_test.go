package memlink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// --- resolveVaultMemory: the three conventions -------------------------------

func TestResolveVaultMemory(t *testing.T) {
	t.Run("convention 1: 10_projects/<project>/memory", func(t *testing.T) {
		vault := t.TempDir()
		want := filepath.Join(vault, "10_projects", "myproj", "memory")
		mkdirAll(t, want)
		if got := resolveVaultMemory("/anywhere/myproj", "myproj", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 2: CWD inside the vault uses CWD/memory", func(t *testing.T) {
		vault := t.TempDir()
		cwd := filepath.Join(vault, "somedir") // under vault, no 10_projects entry
		want := filepath.Join(cwd, "memory")
		mkdirAll(t, want)
		if got := resolveVaultMemory(cwd, "somedir", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 3: work-SDK family/component slug match", func(t *testing.T) {
		vault := t.TempDir()
		want := filepath.Join(vault, "50_work", "45-development", "FamilyX", "CompY", "memory")
		mkdirAll(t, want)
		// CWD slug must contain both 'familyx' and 'compy'.
		if got := resolveVaultMemory("/dev/FamilyX/CompY-repo", "irrelevant", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 1 wins over a present convention-2 dir", func(t *testing.T) {
		vault := t.TempDir()
		conv1 := filepath.Join(vault, "10_projects", "p", "memory")
		mkdirAll(t, conv1)
		cwd := filepath.Join(vault, "p")
		mkdirAll(t, filepath.Join(cwd, "memory"))
		if got := resolveVaultMemory(cwd, "p", vault); got != conv1 {
			t.Errorf("got %q, want conv1 %q", got, conv1)
		}
	})

	t.Run("no source anywhere → empty", func(t *testing.T) {
		if got := resolveVaultMemory("/nowhere/x", "x", t.TempDir()); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// --- resolveByRepoURL: repo directory named differently from its vault slug ---

// vaultProject writes 10_projects/<slug>/context.md with the given frontmatter
// repo_url (none when empty) and, when withMemory, a memory/ dir.
func vaultProject(t *testing.T, vault, slug, repoURL string, withMemory bool) string {
	t.Helper()
	fm := "---\nid: \"" + slug + "\"\n"
	if repoURL != "" {
		fm += "repo_url: \"" + repoURL + "\"\n"
	}
	writeFile(t, filepath.Join(vault, "10_projects", slug, "context.md"), fm+"---\n\n# "+slug+"\n")
	mem := filepath.Join(vault, "10_projects", slug, "memory")
	if withMemory {
		mkdirAll(t, mem)
	}
	return mem
}

func TestResolveVaultMemory_RepoURL(t *testing.T) {
	t.Run("repo name differs from the slug: matched by repo_url basename", func(t *testing.T) {
		vault := t.TempDir()
		want := vaultProject(t, vault, "suite", "ssh://git@host:2222/owner/sensortool.git", true)
		vaultProject(t, vault, "other", "https://example.com/owner/other", true)
		if got := resolveVaultMemory("/home/me/Projects/sensortool", "sensortool", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("convention 1 wins over a repo_url match", func(t *testing.T) {
		vault := t.TempDir()
		conv1 := vaultProject(t, vault, "proj", "", true)
		vaultProject(t, vault, "elsewhere", "https://example.com/owner/proj.git", true)
		if got := resolveVaultMemory("/x/proj", "proj", vault); got != conv1 {
			t.Errorf("got %q, want conv1 %q", got, conv1)
		}
	})

	t.Run("two projects claim the repo → no link rather than the wrong one", func(t *testing.T) {
		vault := t.TempDir()
		vaultProject(t, vault, "a", "https://example.com/owner/dup.git", true)
		vaultProject(t, vault, "b", "ssh://git@host/owner/dup", true)
		if got := resolveVaultMemory("/x/dup", "dup", vault); got != "" {
			t.Errorf("got %q, want empty on an ambiguous repo_url", got)
		}
	})

	t.Run("matched project without memory/ → empty", func(t *testing.T) {
		vault := t.TempDir()
		vaultProject(t, vault, "suite", "https://example.com/owner/tool.git", false)
		if got := resolveVaultMemory("/x/tool", "tool", vault); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("a trailing YAML comment on repo_url still matches", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "suite", "context.md"),
			"---\nrepo_url: https://example.com/owner/tool.git  # canonical\n---\n")
		want := filepath.Join(vault, "10_projects", "suite", "memory")
		mkdirAll(t, want)
		if got := resolveVaultMemory("/x/tool", "tool", vault); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("unterminated frontmatter is not read as frontmatter", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "suite", "context.md"),
			"---\nid: suite\n\n# body\nrepo_url: https://example.com/owner/tool.git\n")
		mkdirAll(t, filepath.Join(vault, "10_projects", "suite", "memory"))
		if got := resolveVaultMemory("/x/tool", "tool", vault); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("repo_url outside the frontmatter is ignored", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "suite", "context.md"),
			"---\nid: suite\n---\n\nrepo_url: https://example.com/owner/tool.git\n")
		mkdirAll(t, filepath.Join(vault, "10_projects", "suite", "memory"))
		if got := resolveVaultMemory("/x/tool", "tool", vault); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// What doctor reads for a repo named differently from its vault slug: repairable on
// a fresh machine, the real-dir divergence once an agent has written there.
func TestStatus_RepoURL(t *testing.T) {
	vault := t.TempDir()
	vaultProject(t, vault, "suite", "ssh://git@host:2222/owner/sensortool.git", true)

	fresh := filepath.Join(t.TempDir(), "memory")
	if got := Status("/x/sensortool", fresh, "", vault); got != StateRepairable {
		t.Errorf("fresh target: got %v, want StateRepairable", got)
	}

	real := filepath.Join(t.TempDir(), "memory")
	writeFile(t, filepath.Join(real, "agent-own.md"), "data")
	if got := Status("/x/sensortool", real, "", vault); got != StateRealDir {
		t.Errorf("real dir: got %v, want StateRealDir", got)
	}
}

func TestEnsure_LinksByRepoURL(t *testing.T) {
	vault := t.TempDir()
	src := vaultProject(t, vault, "suite", "ssh://git@host:2222/owner/sensortool.git", true)
	writeFile(t, filepath.Join(src, "MEMORY.md"), "suite-memory")
	cwd := filepath.Join(t.TempDir(), "sensortool")
	target := filepath.Join(t.TempDir(), "memory")

	msg, err := Ensure(cwd, target, "", vault)
	if err != nil || msg == "" {
		t.Fatalf("Ensure: msg=%q err=%v, want a created link", msg, err)
	}
	// Read through the link rather than resolving it: EvalSymlinks does not follow
	// a Windows junction.
	got, err := os.ReadFile(filepath.Join(target, "MEMORY.md"))
	if err != nil {
		t.Fatalf("link did not read through: %v", err)
	}
	if string(got) != "suite-memory" {
		t.Errorf("read %q through link, want suite-memory", got)
	}
}

// --- ClaudeProjectKey / ClaudeMemoryTarget: cross-OS encoding ----------------

func TestClaudeProjectKey(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"posix", "/home/me/Projects/dotfiles", "-home-me-Projects-dotfiles"},
		{"windows drive colon", `C:\Users\mlorente\Projects\Workspace\dotfiles`, "C--Users-mlorente-Projects-Workspace-dotfiles"},
		// #1553: Claude maps '.' too; keeping it produced a key Claude never reads.
		{"dotted repo name", "/home/manu/Projects/svqtriana.github.io", "-home-manu-Projects-svqtriana-github-io"},
		{"dotted repo name on windows", `C:\Users\me\Projects\svqtriana.github.io`, "C--Users-me-Projects-svqtriana-github-io"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClaudeProjectKey(tc.in); got != tc.want {
				t.Errorf("ClaudeProjectKey(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestClaudeMemoryTarget(t *testing.T) {
	got := ClaudeMemoryTarget("/h", "/home/me/proj")
	want := filepath.Join("/h", ".claude", "projects", "-home-me-proj", "memory")
	if got != want {
		t.Errorf("ClaudeMemoryTarget = %q, want %q", got, want)
	}
}

// --- Status: the read-only classifier doctor reports on ----------------------

func TestStatus(t *testing.T) {
	t.Run("no source → StateNoSource", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "memory")
		if got := Status("/nowhere/x", target, "x", t.TempDir()); got != StateNoSource {
			t.Errorf("got %v, want StateNoSource", got)
		}
	})

	t.Run("source exists, target missing → StateRepairable", func(t *testing.T) {
		vault := t.TempDir()
		mkdirAll(t, filepath.Join(vault, "10_projects", "p", "memory"))
		target := filepath.Join(t.TempDir(), "memory")
		if got := Status("/x/p", target, "p", vault); got != StateRepairable {
			t.Errorf("got %v, want StateRepairable", got)
		}
	})

	t.Run("real non-empty dir → StateRealDir (Ensure would leave it; doctor must not destroy)", func(t *testing.T) {
		vault := t.TempDir()
		mkdirAll(t, filepath.Join(vault, "10_projects", "p", "memory"))
		target := filepath.Join(t.TempDir(), "memory")
		writeFile(t, filepath.Join(target, "own.md"), "agent data")
		if got := Status("/x/p", target, "p", vault); got != StateRealDir {
			t.Errorf("got %v, want StateRealDir", got)
		}
	})

	t.Run("already linked → StateLinked", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "p", "memory", "MEMORY.md"), "x")
		target := filepath.Join(t.TempDir(), "agent", "p", "memory")
		if _, err := Ensure("/w/p", target, "p", vault); err != nil {
			t.Fatalf("Ensure setup: %v", err)
		}
		if got := Status("/w/p", target, "p", vault); got != StateLinked {
			t.Errorf("got %v, want StateLinked", got)
		}
	})
}

// --- Ensure: idempotency + happy path ----------------------------------------

func TestEnsure(t *testing.T) {
	t.Run("no source → no-op, empty message", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "agent", "x", "memory")
		msg, err := Ensure("/nowhere/x", target, "x", t.TempDir())
		if err != nil || msg != "" {
			t.Errorf("msg=%q err=%v", msg, err)
		}
		if _, statErr := os.Lstat(target); statErr == nil {
			t.Errorf("target should not have been created")
		}
	})

	t.Run("target is a non-empty dir → no-op", func(t *testing.T) {
		vault := t.TempDir()
		mkdirAll(t, filepath.Join(vault, "10_projects", "p", "memory"))
		target := filepath.Join(t.TempDir(), "memory")
		writeFile(t, filepath.Join(target, "agent-own.md"), "data")
		msg, err := Ensure("/x/p", target, "p", vault)
		if err != nil || msg != "" {
			t.Errorf("expected no-op, got msg=%q err=%v", msg, err)
		}
	})

	t.Run("happy path creates a link reading through to the vault source", func(t *testing.T) {
		vault := t.TempDir()
		src := filepath.Join(vault, "10_projects", "myproj", "memory")
		writeFile(t, filepath.Join(src, "MEMORY.md"), "vault-content")
		target := filepath.Join(t.TempDir(), "agent", "myproj", "memory")

		msg, err := Ensure("/work/myproj", target, "myproj", vault)
		if err != nil {
			t.Fatalf("Ensure: %v", err)
		}
		if !strings.Contains(msg, "Created") || !strings.Contains(msg, "myproj") {
			t.Errorf("unexpected message: %q", msg)
		}
		got, err := os.ReadFile(filepath.Join(target, "MEMORY.md"))
		if err != nil {
			t.Fatalf("link did not read through: %v", err)
		}
		if string(got) != "vault-content" {
			t.Errorf("read %q through link, want vault-content", got)
		}
	})

	t.Run("already linked → no-op on a second call", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "p", "memory", "MEMORY.md"), "x")
		target := filepath.Join(t.TempDir(), "agent", "p", "memory")

		if _, err := Ensure("/w/p", target, "p", vault); err != nil {
			t.Fatalf("first Ensure: %v", err)
		}
		msg, err := Ensure("/w/p", target, "p", vault)
		if err != nil || msg != "" {
			t.Errorf("second call should be a no-op, got msg=%q err=%v", msg, err)
		}
	})
}

// --- a dangling link: the vault source it named moved or was archived -------

// danglingLink leaves target a link to a directory that no longer exists, the
// state a vault project leaves behind when it moves to 50_work or 90_archive.
func danglingLink(t *testing.T, target string) {
	t.Helper()
	gone := filepath.Join(t.TempDir(), "10_projects", "moved", "memory")
	mkdirAll(t, gone)
	mkdirAll(t, filepath.Dir(target))
	if err := createLink(gone, target); err != nil {
		t.Fatalf("createLink: %v", err)
	}
	if err := os.RemoveAll(filepath.Dir(filepath.Dir(gone))); err != nil {
		t.Fatalf("remove link source: %v", err)
	}
}

func TestDanglingLink(t *testing.T) {
	t.Run("a source resolves: Status is repairable and Ensure relinks to it", func(t *testing.T) {
		vault := t.TempDir()
		writeFile(t, filepath.Join(vault, "10_projects", "p", "memory", "MEMORY.md"), "current")
		target := filepath.Join(t.TempDir(), "agent", "p", "memory")
		danglingLink(t, target)

		if got := Status("/w/p", target, "p", vault); got != StateRepairable {
			t.Fatalf("Status = %v, want StateRepairable", got)
		}
		msg, err := Ensure("/w/p", target, "p", vault)
		if err != nil || msg == "" {
			t.Fatalf("Ensure: msg=%q err=%v, want a relink", msg, err)
		}
		got, err := os.ReadFile(filepath.Join(target, "MEMORY.md"))
		if err != nil || string(got) != "current" {
			t.Fatalf("read through the relinked target: %q, %v", got, err)
		}
	})

	t.Run("no source resolves: Status says dangling and Ensure leaves it", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "agent", "x", "memory")
		danglingLink(t, target)

		if got := Status("/nowhere/x", target, "x", t.TempDir()); got != StateDangling {
			t.Fatalf("Status = %v, want StateDangling", got)
		}
		if msg, err := Ensure("/nowhere/x", target, "x", t.TempDir()); err != nil || msg != "" {
			t.Fatalf("Ensure: msg=%q err=%v, want a no-op", msg, err)
		}
		if !isLink(target) {
			t.Fatal("Ensure removed a link it had nothing to replace with")
		}
	})
}

// --- createLink: path components containing a cmd.exe word delimiter -------

// On Windows, mklink runs through cmd.exe's own tokenizer (createLink's
// Windows implementation, memlink_windows.go), which treats a bare comma,
// semicolon or equals sign as a word separator outside quotes — unlike Go's
// ordinary argv escaping, which only quotes on space/tab/quote and so let
// these through unquoted (HARNESS-050, #575). On POSIX this exercises
// os.Symlink, which never had the defect; the table still documents the
// contract and catches a regression on either OS.
func TestCreateLink_CmdDelimiterPaths(t *testing.T) {
	for _, name := range []string{
		"comma,here", "semi;colon", "paren(here)", "equals=here", "space here",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			src := filepath.Join(root, "src "+name)
			writeFile(t, filepath.Join(src, "MEMORY.md"), "content")
			target := filepath.Join(root, "link "+name)

			if err := createLink(src, target); err != nil {
				t.Fatalf("createLink(%q, %q): %v", src, target, err)
			}
			got, err := os.ReadFile(filepath.Join(target, "MEMORY.md"))
			if err != nil {
				t.Fatalf("link did not read through: %v", err)
			}
			if string(got) != "content" {
				t.Errorf("read %q through link, want %q", got, "content")
			}
		})
	}
}
