package lane

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// PRInfo is the newest pull request of a branch, as the forge reports it.
type PRInfo struct {
	Number int
	State  PRState
}

// PRSource answers for one branch. It is asked once per branch per run: a
// fleet of N worktrees must cost N requests, not N times the rows' questions.
type PRSource interface {
	PRFor(branch string) (PRInfo, error)
}

// Finder is the part of a session backend status reads: whether herdr knows a
// branch's worktree. Backend satisfies it.
type Finder interface {
	Find(repo, branch string) (Worktree, bool, error)
}

// Herdr readings of a row. Absent means the server could not be asked, which
// is a supported state and never an error.
const (
	HerdrAbsent    = "absent"
	HerdrNone      = "none"
	HerdrListed    = "listed"
	HerdrWorkspace = "workspace"
)

// Sources is everything Status reads, each behind a seam so a test answers
// from a fixture and production from git, gh, the process table and herdr.
type Sources struct {
	Now  time.Time
	Repo string // the repository's main checkout, herdr's --cwd

	List  func() ([]worktree.Info, error)
	Forge PRSource
	// Live reports whether someone is working in a worktree: a process inside
	// it, or a recent session. An error reads as live (see Status).
	Live  func(path string) (bool, error)
	Herdr Finder

	// The queue, for the lanes that have no worktree yet or no longer.
	Queue        []int
	Issues       IssueSource
	BranchPrefix string
}

// Row is one lane in the status table.
type Row struct {
	Ticket  int    `json:"ticket,omitempty"`
	Branch  string `json:"branch"`
	Path    string `json:"path,omitempty"`
	PR      int    `json:"pr,omitempty"`
	PRState string `json:"pr_state,omitempty"`
	Agent   string `json:"agent,omitempty"`
	Lease   string `json:"lease,omitempty"`
	Live    bool   `json:"live"`
	Herdr   string `json:"herdr"`
	Dirty   bool   `json:"dirty,omitempty"`
	Phase   Phase  `json:"phase"`
}

// Status is the read-only join of git, the forge, liveness and herdr. It
// changes nothing and stores nothing: a second call recomputes everything.
func Status(src Sources) ([]Row, error) {
	infos, err := src.List()
	if err != nil {
		return nil, fmt.Errorf("lane status: worktrees: %w", err)
	}
	forge := CachedForge(src.Forge)
	var lanes []lane
	onDisk := map[string]bool{}
	for _, info := range infos {
		if info.IsMain {
			continue
		}
		onDisk[info.Branch] = true
		l, err := src.onDiskLane(info, forge)
		if err != nil {
			return nil, err
		}
		lanes = append(lanes, l)
	}
	queued, err := src.queuedLanes(forge, onDisk)
	if err != nil {
		return nil, err
	}
	lanes = append(lanes, queued...)
	markHead(lanes)
	var rows []Row
	for _, l := range lanes {
		l.row.Phase = Derive(l.facts, src.Now)
		if l.row.Phase != PhaseNone {
			rows = append(rows, l.row)
		}
	}
	return rows, nil
}

type lane struct {
	row   Row
	facts Facts
}

func (s Sources) onDiskLane(info worktree.Info, forge PRSource) (lane, error) {
	pr, err := forge.PRFor(info.Branch)
	if err != nil {
		return lane{}, fmt.Errorf("lane status: forge: %w", err)
	}
	live, err := s.Live(info.Path)
	if err != nil {
		live = true // unsure reads as alive: a false "dead" reaps a live session
	}
	herdr := s.herdr(info.Branch)
	f := Facts{Info: info, Exists: true, PR: pr.State, Live: live, HerdrPresent: herdr != HerdrAbsent}
	r := Row{Branch: info.Branch, Path: info.Path, PR: pr.Number, PRState: string(pr.State), Live: live, Herdr: herdr, Dirty: info.Dirty, Ticket: ticketOf(info)}
	if m := info.Metadata; m != nil {
		r.Agent, r.Lease = m.Creator, m.LeaseExpiresAt.Format(time.RFC3339)
	}
	return lane{row: r, facts: f}, nil
}

// queuedLanes are the queue's issues that no worktree on disk carries: not
// started yet (Pending) or finished and reaped (Reaped).
func (s Sources) queuedLanes(forge PRSource, onDisk map[string]bool) ([]lane, error) {
	var out []lane
	for _, n := range s.Queue {
		is, err := s.Issues.Issue(n)
		if err != nil {
			return nil, fmt.Errorf("lane status: issue #%d: %w", n, err)
		}
		branch := s.BranchPrefix + "/" + Slug(is.Title)
		if onDisk[branch] {
			continue
		}
		pr, err := forge.PRFor(branch)
		if err != nil {
			return nil, fmt.Errorf("lane status: forge: %w", err)
		}
		herdr := s.herdr(branch)
		f := Facts{Queued: is.State == "OPEN" && len(is.Assignees) == 0, PR: pr.State, HerdrPresent: herdr != HerdrAbsent}
		out = append(out, lane{row: Row{Ticket: n, Branch: branch, PR: pr.Number, PRState: string(pr.State), Herdr: herdr}, facts: f})
	}
	return out, nil
}

// herdr reads one branch's place in herdr; an error is "absent", not a failure.
func (s Sources) herdr(branch string) string {
	if s.Herdr == nil {
		return HerdrAbsent
	}
	w, found, err := s.Herdr.Find(s.Repo, branch)
	switch {
	case err != nil:
		return HerdrAbsent
	case !found:
		return HerdrNone
	case w.Open:
		return HerdrWorkspace
	}
	return HerdrListed
}

// markHead flags the open PR first in the landing order, the lowest number.
func markHead(lanes []lane) {
	var open []int
	for i, l := range lanes {
		if l.facts.PR == PROpen && l.facts.Exists {
			open = append(open, i)
		}
	}
	if len(open) == 0 {
		return
	}
	sort.Slice(open, func(a, b int) bool { return lanes[open[a]].row.PR < lanes[open[b]].row.PR })
	lanes[open[0]].facts.Head = true
}

// ticketOf is the issue a worktree is for: the lease's, else the number a
// `<type>/<issue>-<slug>` branch leads with.
func ticketOf(info worktree.Info) int {
	if info.Metadata != nil && info.Metadata.Issue != 0 {
		return info.Metadata.Issue
	}
	_, tail, ok := strings.Cut(info.Branch, "/")
	if !ok {
		return 0
	}
	digits := tail[:len(tail)-len(strings.TrimLeft(tail, "0123456789"))]
	n, _ := strconv.Atoi(digits)
	return n
}

// CachedForge asks src once per branch and remembers the answer. Status wraps
// its source in it, and a caller that also feeds the same source to ForgeGit
// wraps it first, so a branch costs one request however many readers ask.
func CachedForge(src PRSource) PRSource {
	if c, ok := src.(*onceForge); ok {
		return c
	}
	return &onceForge{src: src}
}

type onceForge struct {
	src  PRSource
	seen map[string]PRInfo
}

func (o *onceForge) PRFor(branch string) (PRInfo, error) {
	if pr, ok := o.seen[branch]; ok {
		return pr, nil
	}
	pr, err := o.src.PRFor(branch)
	if err != nil {
		return PRInfo{}, err
	}
	if o.seen == nil {
		o.seen = map[string]PRInfo{}
	}
	o.seen[branch] = pr
	return pr, nil
}
