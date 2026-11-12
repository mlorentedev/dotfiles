// Package diskfree reports the space left on the volume that holds a path, on
// every OS dotf runs on. It answers what an unprivileged user may still write,
// not the raw free block count: on Unix the blocks reserved for root are not
// counted, and on Windows a disk quota is.
package diskfree

// Free returns the bytes available to the current user on the volume that
// holds path.
func Free(path string) (uint64, error) { return free(path) }
