package agent

import (
	"context"
	"fmt"
)

// DryRun resolves a route and deliberately does not execute it.
//
// It is a first-class backend, not a test double: inspecting which pool and
// model a tier resolves to on this machine is worth doing without spending a
// slot of a shared quota, and it stays useful after the real backends land.
// The scripted fake the tests use is Go-test-only and reachable only through
// the seam — a `--backend` value that exists to make tests pass is a surface
// users can reach by accident.
type DryRun struct{}

func (DryRun) Dispatch(_ context.Context, req Request) Response {
	return Response{
		Status: StatusDryRun,
		Output: fmt.Sprintf("would dispatch role %q to %s:%s (no request was sent)",
			req.Role, req.Pool, req.Model),
	}
}
