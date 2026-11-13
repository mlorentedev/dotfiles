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

// PromptDir resolves the prompt directory under commonDir. With migrate set it
// moves a directory left under the old name once, so existing prompts stay
// reachable; a dry run passes false and the move waits for the real one. A
// regular file under the old name is not a prompt directory and is left
// alone; when both directories exist the new one wins and the old is left for
// a hand look. Two callers racing the move both resolve to the new directory:
// the loser's rename finds the old path gone and the new one in place.
func PromptDir(commonDir string, migrate bool) (string, error) {
	dir := filepath.Join(commonDir, PromptDirName)
	legacy := filepath.Join(commonDir, legacyPromptDirName)
	if st, err := os.Stat(legacy); err != nil || !st.IsDir() {
		return dir, nil
	}
	if _, err := os.Stat(dir); err == nil || !migrate {
		return dir, nil
	}
	if err := os.Rename(legacy, dir); err != nil {
		if _, again := os.Stat(dir); again == nil && os.IsNotExist(err) {
			return dir, nil
		}
		return "", fmt.Errorf("lane: moving the prompt directory %s to %s: %w", legacy, dir, err)
	}
	return dir, nil
}
