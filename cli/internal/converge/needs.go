package converge

// NotReached opens the detail of a step the run did not run because a step it
// needs failed. The cause follows it.
const NotReached = "not reached"

// Needer is a reconciler that names the earlier reconcilers it needs. When one
// of them fails, or is itself not reached, the run does not run this one. A
// need that is not part of the run (`--only`), that was skipped, or that is
// not supported on this OS does not hold anything back.
//
// A reconciler that is not a Needer needs every reconciler before it: the
// safe reading, and the one the run had before steps could say otherwise.
// Every reconciler in Registry declares its needs, in needs_registry.go.
type Needer interface {
	Needs() []string
}

// needsOf is what the reconciler at i needs, by name.
func needsOf(reconcilers []Reconciler, i int) []string {
	if n, ok := reconcilers[i].(Needer); ok {
		return n.Needs()
	}
	names := make([]string, 0, i)
	for _, r := range reconcilers[:i] {
		names = append(names, r.Name())
	}
	return names
}

// downSet holds the reconcilers that did not converge in this run, failed or
// not reached, each with the reason a step needing it is told.
type downSet map[string]string

// blocks answers why a step with these needs cannot run, "" when it can: the
// first need that is down, and through it the step that failed.
func (d downSet) blocks(needs []string) string {
	for _, n := range needs {
		why, isDown := d[n]
		switch {
		case !isDown:
		case why == n+" failed":
			return why
		default:
			return "needs " + n + ", " + why
		}
	}
	return ""
}

// Unreached lists, in order, the reconcilers a run would not reach if the
// named ones failed. It asks the same question the run asks, without running
// anything, so the registry's declarations can be read and tested as a graph.
func Unreached(reconcilers []Reconciler, failed ...string) []string {
	down := downSet{}
	for _, f := range failed {
		down[f] = f + " failed"
	}
	var out []string
	for i, r := range reconcilers {
		if _, isFailed := down[r.Name()]; isFailed {
			continue
		}
		if why := down.blocks(needsOf(reconcilers, i)); why != "" {
			down[r.Name()] = why
			out = append(out, r.Name())
		}
	}
	return out
}
