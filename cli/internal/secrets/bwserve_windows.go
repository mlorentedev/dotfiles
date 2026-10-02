//go:build windows

package secrets

import "syscall"

// detachedProcess is Win32's DETACHED_PROCESS creation flag. The syscall
// package exports CREATE_NEW_PROCESS_GROUP but not this one; golang.org/x/sys
// does, but promoting it to a direct dependency for a single constant is not
// worth the diff — the value is documented and fixed.
const detachedProcess = 0x00000008

// bwServeDetachAttr detaches the daemon from this process's console so it
// outlives the terminal that started it (the whole point of one unlock serving
// every later `dotf` call). Three properties give the Windows analogue of Setsid
// plus hidden execution for shim binaries:
//
//   - CREATE_NEW_PROCESS_GROUP re-routes Ctrl+C so a Ctrl+C to the CLI does
//     not also hit the daemon. It does NOT detach from the console.
//   - DETACHED_PROCESS creates the child with no console at all. Without it a
//     console-subsystem child stays attached to its parent's console, and when
//     that console closes Windows delivers CTRL_CLOSE_EVENT and terminates the
//     child.
//   - HideWindow: true passes SW_HIDE to STARTF_USESHOWWINDOW, which suppresses
//     the visible console window when executing through a GUI/Shim boundary
//     (like Scoop wrappers). CREATE_NO_WINDOW is ignored when combined with
//     DETACHED_PROCESS, making HideWindow the correct semantic flag.
//
// TestBWServeDetachAttr_ChildHasNoConsole asserts the property by effect.
func bwServeDetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
}
