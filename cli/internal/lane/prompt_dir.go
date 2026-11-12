package lane

import (
	"fmt"
	"os"
	"path/filepath"
)

// PromptDirName is the directory under the common git dir where `dotf lane
// next` leaves a hand-off prompt when no backend can start the successor. It
// was `dotf-lane` until #3320, the name of the identity record a linked
// worktree's git dir carries (ADR-061): the two never shared a path, but one
// name for two artifacts in the git-dir namespace let `ls .git/dotf-lane`
// answer with the wrong one and no error.
const PromptDirName = "dotf-lane-prompts"

// legacyPromptDirName is the name the prompt directory had before #3320.
const legacyPromptDirName = "dotf-lane"

// PromptDir resolves the prompt directory under commonDir, moving a directory
// left under the old name once so existing prompts stay reachable. A regular
// file under the old name is not a prompt directory and is left alone; when
// both directories exist the new one wins and the old is left for a hand look.
func PromptDir(commonDir string) (string, error) {
	dir := filepath.Join(commonDir, PromptDirName)
	legacy := filepath.Join(commonDir, legacyPromptDirName)
	if st, err := os.Stat(legacy); err != nil || !st.IsDir() {
		return dir, nil
	}
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	if err := os.Rename(legacy, dir); err != nil {
		return "", fmt.Errorf("lane: moving the prompt directory %s to %s: %w", legacy, dir, err)
	}
	return dir, nil
}
