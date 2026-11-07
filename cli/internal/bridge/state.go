// Package bridge owns dotf's non-secret lifecycle state for the role-gated
// ts-bridge client.
package bridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var profileNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// LaunchDescriptor is the reusable, non-secret part of a ts-bridge launch.
type LaunchDescriptor struct {
	Profile    string `json:"profile"`
	Config     string `json:"config,omitempty"`
	Socks5     string `json:"socks5"`
	HealthAddr string `json:"health_addr"`
}

// Validate refuses state that could escape its profile directory or expose a
// listener beyond loopback.
func (d LaunchDescriptor) Validate() error {
	if !profileNameRE.MatchString(d.Profile) {
		return fmt.Errorf("profile %q must be 1-64 letters, digits, dot, underscore or dash", d.Profile)
	}
	if d.Config != "" && !filepath.IsAbs(d.Config) {
		return fmt.Errorf("config path %q must be absolute", d.Config)
	}
	if err := loopbackAddress("SOCKS", d.Socks5); err != nil {
		return err
	}
	return loopbackAddress("health", d.HealthAddr)
}

// ProcessRecord identifies the detached process a later stop may own.
type ProcessRecord struct {
	PID           int    `json:"pid"`
	Executable    string `json:"executable"`
	StartIdentity string `json:"start_identity"`
	Fingerprint   string `json:"fingerprint"`
}

// Validate refuses an identity that cannot distinguish an owned process.
func (p ProcessRecord) Validate() error {
	switch {
	case p.PID <= 0:
		return errors.New("process PID must be positive")
	case !filepath.IsAbs(p.Executable):
		return errors.New("process executable must be an absolute path")
	case p.StartIdentity == "":
		return errors.New("process start identity is missing")
	case p.Fingerprint == "":
		return errors.New("process launch fingerprint is missing")
	default:
		return nil
	}
}

// Store persists bridge lifecycle state below dotf's per-user state directory.
type Store struct {
	Root string
}

// NewStore returns a bridge state store below stateDir.
func NewStore(stateDir string) Store {
	return Store{Root: filepath.Join(stateDir, "bridge")}
}

// SaveLaunch writes one launch descriptor atomically.
func (s Store) SaveLaunch(d LaunchDescriptor) error {
	if err := d.Validate(); err != nil {
		return err
	}
	path, err := s.profileFile(d.Profile, "launch.json")
	if err != nil {
		return err
	}
	return writeJSON(path, d)
}

// LoadLaunch reads and validates one launch descriptor.
func (s Store) LoadLaunch(profile string) (LaunchDescriptor, error) {
	var d LaunchDescriptor
	path, err := s.profileFile(profile, "launch.json")
	if err != nil {
		return d, err
	}
	if err := readJSON(path, &d); err != nil {
		return d, fmt.Errorf("read bridge launch state: %w", err)
	}
	return d, d.Validate()
}

// SaveProcess writes one owned process record atomically.
func (s Store) SaveProcess(profile string, p ProcessRecord) error {
	if err := p.Validate(); err != nil {
		return err
	}
	path, err := s.profileFile(profile, "process.json")
	if err != nil {
		return err
	}
	return writeJSON(path, p)
}

// LoadProcess reads and validates one owned process record.
func (s Store) LoadProcess(profile string) (ProcessRecord, error) {
	var p ProcessRecord
	path, err := s.profileFile(profile, "process.json")
	if err != nil {
		return p, err
	}
	if err := readJSON(path, &p); err != nil {
		return p, fmt.Errorf("read bridge process state: %w", err)
	}
	return p, p.Validate()
}

// ResolveProfile returns the named profile, or the only stored profile.
func (s Store) ResolveProfile(profile string) (string, error) {
	if profile != "" {
		if _, err := s.LoadLaunch(profile); err != nil {
			return "", err
		}
		return profile, nil
	}
	profiles, err := s.Profiles()
	if err != nil {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", errors.New("no bridge profile is configured")
	case 1:
		return profiles[0], nil
	default:
		return "", fmt.Errorf("multiple bridge profiles are configured: %s", joinProfiles(profiles))
	}
}

// Profiles lists stored launch profiles in deterministic order.
func (s Store) Profiles() ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read bridge state: %w", err)
	}
	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(s.Root, entry.Name(), "launch.json")); err == nil {
				profiles = append(profiles, entry.Name())
			}
		}
	}
	sort.Strings(profiles)
	return profiles, nil
}

func (s Store) profileFile(profile, name string) (string, error) {
	if !filepath.IsAbs(s.Root) {
		return "", fmt.Errorf("bridge state root %q must be absolute", s.Root)
	}
	if !profileNameRE.MatchString(profile) {
		return "", fmt.Errorf("profile %q must be 1-64 letters, digits, dot, underscore or dash", profile)
	}
	return filepath.Join(s.Root, profile, name), nil
}

func loopbackAddress(label, address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%s address %q is invalid: %w", label, address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("%s address %q must use a loopback IP", label, address)
	}
	return nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bridge-*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func readJSON(path string, value any) error {
	f, err := os.Open(path) //nolint:gosec // path is constrained below the state root
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(value); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("bridge state contains trailing JSON")
	}
	return nil
}

func joinProfiles(profiles []string) string {
	var out string
	for i, profile := range profiles {
		if i > 0 {
			out += ", "
		}
		out += profile
	}
	return out
}
