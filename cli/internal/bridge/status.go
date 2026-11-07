package bridge

import (
	"errors"
	"fmt"
	"os"
)

// State is one observable bridge lifecycle state.
type State string

const (
	StateStopped           State = "stopped"
	StateUnowned           State = "unowned"
	StateStarting          State = "starting"
	StateDestinationFailed State = "destination_failed"
	StateHealthy           State = "healthy"
)

// Observation is the evidence status classification consumes.
type Observation struct {
	ProcessPresent   bool
	ProcessOwned     bool
	ListenerReady    bool
	DestinationReady bool
}

// Service resolves stored state and produces a read-only lifecycle verdict.
type Service struct {
	Store   Store
	Inspect func(ProcessRecord) (ProcessInspection, error)
	Probe   func(LaunchDescriptor) Health
}

// StatusResult is the human and JSON lifecycle verdict.
type StatusResult struct {
	Profile string `json:"profile"`
	State   State  `json:"state"`
	Error   string `json:"error,omitempty"`
	Why     string `json:"why,omitempty"`
	Fix     string `json:"fix,omitempty"`
}

// Classify turns process and network evidence into one state.
func Classify(o Observation) State {
	switch {
	case !o.ProcessPresent:
		return StateStopped
	case !o.ProcessOwned:
		return StateUnowned
	case !o.ListenerReady:
		return StateStarting
	case !o.DestinationReady:
		return StateDestinationFailed
	default:
		return StateHealthy
	}
}

// NewStatusResult renders an actionable verdict for state.
func NewStatusResult(profile string, state State) StatusResult {
	r := StatusResult{Profile: profile, State: state}
	switch state {
	case StateStopped:
		r.Error = "bridge is stopped"
		r.Why = "no owned bridge process is running"
		r.Fix = "run `dotf bridge start --profile " + profile + "`"
	case StateUnowned:
		r.Error = "bridge process identity is unowned"
		r.Why = "the recorded PID does not match the managed launch; refusing to signal it"
		r.Fix = "inspect the bridge state and start the profile again after the foreign process is resolved"
	case StateStarting:
		r.Error = "bridge is not ready"
		r.Why = "the owned process is running but its listener is not ready"
		r.Fix = "read the bounded bridge log, then retry `dotf bridge status --profile " + profile + "`"
	case StateDestinationFailed:
		r.Error = "bridge destination is unreachable"
		r.Why = "the listener is ready but the private destination failed its TLS health probe"
		r.Fix = "check the SSH bootstrap and private route, then restart the profile"
	}
	return r
}

// Status reports one profile without changing process or network state.
func (s Service) Status(profile string) (StatusResult, error) {
	resolved, err := s.Store.ResolveProfile(profile)
	if err != nil {
		return StatusResult{}, err
	}
	launch, err := s.Store.LoadLaunch(resolved)
	if err != nil {
		return StatusResult{}, err
	}
	process, err := s.Store.LoadProcess(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return NewStatusResult(resolved, StateStopped), nil
	}
	if err != nil {
		return StatusResult{}, err
	}
	if s.Inspect == nil {
		return StatusResult{}, errors.New("bridge process inspector is not configured")
	}
	inspection, err := s.Inspect(process)
	if err != nil {
		return StatusResult{}, fmt.Errorf("inspect bridge process: %w", err)
	}
	observation := Observation{
		ProcessPresent: inspection.Present,
		ProcessOwned:   inspection.Owned,
	}
	if inspection.Present && inspection.Owned {
		if s.Probe == nil {
			return StatusResult{}, errors.New("bridge health probe is not configured")
		}
		health := s.Probe(launch)
		observation.ListenerReady = health.ListenerReady
		observation.DestinationReady = health.DestinationReady
	}
	return NewStatusResult(resolved, Classify(observation)), nil
}
