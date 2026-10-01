package mem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/filelock"
)

// HandoffMemoryLockWait bounds how long one handoff operation waits for another.
const HandoffMemoryLockWait = 10 * time.Second

// LockHandoffMemory takes the cross-process lock for one MEMORY.md.
func LockHandoffMemory(memoryPath string) (func(), error) {
	return LockHandoffMemoryWithin(memoryPath, HandoffMemoryLockWait)
}

// LockHandoffMemoryWithin is LockHandoffMemory with an explicit wait for tests.
func LockHandoffMemoryWithin(memoryPath string, wait time.Duration) (func(), error) {
	canon, err := filepath.Abs(memoryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", memoryPath, err)
	}
	if realDir, err := filepath.EvalSymlinks(filepath.Dir(canon)); err == nil {
		canon = filepath.Join(realDir, filepath.Base(canon))
	}
	if runtime.GOOS == "windows" {
		canon = strings.ToLower(canon)
	}
	dir, err := handoffMemoryLockDir()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(canon))
	unlock, err := filelock.Lock(filepath.Join(dir, "handoff-"+hex.EncodeToString(sum[:8])+".lock"), wait)
	if err != nil {
		return nil, fmt.Errorf("another handoff operation is still using %s: %w", memoryPath, err)
	}
	return unlock, nil
}

func handoffMemoryLockDir() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("no runtime or cache directory for the handoff lock: %w", err)
		}
		base = cache
	}
	dir := filepath.Join(base, "dotf", "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create the handoff lock directory %s: %w", dir, err)
	}
	return dir, nil
}
