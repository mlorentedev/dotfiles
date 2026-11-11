package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

type ciFixture map[string]map[string]string // repo -> path -> content

func (f ciFixture) List(repo string) ([]string, error) {
	var out []string
	for p := range f[repo] {
		out = append(out, p)
	}
	return out, nil
}

func (f ciFixture) Read(repo, path, _ string) ([]byte, error) {
	return []byte(f[repo][path]), nil
}

func TestRunCICheckReportsNamesAndFails(t *testing.T) {
	src := ciFixture{
		"o/bad": {".github/workflows/a.yml": "jobs:\n  a:\n    env:\n      K: ${{ secrets.NEEDED }}\n"},
		"o/ok":  {".github/workflows/a.yml": "jobs:\n  a:\n    env:\n      K: ${{ secrets.HAVE }}\n"},
	}
	reg := &secrets.Registry{Secrets: []secrets.Secret{{
		ID: "have", Consumers: []string{"ci:o/ok"},
		Expose: secrets.Expose{Env: secrets.EnvExpose{Vars: []secrets.EnvVar{{Name: "HAVE"}}}},
	}}}
	var out bytes.Buffer
	err := runCICheck(&out, secrets.NewCIChecker(src, reg), []string{"o/bad", "o/ok"})
	if err == nil || !strings.Contains(err.Error(), "1 of 2") {
		t.Fatalf("err = %v, want 1 of 2 failing", err)
	}
	want := "o/bad: FAIL\n  missing  NEEDED\no/ok: ok\n"
	if out.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", out.String(), want)
	}
}

type ciFailing struct{}

func (ciFailing) List(string) ([]string, error)               { return nil, errors.New("boom") }
func (ciFailing) Read(string, string, string) ([]byte, error) { return nil, errors.New("boom") }

func TestRunCICheckNamesTheRepositoryOnce(t *testing.T) {
	var out bytes.Buffer
	err := runCICheck(&out, secrets.NewCIChecker(ciFailing{}, &secrets.Registry{}), []string{"o/broken"})
	if err == nil {
		t.Fatal("an unreadable repository must fail")
	}
	if want := "o/broken: ERROR list workflows: boom\n"; out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}
