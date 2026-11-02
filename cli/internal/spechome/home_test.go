package spechome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	githubOrigin    = "git@github.com:mlorentedev/dotfiles.git"
	giteaHTTPS      = "https://deploy:s3cr3t@git.example.com/teledyne/projects-toolkit.git"
	giteaSSH        = "git@git.example.com:teledyne/projects-toolkit.git"
	declaredHomeRaw = "mlorentedev/knowledge\n"
)

func homeRoot(t *testing.T, bitacoraFile string) string {
	t.Helper()
	root := t.TempDir()
	if bitacoraFile != "" {
		if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "specs", bitacoraRepoFile), []byte(bitacoraFile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestResolvePrecedence(t *testing.T) {
	cases := []struct {
		name, flag, file, env, origin string
		wantIssues, wantArchive       string
	}{
		{"origin alone", "", "", "", githubOrigin, "mlorentedev/dotfiles", "mlorentedev/dotfiles"},
		{"env beats origin", "", "", "o/env", githubOrigin, "o/env", "mlorentedev/dotfiles"},
		{"file beats env", "", declaredHomeRaw, "o/env", githubOrigin, "mlorentedev/knowledge", "mlorentedev/dotfiles"},
		{"flag beats file", "o/flag", declaredHomeRaw, "o/env", githubOrigin, "o/flag", "mlorentedev/dotfiles"},
		{"no origin, flag", "o/flag", "", "", "", "o/flag", ""},
		{"gitea origin, file", "", declaredHomeRaw, "", giteaSSH, "mlorentedev/knowledge", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := Resolve(Inputs{
				Root: homeRoot(t, tc.file), Flag: tc.flag, FlagName: "--repo", Env: tc.env, OriginURL: tc.origin,
			})
			if err != nil {
				t.Fatal(err)
			}
			if h.Issues != tc.wantIssues || h.Archive != tc.wantArchive {
				t.Fatalf("got issues=%q archive=%q, want %q %q", h.Issues, h.Archive, tc.wantIssues, tc.wantArchive)
			}
		})
	}
}

func TestResolveBitacoraRepoFileMalformed(t *testing.T) {
	for _, body := range []string{"", "\n", "a/b c/d\n", "a/b\nc/d\n", "not-a-slug\n"} {
		_, err := Resolve(Inputs{Root: homeRoot(t, body+" "), FlagName: "--repo", OriginURL: githubOrigin})
		if err == nil || !strings.Contains(err.Error(), filepath.Join("specs", bitacoraRepoFile)) {
			t.Errorf("file %q: want an error naming the file, got %v", body, err)
		}
	}
}

// A file read before the flag is still read: a broken committed declaration
// is loud on every run, not only on the runs that happen to rely on it.
func TestResolveBitacoraRepoFileMalformedEvenUnderAFlag(t *testing.T) {
	_, err := Resolve(Inputs{Root: homeRoot(t, "nope\n"), Flag: "o/flag", FlagName: "--repo", OriginURL: githubOrigin})
	if err == nil {
		t.Fatal("want an error for the malformed file")
	}
}

func TestResolveFailsClosedOffGitHub(t *testing.T) {
	cases := []struct{ name, origin, wantForge, wantHost string }{
		{"gitea https", giteaHTTPS, "other", "git.example.com"},
		{"gitea ssh", giteaSSH, "other", "git.example.com"},
		{"absent", "", "absent", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Resolve(Inputs{Root: homeRoot(t, ""), FlagName: "--bitacora-repo", OriginURL: tc.origin})
			if err == nil {
				t.Fatal("want a refusal")
			}
			msg := err.Error()
			for _, w := range []string{tc.wantForge, tc.wantHost, "--bitacora-repo", filepath.Join("specs", bitacoraRepoFile)} {
				if !strings.Contains(msg, w) {
					t.Errorf("message missing %q:\n%s", w, msg)
				}
			}
			if strings.Contains(msg, "s3cr3t") || strings.Contains(msg, "projects-toolkit") {
				t.Errorf("message leaks the origin URL:\n%s", msg)
			}
		})
	}
}

// A checkout that declares .gitea/ is classified as Gitea, as `dotf init` does.
func TestResolveNamesAGiteaCheckout(t *testing.T) {
	root := homeRoot(t, "")
	if err := os.MkdirAll(filepath.Join(root, ".gitea"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Resolve(Inputs{Root: root, FlagName: "--repo", OriginURL: giteaSSH})
	if err == nil || !strings.Contains(err.Error(), "gitea") {
		t.Fatalf("want the error to name gitea, got %v", err)
	}
}

func TestResolveRejectsAMalformedFlagOrEnv(t *testing.T) {
	for _, in := range []Inputs{
		{Flag: "nope", FlagName: "--repo", OriginURL: githubOrigin},
		{Env: "nope", FlagName: "--repo", OriginURL: githubOrigin},
	} {
		in.Root = homeRoot(t, "")
		if _, err := Resolve(in); err == nil {
			t.Errorf("%+v: want an error", in)
		}
	}
}
