package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// copyBackend keeps `limactl copy` off rsync, whose remote shell splits a guest
// path at a space and which copies a directory to nowhere with exit 0 (measured
// on Lima 2.2.1). Over scp a path is literal and a directory fails out loud.
const copyBackend = "--backend=scp"

// GuestSide answers whether arg is <session>:<path>. A host path with a colon
// is written ./a:b, whose prefix is no session name.
func GuestSide(arg string) (session string, ok bool) {
	name, p, found := strings.Cut(arg, ":")
	if !found || p == "" || ValidSessionName(name) != nil {
		return "", false
	}
	return name, true
}

// CopySession names the session of a transfer, which has exactly one guest side.
func CopySession(src, dst string) (string, error) {
	from, fromGuest := GuestSide(src)
	to, toGuest := GuestSide(dst)
	switch {
	case fromGuest && toGuest:
		return "", errors.New("one side of cp is the host: a guest cannot copy to a guest")
	case fromGuest:
		return from, nil
	case toGuest:
		return to, nil
	}
	return "", errors.New("one side of cp is <session>:<guest path>")
}

// Copy moves one file through the channel, in the direction src and dst give,
// with no share and no mount (ADR-055 decision 5). It runs no guest command, so
// Exit stays null. What limactl says goes to stderr and into the record.
func (s LinuxSessions) Copy(ctx context.Context, src, dst string, stderr io.Writer) Record {
	start := time.Now()
	name, err := CopySession(src, dst)
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	if refused, ok := s.serving(ctx, name, start); !ok {
		return refused
	}
	limaPath := func(arg string) string {
		if _, guest := GuestSide(arg); guest {
			return "dotf-" + arg
		}
		return arg
	}
	var errRec capped
	copyErr := s.L(ctx, nil, io.Discard, io.MultiWriter(&errRec, stderr), "copy", copyBackend, limaPath(src), limaPath(dst))
	r := s.rec(name, StatusOK, start, errRec.buf.String())
	r.Truncated = errRec.over
	if copyErr != nil {
		r.Status = StatusFailed
		r.Stderr += fmt.Sprintf("limactl copy: %v\n", copyErr)
	}
	return r
}
