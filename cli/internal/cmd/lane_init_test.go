package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// `lane init` provisions the lanes and writes no marker; the marker is written
// only on request, for the commit that declares the repository.
func TestLaneInit_WritesTheMarkerOnlyOnRequest(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"lane", "init"}, false},
		{[]string{"lane", "init", "--write-marker"}, true},
	} {
		repo := worktreeTestRepo(t)
		t.Chdir(repo)
		if _, stderr, err := execute(t, tc.args...); err != nil {
			t.Fatalf("%v: %v %s", tc.args, err, stderr)
		}
		_, err := os.Stat(filepath.Join(repo, filepath.FromSlash(lane.LanesMarkerFile)))
		if (err == nil) != tc.want {
			t.Errorf("%v: marker present = %v, want %v", tc.args, err == nil, tc.want)
		}
	}
}
