//go:build !linux && !darwin

package worktree

// isCallerInside has no implementation off Linux and darwin, and answers "not
// inside".
//
// That is the permissive answer, and it is chosen rather than inherited: Gate f
// refuses off Linux because an inert sweep only costs disk, but a done that
// refused every call would leave no way to remove a worktree at all. What still
// holds everywhere is the caller's own cwd (DoneOptions.Cwd), which is the
// launch directory of an agent's shell tool unless it changed directory first.
// The walk of the process tree (ToolHelp32Snapshot on Windows) is the gap,
// tracked on #1653.
func isCallerInside(_ string) (ancestor, bool) {
	return ancestor{}, false
}
