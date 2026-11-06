package doctor

import (
	"bytes"
	"path/filepath"
	"testing"

	envpkg "github.com/mlorentedev/dotfiles/cli/internal/env"
)

func TestCheckPathFilesFailsOnStaleGeneratedProcessValue(t *testing.T) {
	home := t.TempDir()
	mirror := filepath.Join(home, ".dotfiles")
	contract := filepath.Join(mirror, "env-contract.json")
	writeFile(t, contract, `{"env_vars":[
		{"name":"VAULT_PATH","default":{"linux":"$HOME/Projects/knowledge"}}
	]}`)
	output := envpkg.DefaultOutput("linux", mirror)
	if _, err := envpkg.Generate(envpkg.Options{
		ContractPath: contract,
		MachinePath:  envpkg.MachinePath(home),
		GOOS:         "linux",
		Home:         home,
		Output:       output,
	}); err != nil {
		t.Fatal(err)
	}

	sys := newSys(map[string]string{
		"HOME":                 home,
		"DOTFILES_DIR":         mirror,
		envpkg.GeneratedMarker: "VAULT_PATH",
		"VAULT_PATH":           filepath.Join(home, "old-vault"),
	}, nil, nil)
	sys.GOOS = "linux"
	var buf bytes.Buffer
	checkPathFiles(sys, &Config{ContractPath: contract}, capture(&buf))

	if got := statusOfLine(buf.String(), "VAULT_PATH"); got != StatusFail {
		t.Fatalf("stale generated process value status = %q, want FAIL\n%s", tagOf(got), buf.String())
	}
}
