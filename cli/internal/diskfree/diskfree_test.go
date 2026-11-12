package diskfree

import "testing"

func TestFreeAnswersForATempDir(t *testing.T) {
	n, err := Free(t.TempDir())
	if err != nil || n == 0 {
		t.Fatalf("Free = %d, %v; want a positive size", n, err)
	}
}

func TestFreeFailsForAMissingPath(t *testing.T) {
	if _, err := Free(t.TempDir() + "/missing/deeper"); err == nil {
		t.Fatal("want an error for a path that does not exist")
	}
}
