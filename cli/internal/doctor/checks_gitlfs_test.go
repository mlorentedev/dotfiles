package doctor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
)

// lfsSys is a machine with git on PATH, git-lfs installed when lfs, and the
// given global config; no system config.
func lfsSys(t *testing.T, lfs bool, cfg map[string][]string) *System {
	t.Helper()
	s := newSys(map[string]string{"HOME": t.TempDir()}, []string{"git"}, nil)
	s.CommandStdoutDir = func(_, name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		switch {
		case cmd == "git lfs version" && lfs:
			return "git-lfs/3.8.0\n", nil
		case cmd == "git lfs version":
			return "", gitExitErr(1)
		case strings.HasPrefix(cmd, "git config --global --get-all "):
			if v, ok := cfg[args[3]]; ok {
				return strings.Join(v, "\n") + "\n", nil
			}
			return "", gitExitErr(1)
		case strings.HasPrefix(cmd, "git config --system --get-all "):
			return "", gitExitErr(1)
		}
		t.Fatalf("unexpected command: %s", cmd)
		return "", nil
	}
	return s
}

func installedLFS() map[string][]string {
	cfg := map[string][]string{}
	for _, k := range gitconfig.LFSFilter {
		cfg[k.Key] = []string{k.Value}
	}
	return cfg
}

func TestCheckGitLFS_ByState(t *testing.T) {
	differ := installedLFS()
	differ["filter.lfs.required"] = []string{"false"}
	for _, tc := range []struct {
		name string
		lfs  bool
		cfg  map[string][]string
		want string
		st   Status
	}{
		{"filter in place", true, installedLFS(), "git-lfs filter", StatusPass},
		{"no filter", true, map[string][]string{}, "dotf converge --only git-lfs", StatusWarn},
		{"not installed", false, map[string][]string{}, "dotf tools install git-lfs", StatusWarn},
		{"a different value", true, differ, "left as it is", StatusWarn},
		{"absent beside a different value", true, map[string][]string{"filter.lfs.smudge": {"x"}}, "git lfs install --force", StatusWarn},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			checkGitLFS(lfsSys(t, tc.lfs, tc.cfg), capture(&b))
			if got := statusOfLine(b.String(), tc.want); got != tc.st {
				t.Errorf("line naming %q is %s, want %s\n%s", tc.want, tagOf(got), tagOf(tc.st), b.String())
			}
		})
	}
}

func TestCheckGitLFS_SkipsWithoutGit(t *testing.T) {
	s := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
	var b bytes.Buffer
	checkGitLFS(s, capture(&b))
	if statusOfLine(b.String(), "git not on PATH") != StatusSkip {
		t.Errorf("want a SKIP\n%s", b.String())
	}
}
