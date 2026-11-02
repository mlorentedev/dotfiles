package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Copy moves one file through the agent, in the direction src and dst give,
// with no share and no mount (ADR-055 decision 5). It runs no guest command, so
// Exit stays null.
func (s WindowsSessions) Copy(_ context.Context, src, dst string, stderr io.Writer) Record {
	start := time.Now()
	name, err := CopySession(src, dst)
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	if refused, ok := s.serving(name, start); !ok {
		return refused
	}
	q, err := s.connect()
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	defer func() { _ = q.Close() }()
	guestPath := func(arg string) string { return strings.TrimPrefix(arg, name+":") }
	if _, fromGuest := GuestSide(src); fromGuest {
		err = copyOut(q, guestPath(src), dst)
	} else {
		err = copyIn(q, src, guestPath(dst))
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return s.rec(name, StatusFailed, start, err.Error())
	}
	return s.rec(name, StatusOK, start, "")
}

func copyIn(q *QGA, host, guest string) error {
	f, err := os.Open(host)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := q.WriteFile(guest, f); err != nil {
		return fmt.Errorf("copy to %s: %w", guest, err)
	}
	return nil
}

func copyOut(q *QGA, guest, host string) error {
	b, err := q.ReadFile(guest)
	if err != nil {
		return fmt.Errorf("copy from %s: %w", guest, err)
	}
	return os.WriteFile(host, b, 0o644)
}
