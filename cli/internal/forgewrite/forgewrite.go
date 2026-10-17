// Package forgewrite is the one write surface agents use to talk to a forge:
// comment on an issue or pull request today, more verbs later. It exists so a
// body never rides on a command line, where a permission glob can match its
// prose, and so one set of rules covers every forge (TOOL-030, ADR-050).
//
// It has no merge and no close verb on purpose: merging stays `dotf pr land`
// behind the merge grant, and closing stays with the owner.
package forgewrite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Repo names one repository on one forge.
type Repo struct{ Host, Owner, Name string }

func (r Repo) String() string { return r.Host + "/" + r.Owner + "/" + r.Name }

// Backend writes to one kind of forge. A third forge is a new Backend and one
// Kind, with no change to the verbs.
type Backend interface {
	// CommentIssue posts body on issue or pull request number and returns the
	// comment's URL. A pull request is an issue on both supported forges.
	CommentIssue(ctx context.Context, r Repo, number int, body string) (string, error)
}

// Kind is a forge family.
type Kind string

const (
	KindGitHub Kind = "github"
	KindGitea  Kind = "gitea"
)

// Host declares one non-GitHub forge: its kind, the registry secret that
// holds its write token, and optionally an API base when it is not
// https://<host>.
type Host struct {
	Kind   Kind   `json:"kind"`
	Secret string `json:"secret"`
	API    string `json:"api,omitempty"`
}

// Hosts is forge/hosts.json: the forges dotf may write to besides github.com.
type Hosts map[string]Host

// LoadHosts reads the declaration. A missing file is an empty declaration, so
// GitHub keeps working before anyone declares a second forge.
func LoadHosts(path string) (Hosts, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Hosts{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc struct {
		Hosts Hosts `json:"hosts"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	for h, c := range doc.Hosts {
		if c.Kind != KindGitea || c.Secret == "" {
			return nil, fmt.Errorf("%s: host %q needs kind %q and a secret id", path, h, KindGitea)
		}
	}
	return doc.Hosts, nil
}

// ParseOrigin reads owner/name and host out of an origin URL in any of the
// three shapes git accepts: https, ssh:// and scp-like git@host:owner/name.
func ParseOrigin(raw string) (Repo, error) {
	u := strings.TrimSpace(raw)
	var host, path string
	switch {
	case strings.Contains(u, "://"):
		rest := u[strings.Index(u, "://")+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		host, path, _ = strings.Cut(rest, "/")
	case strings.Contains(u, "@") && strings.Contains(u, ":"):
		rest := u[strings.LastIndex(u, "@")+1:]
		host, path, _ = strings.Cut(rest, ":")
	default:
		return Repo{}, originError(raw)
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i] // drop a port
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	owner, name, ok := strings.Cut(path, "/")
	if host == "" || !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return Repo{}, originError(raw)
	}
	return Repo{Host: strings.ToLower(host), Owner: owner, Name: name}, nil
}

func originError(raw string) error {
	return fmt.Errorf("ERROR: cannot read a repository from origin %q. WHY: it is not an https, ssh or git@host:owner/name URL. FIX: set origin to the forge URL (git remote set-url origin <url>)", raw)
}

// Select picks the forge kind for a repository. Anything not declared is an
// error with the fix in it, never a guess: writing to the wrong forge with a
// token meant for another is the failure this refuses to risk.
func Select(r Repo, hosts Hosts) (Kind, Host, error) {
	if r.Host == "github.com" {
		return KindGitHub, Host{Kind: KindGitHub}, nil
	}
	if h, ok := hosts[r.Host]; ok {
		return h.Kind, h, nil
	}
	return "", Host{}, fmt.Errorf("ERROR: no write backend for %s. WHY: only github.com and the hosts declared in forge/hosts.json are written to. FIX: add %q to forge/hosts.json with its kind and registry secret", r, r.Host)
}

// redact removes a secret from text before it leaves the package, so no error
// path can carry the token into a log or a terminal.
func redact(s, secret string) string {
	if secret == "" {
		return s
	}
	return strings.ReplaceAll(s, secret, "[redacted]")
}
