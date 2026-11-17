package doctor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func diskSys(free uint64, err error) *System {
	return &System{
		Getenv:   func(k string) string { return map[string]string{"HOME": "/home/u"}[k] },
		DiskFree: func(string) (uint64, error) { return free, err },
	}
}

func TestDiskFree_LowWarnsAndNamesTheRemedy(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	checkDiskFree(diskSys(5<<30, nil), rep)
	got := buf.String()
	if rep.Warnings() != 1 || !strings.Contains(got, "dotf converge --only cache-gc") || !strings.Contains(got, "5.0 GiB") {
		t.Fatalf("want one warning naming the remedy, got %d: %s", rep.Warnings(), got)
	}
}

func TestDiskFree_EnoughPasses(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	checkDiskFree(diskSys(200<<30, nil), rep)
	if rep.Warnings() != 0 {
		t.Fatalf("200 GiB free must pass: %s", buf.String())
	}
}

func TestDiskFree_UnmeasurableWarnsNeverPasses(t *testing.T) {
	var buf bytes.Buffer
	rep := capture(&buf)
	checkDiskFree(diskSys(0, errors.New("statfs: no such file")), rep)
	if rep.Warnings() != 1 || !strings.Contains(buf.String(), "cannot measure") {
		t.Fatalf("an unreadable measurement must warn, not pass: %s", buf.String())
	}
}
