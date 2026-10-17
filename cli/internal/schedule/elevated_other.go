//go:build !windows

package schedule

// Elevated is false off Windows: only the Task Scheduler asks.
func Elevated() bool { return false }
