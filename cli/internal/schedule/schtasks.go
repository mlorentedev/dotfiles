package schedule

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"
)

// Logon types a task's principal runs under. S4U runs it in session 0 whether
// or not the user is logged on, so no console window appears, but registering
// one needs an elevated caller. InteractiveToken is what a standard user can
// register: it runs in the desktop session (Register-HiveScheduledTask, ADR-015).
const (
	LogonS4U         = "S4U"
	LogonInteractive = "InteractiveToken"
)

var (
	digestRe = regexp.MustCompile(`\[dotf:([0-9a-f]{12})\]`)
	logonRe  = regexp.MustCompile(`<LogonType>([A-Za-z0-9]+)</LogonType>`)
)

// TaskDigest names the rendered job in the task's Description. schtasks hands a
// registered task back normalised, so its XML is never compared with ours; the
// digest it carries is. The principal is left out on purpose: the logon type
// depends on who registers the task, not on the job.
func TaskDigest(job Job) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\x00%v", job.Description, job.Program, strings.Join(job.Args, "\x00"), job.Daily)
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// TaskXML renders job as Task Scheduler XML for user under logon. Unset
// settings take the schema's defaults, as New-ScheduledTaskSettingsSet does;
// StartWhenAvailable runs a slot missed while the machine was off.
func TaskXML(job Job, user, logon string) string {
	start := "2026-01-01T03:00:00"
	if c := job.Daily; c != nil {
		start = fmt.Sprintf("2026-01-01T%02d:%02d:00", c.Hour, c.Minute)
	}
	args := make([]string, len(job.Args))
	for i, a := range job.Args {
		if strings.ContainsAny(a, " \t\"") {
			a = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
		}
		args[i] = a
	}
	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>` + xmlEscape(job.Description+" [dotf:"+TaskDigest(job)+"]") + `</Description>
  </RegistrationInfo>
  <Triggers>
    <CalendarTrigger>
      <StartBoundary>` + start + `</StartBoundary>
      <Enabled>true</Enabled>
      <ScheduleByDay>
        <DaysInterval>1</DaysInterval>
      </ScheduleByDay>
    </CalendarTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>` + xmlEscape(user) + `</UserId>
      <LogonType>` + logon + `</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <StartWhenAvailable>true</StartWhenAvailable>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>` + xmlEscape(job.Program) + `</Command>
      <Arguments>` + xmlEscape(strings.Join(args, " ")) + `</Arguments>
    </Exec>
  </Actions>
</Task>
`
}

// Schtasks is the windows backend: a task registered through schtasks.exe from
// rendered XML.
type Schtasks struct {
	Run      Runner
	User     string
	Elevated func() bool
	TempDir  string
}

func (s Schtasks) logon() string {
	if s.Elevated != nil && s.Elevated() {
		return LogonS4U
	}
	return LogonInteractive
}

type taskState struct {
	present, managed, stale, upgrade bool
}

// read queries the task. `/Query /TN` exits non-zero both for a task that
// does not exist and for a query it refused, and only its localised stderr
// tells them apart, so absence is read from the task list instead: a list
// that does not name the task is an absent task, and any failure is an error,
// never "nothing to remove".
func (s Schtasks) read(job Job) (taskState, error) {
	list, err := s.Run("/Query", "/FO", "CSV", "/NH")
	if err != nil {
		return taskState{}, fmt.Errorf("schtasks /Query: %w", err)
	}
	if !listsTask(decodeText([]byte(list)), job.Names.Task) {
		return taskState{}, nil
	}
	out, err := s.Run("/Query", "/TN", job.Names.Task, "/XML")
	if err != nil {
		return taskState{}, fmt.Errorf("schtasks /Query /TN %s: %w", job.Names.Task, err)
	}
	text := decodeText([]byte(out))
	st := taskState{present: true}
	m := digestRe.FindStringSubmatch(text)
	st.managed = m != nil
	st.stale = m == nil || m[1] != TaskDigest(job)
	// An elevated caller upgrades a task a standard user registered; a
	// standard user never downgrades an S4U one.
	if l := logonRe.FindStringSubmatch(text); s.logon() == LogonS4U && (l == nil || l[1] != LogonS4U) {
		st.upgrade = true
	}
	return st, nil
}

// listsTask reports whether a `/Query /FO CSV /NH` listing names the task in
// the root folder. The first field is the task's path, which is not localised.
func listsTask(list, name string) bool {
	for _, line := range strings.Split(list, "\n") {
		if first, _, _ := strings.Cut(strings.TrimSpace(line), ","); first == `"\`+name+`"` {
			return true
		}
	}
	return false
}

func (s Schtasks) Inspect(job Job) (State, error) {
	st, err := s.read(job)
	if err != nil {
		return State{}, err
	}
	out := State{Present: st.present, Managed: st.managed, Where: `task \` + job.Names.Task}
	switch {
	case !st.present || st.stale:
		out.Pending = []string{"register task " + job.Names.Task + " (" + s.logon() + ")"}
	case st.upgrade:
		out.Pending = []string{"re-register task " + job.Names.Task + " as " + LogonS4U}
	}
	return out, nil
}

// Install registers the task from XML staged in TempDir. schtasks reads task
// XML as UTF-16, so the file is written in it, with a byte-order mark.
func (s Schtasks) Install(job Job) error {
	st, err := s.read(job)
	if err != nil {
		return err
	}
	if st.present && !st.stale && !st.upgrade {
		return nil
	}
	f, err := os.CreateTemp(s.TempDir, "dotf-task-*.xml")
	if err != nil {
		return err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	if _, err := f.Write(utf16LE(TaskXML(job, s.User, s.logon()))); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, err := s.Run("/Create", "/TN", job.Names.Task, "/XML", filepath.Clean(path), "/F"); err != nil {
		return fmt.Errorf("schtasks /Create %s: %w", job.Names.Task, err)
	}
	return nil
}

// Remove deletes the task.
func (s Schtasks) Remove(job Job) error {
	st, err := s.read(job)
	if err != nil || !st.present {
		return err
	}
	if _, err := s.Run("/Delete", "/TN", job.Names.Task, "/F"); err != nil {
		return fmt.Errorf("schtasks /Delete %s: %w", job.Names.Task, err)
	}
	return nil
}

func utf16LE(s string) []byte {
	units := utf16.Encode([]rune(s))
	b := make([]byte, 2, 2+2*len(units))
	b[0], b[1] = 0xFF, 0xFE
	for _, u := range units {
		b = append(b, byte(u), byte(u>>8))
	}
	return b
}

// decodeText reads schtasks output, which arrives as UTF-16LE when it starts
// with that byte-order mark and as the console's code page otherwise. The
// parts read from it (the digest, the logon type) are ASCII in either.
func decodeText(b []byte) string {
	if len(b) < 2 || b[0] != 0xFF || b[1] != 0xFE {
		return string(b)
	}
	b = b[2:]
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		units = append(units, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(units))
}
