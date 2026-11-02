package prreview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
)

// A PR nothing can review is declared, because proceeding unreviewed is allowed
// and proceeding silently is not. The declaration is the two halves the
// review-attestation gate requires, the label and a non-empty section, and they
// come from the registry the gate reads, so the two cannot disagree. A PR that
// started that way and later gains a reviewable file would otherwise carry a real
// review and a stale label for good, so the label is cleared then. The section
// stays for a human: editing prose by script is a sharper edge than this needs.

// ErrNotDeclared means the PR is not as this run meant to leave it. Why has
// already been written to DeclareEnv.Out as an ::error:: line.
var ErrNotDeclared = errors.New("pr declare-unreviewed: the PR does not carry the declaration")

const registryPath = "harness/review-attestation.json"

// rationale takes the branch the ignore list was read from. It tells the reader
// what works, so it names no command: a review asked for on this commit reaches
// the same filter. It holds no issue number either, which would link to something
// else in every other repository.
const rationale = "PR-Agent (github-actions) published no review, and this section exists so that\n" +
	"fact is declared rather than implied. Proceeding unreviewed is allowed here;\n" +
	"proceeding silently is not.\n\n" +
	"Every changed file in this PR is in `.pr_agent.toml`'s `[ignore]` list, read from\n" +
	"`%s`, or of a type PR-Agent never reads, so the diff left after its own filters\n" +
	"is empty. That is a property of the diff, not a quota or saturation failure, and\n" +
	"asking for another review of this commit hits the same empty diff.\n\n" +
	"A human verifies this change by reading it. For encrypted files, decrypt both\n" +
	"sides and diff the key names, never the values.\n"

// DeclareInputs are Repo (GITHUB_REPOSITORY), PR (PR_NUMBER), DefaultBranch
// (DEFAULT_BRANCH), where the registry is read, since a PR could otherwise choose
// its own escape, and Diff, one of the Diff constants from Reviewable.
type DeclareInputs struct {
	Repo, PR, DefaultBranch, Diff string
}

// DeclareEnv is what the declaration reaches outside itself through.
type DeclareEnv struct {
	Out io.Writer // workflow annotations
	GH  GHRunner  // nil: the gh on PATH
}

// DeclareUnreviewed puts the escape on a PR whose diff is empty, takes its label
// off one that is reviewable, and leaves one it knows nothing about alone.
func DeclareUnreviewed(ctx context.Context, in DeclareInputs, env DeclareEnv) error {
	if env.GH == nil {
		env.GH = execGH
	}
	d := declaration{in: in, env: env, issue: "repos/" + in.Repo + "/issues/" + in.PR}
	if in.Diff != DiffEmpty && in.Diff != DiffReviewable {
		d.say("nothing is known about PR #%s's diff; declaring nothing and clearing nothing.", in.PR)
		return nil
	}
	label, section, err := d.readEscape(ctx)
	if err != nil {
		return err
	}
	if in.Diff == DiffEmpty {
		return d.declare(ctx, label, section)
	}
	return d.clear(ctx, label)
}

// declaration is one run, with what it reads from and writes to the forge.
type declaration struct {
	in    DeclareInputs
	env   DeclareEnv
	issue string
}

func (d declaration) say(format string, a ...any) {
	_, _ = fmt.Fprintf(d.env.Out, format+"\n", a...)
}

// fail reports why and returns ErrNotDeclared.
func (d declaration) fail(format string, a ...any) error {
	d.say("::error::"+format, a...)
	return ErrNotDeclared
}

// gh runs one call. Its error carries gh's reason.
func (d declaration) gh(ctx context.Context, args ...string) ([]byte, error) {
	out, stderr, err := d.env.GH(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.ReplaceAll(strings.TrimSpace(string(stderr)), "\n", " "))
	}
	return out, nil
}

// readEscape is the registry's label and section at the default branch.
func (d declaration) readEscape(ctx context.Context) (label, section string, err error) {
	where := registryPath + " at " + d.in.DefaultBranch
	raw, err := fetchContent(ctx, d.env.GH, d.in.Repo, registryPath, d.in.DefaultBranch)
	if errors.Is(err, errContentAbsent) {
		return "", "", d.fail("%s does not exist, so it declares no escape.label/escape.section", where)
	}
	if err != nil {
		return "", "", d.fail("cannot read %s: %v", where, err)
	}
	var reg struct {
		Escape *struct {
			Label   string `json:"label"`
			Section string `json:"section"`
		} `json:"escape"`
	}
	if err := json.Unmarshal(raw, &reg); err != nil {
		return "", "", d.fail("cannot parse %s: %v", where, err)
	}
	if reg.Escape == nil || strings.TrimSpace(reg.Escape.Label) == "" || strings.TrimSpace(reg.Escape.Section) == "" {
		return "", "", d.fail("%s declares no escape.label/escape.section", where)
	}
	return reg.Escape.Label, reg.Escape.Section, nil
}

// declare writes the body first and the label last, as the gate needs both and
// the read-back below reports on the label.
func (d declaration) declare(ctx context.Context, label, section string) error {
	body, err := d.readBody(ctx)
	if err != nil {
		// Never treated as an empty body: the write below would replace the PR's.
		return d.fail("cannot read PR #%s's body: %v", d.in.PR, err)
	}
	text := fmt.Sprintf(rationale, d.in.DefaultBranch)
	if next, changed := withSection(body, section, text); changed {
		if err := d.writeBody(ctx, next); err != nil {
			return d.fail("cannot write PR #%s's body: %v", d.in.PR, err)
		}
		d.say("rationale written to PR #%s", d.in.PR)
	}
	// An additive POST: a PUT would drop every other label on the PR.
	if _, err := d.gh(ctx, "api", "-X", "POST", d.issue+"/labels", "-f", "labels[]="+label); err != nil {
		return d.fail("cannot add %q to PR #%s: %v", label, d.in.PR, err)
	}
	// A zero exit is not evidence: a half-written declaration leaves this step
	// green and the gate red for as long as the PR lives.
	have, err := d.labels(ctx)
	if err != nil {
		return d.fail("cannot read PR #%s's labels back: %v", d.in.PR, err)
	}
	if !slices.Contains(have, label) {
		return d.fail("%q is absent from PR #%s after the write, so the gate reads it as undeclared; it carries %q",
			label, d.in.PR, have)
	}
	d.say("declared: %q is on PR #%s", label, d.in.PR)
	return nil
}

// readBody is the PR's body, "" for the API's null.
func (d declaration) readBody(ctx context.Context) (string, error) {
	out, err := d.gh(ctx, "api", "repos/"+d.in.Repo+"/pulls/"+d.in.PR)
	if err != nil {
		return "", err
	}
	var pr struct {
		Body *string `json:"body"`
	}
	if err := json.Unmarshal(out, &pr); err != nil {
		return "", err
	}
	if pr.Body == nil {
		return "", nil
	}
	return *pr.Body, nil
}

func (d declaration) clear(ctx context.Context, label string) error {
	have, err := d.labels(ctx)
	if err != nil {
		return d.fail("cannot read PR #%s's labels: %v", d.in.PR, err)
	}
	if !slices.Contains(have, label) {
		return nil
	}
	d.say("::warning::removing stale %q label: this PR is reviewable now.", label)
	d.say("::warning::the rationale section in the PR body is left for a human to clean up.")
	// A 404 is the label already gone, which is what this came to do.
	_, err = d.gh(ctx, "api", "-X", "DELETE", d.issue+"/labels/"+url.PathEscape(label))
	if err != nil && !strings.Contains(err.Error(), "HTTP 404") {
		return d.fail("cannot remove %q from PR #%s: %v", label, d.in.PR, err)
	}
	return nil
}

// labels is every label on the PR. --paginate prints one array per page, and a
// PR past the first page would read as not carrying its own label.
func (d declaration) labels(ctx context.Context) ([]string, error) {
	out, err := d.gh(ctx, "api", d.issue+"/labels", "--paginate")
	if err != nil {
		return nil, err
	}
	var names []string
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for {
		var page []struct {
			Name string `json:"name"`
		}
		if err := dec.Decode(&page); errors.Is(err, io.EOF) {
			return names, nil
		} else if err != nil {
			return nil, err
		}
		for _, l := range page {
			names = append(names, l.Name)
		}
	}
}

// writeBody goes through a file, as the seam has no stdin: a body on argv is
// limited by the OS and visible to every process on the box.
func (d declaration) writeBody(ctx context.Context, body string) (err error) {
	data, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp("", "pr-body-*.json")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.Remove(f.Name())) }()
	_, err = f.Write(data)
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	_, err = d.gh(ctx, "api", "-X", "PATCH", "repos/"+d.in.Repo+"/pulls/"+d.in.PR, "--input", f.Name())
	return err
}

// withSection is body carrying section with text under it, and whether that took
// a change. It reads the body as check-review-attestation.sh does: \r and trailing
// blanks do not count, since GitHub's web editor saves CRLF, the first matching
// line is the section, and it ends at the next `## ` heading. Otherwise a body the
// gate reads as declared would gain a second copy, and a heading the gate reads as
// empty would stay that way with the text below a later one.
func withSection(body, section, text string) (string, bool) {
	lines := strings.Split(body, "\n")
	bare := func(line string) string { return strings.TrimRight(line, " \t\r") }
	at := slices.IndexFunc(lines, func(line string) bool { return bare(line) == section })
	if at < 0 {
		if body == "" {
			return section + "\n\n" + text, true
		}
		return body + "\n\n" + section + "\n\n" + text, true
	}
	for _, line := range lines[at+1:] {
		if strings.HasPrefix(bare(line), "## ") {
			break
		}
		if bare(line) != "" {
			return body, false
		}
	}
	filled := slices.Concat(lines[:at+1], []string{"", text}, lines[at+1:])
	return strings.Join(filled, "\n"), true
}
