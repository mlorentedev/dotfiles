package worktree

// ProcessInside is Gate f's reading for one path: whether a process on this
// host has its working directory in it, and how many processes the scan could
// not read. supported is false where the platform has no scan; the reading
// then says nothing about liveness, and a caller must not take its Inside as
// an answer either way.
//
// It is read-only. `lane status` asks it for every worktree, where sweep asks
// only for its reap candidates.
func ProcessInside(path string) (r GateFReading, supported bool) {
	if !processDiscoverySupported {
		return GateFReading{}, false
	}
	return hostProcessInside(path), true
}
