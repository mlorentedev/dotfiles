// Package diskfree reports the space left on the volume that holds a path, on
// every OS dotf runs on. It answers what an unprivileged user may still write,
// not the raw free block count: on Unix the blocks reserved for root are not
// counted, and on Windows a disk quota is.
package diskfree

// Free returns the bytes available to the current user on the volume that
// holds path.
func Free(path string) (uint64, error) { return free(path) }

// Low is the free space under which a workstation is short of disk: the
// cache-gc converge step runs whatever its weekly cadence says, and doctor
// warns. An absolute size, not a share of the volume: what runs out is room
// for a build or an image pull, whose size does not scale with the disk.
const Low uint64 = 20 << 30
