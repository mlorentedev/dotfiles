package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// probeRegistry: one bw-backed secret (probeable) and one age-backed (not), the
// two shapes the command must treat differently.
const probeRegistry = `
version: 1
secrets:
  - id: NAN_API_KEY
    plane: app
    backend: bw
    bw: { item: nan-api-key, field: api-key, folder: apps }
    expose: { env: NAN_API_KEY }
  - id: SSH_KEY
    plane: app
    backend: age
    age: ssh
    expose: { env: SSH_KEY }
`

func runProbe(t *testing.T, args ...string) (string, error) {
	t.Helper()
	useTempRegistry(t, probeRegistry)
	c := newSecretsProbeCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}

// A count of zero ran the loop zero times and exited 0 — success reported for
// work never done, which is the exact class this command was built to close,
// reproduced inside it. It must refuse instead.
func TestProbeCmd_NonPositiveCountIsRefused(t *testing.T) {
	for _, n := range []string{"0", "-5"} {
		out, err := runProbe(t, "NAN_API_KEY", "--count", n)
		if err == nil {
			t.Errorf("--count %s must be refused, not silently do nothing (output: %q)", n, out)
			continue
		}
		if !strings.Contains(err.Error(), "at least 1") {
			t.Errorf("--count %s: the error must name the constraint, got: %v", n, err)
		}
	}
}

// Probing a secret that does not resolve through the daemon has nothing to
// probe. Saying so beats reporting on a request that was never going to happen.
func TestProbeCmd_NonBWSecretIsRefused(t *testing.T) {
	_, err := runProbe(t, "SSH_KEY")
	if err == nil {
		t.Fatal("an age-backed secret has no daemon request to probe; want a refusal")
	}
	if !strings.Contains(err.Error(), "nothing to probe") {
		t.Errorf("the refusal must say why, got: %v", err)
	}
}

func TestProbeCmd_UnknownSecretIsRefused(t *testing.T) {
	_, err := runProbe(t, "NO_SUCH_SECRET")
	if err == nil {
		t.Fatal("an id absent from the registry must be refused")
	}
	if !strings.Contains(err.Error(), "not in the registry") {
		t.Errorf("the refusal must name the cause, got: %v", err)
	}
}
