package harness

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Shell names the syntax family a command string is written in.
type Shell int

const (
	// ShellPOSIX is bash, zsh and sh.
	ShellPOSIX Shell = iota
	// ShellPowerShell is Claude's PowerShell tool.
	ShellPowerShell
)

// PushEnv answers the questions about the working tree that a command string
// cannot: which branch is checked out and which branch the remote treats as
// its default. Real answers come from git (cmd layer); tests inject fakes.
type PushEnv struct {
	// Dir is the working directory the command starts in.
	Dir string
	// CurrentBranch returns the checked-out branch of dir, "" when detached or unknown.
	CurrentBranch func(dir string) string
	// DefaultBranch returns the default branch of remote as seen from dir, "" when unresolved.
	DefaultBranch func(dir, remote string) string
}

// PushVerdict is the guard's answer for one shell command.
type PushVerdict struct {
	Refused bool
	Reason  string
}

func refuse(format string, a ...any) PushVerdict {
	return PushVerdict{Refused: true, Reason: fmt.Sprintf(format, a...)}
}

// maxShellDepth bounds `bash -c '...'` and `eval` recursion.
const maxShellDepth = 4

// GuardCommand decides whether a shell command may run, for the one question
// this guard owns: a git push that forces (HARNESS-201).
//
// THE PARSER IS ASYMMETRIC ON PURPOSE. A push is force-like when it carries
// `--force`, `-f`, any `--force-with-lease` / `--force-if-includes` spelling,
// `--mirror`, or a `+<refspec>`; everything else passes untouched, exactly as it
// did before this guard existed. A force-like push is allowed in ONE fully
// literal shape (see checkLeasePush), and anything force-like the parser cannot
// read to the end is refused. A parser bug therefore fails toward refusal of a
// force, never toward allowing one, and never toward blocking a normal push.
//
// It reads the command text only. Aliases, git config on disk and scripts that
// run git internally are outside what a string can show.
func GuardCommand(command string, shell Shell, env PushEnv) PushVerdict {
	g := &guard{shell: shell, env: env, dir: normaliseDir(env.Dir), dirKnown: env.Dir != "", memo: map[string]string{}}
	return g.run(command, 0)
}

type guard struct {
	shell    Shell
	env      PushEnv
	dir      string
	dirKnown bool
	// memo holds resolver answers for one command: each question costs a git
	// process, and the hook is bounded at a few seconds.
	memo map[string]string
}

func (g *guard) currentBranch(dir string) string {
	k := "cur\x00" + dir
	if v, ok := g.memo[k]; ok {
		return v
	}
	v := g.env.CurrentBranch(dir)
	g.memo[k] = v
	return v
}

func (g *guard) defaultBranch(dir, remote string) string {
	k := "def\x00" + dir + "\x00" + remote
	if v, ok := g.memo[k]; ok {
		return v
	}
	v := g.env.DefaultBranch(dir, remote)
	g.memo[k] = v
	return v
}

func (g *guard) run(s string, depth int) PushVerdict {
	if depth > maxShellDepth {
		return refuse("shell nesting deeper than %d levels cannot be checked for a force push", maxShellDepth)
	}
	cmds, ok := splitCommands(s, g.shell)
	if !ok && ((strings.Contains(s, "git") && strings.Contains(s, "push")) || strings.Contains(s, "land:push")) {
		return refuse("cannot parse this command (open quote or substitution) and it mentions a push; refusing in case it forces")
	}
	for _, c := range cmds {
		if v := g.simple(stripRedirects(c), depth); v.Refused {
			return v
		}
	}
	return PushVerdict{}
}

var envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// prefixWords run another command and change nothing the guard reads.
var prefixWords = map[string]bool{
	"!": true, "if": true, "then": true, "do": true, "else": true, "elif": true,
	"while": true, "until": true, "time": true, "builtin": true,
	"nohup": true, "{": true, "&": true,
}

// simple examines one simple command.
func (g *guard) simple(cmd command, depth int) PushVerdict {
	envDir := false
	for len(cmd) > 0 {
		t := cmd[0].text
		switch {
		case envAssign.MatchString(t):
			envDir = envDir || movesRepo(t)
			cmd = cmd[1:]
		case prefixWords[t]:
			cmd = cmd[1:]
		case wrappers[programName(t)]:
			var dir bool
			cmd, dir = skipOptions(cmd[1:])
			envDir = envDir || dir
		default:
			return g.byProgram(cmd, envDir, depth)
		}
	}
	if envDir {
		g.dirKnown = false // `GIT_DIR=x` on its own line sticks to the shell
	}
	return PushVerdict{}
}

// wrappers run the command that follows them and take options of their own.
var wrappers = map[string]bool{"env": true, "sudo": true, "command": true, "exec": true}

// wrapped is a program the guard reads; an option's argument is never one of
// these in practice, which is what tells `sudo -u root git` from `sudo -E git`.
var wrapped = map[string]bool{
	"git": true, "bash": true, "sh": true, "zsh": true, "dash": true, "ksh": true,
	"pwsh": true, "powershell": true, "eval": true, "cd": true, "npm": true,
	"pnpm": true, "yarn": true, "bun": true,
}

// skipOptions drops what a wrapper takes before its command: `-x` options, the
// word after one (`-u root`, `-a name`), and `VAR=v` assignments. It reports
// whether an assignment moved the repository (GIT_DIR and friends).
func skipOptions(cmd command) (command, bool) {
	envDir, afterOpt := false, false
	for len(cmd) > 0 {
		t := cmd[0].text
		switch {
		case strings.HasPrefix(t, "-"):
			afterOpt = true
			envDir = envDir || t == "-C" || strings.HasPrefix(t, "--chdir")
		case envAssign.MatchString(t):
			envDir = envDir || movesRepo(t)
			afterOpt = false
		case afterOpt && !wrapped[programName(t)]:
			afterOpt = false
		default:
			return cmd, envDir
		}
		cmd = cmd[1:]
	}
	return cmd, envDir
}

func movesRepo(assign string) bool {
	switch strings.ToUpper(assign[:strings.IndexByte(assign, '=')]) {
	case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR":
		return true
	}
	return false
}

func programName(t string) string {
	t = strings.ReplaceAll(t, `\`, "/")
	t = t[strings.LastIndexByte(t, '/')+1:]
	return strings.TrimSuffix(strings.ToLower(t), ".exe")
}

func (g *guard) byProgram(cmd command, envDir bool, depth int) PushVerdict {
	args := cmd[1:]
	if cmd[0].dynamic && hasText(args, "push") {
		return refuse("the program name is a variable and the arguments push; cannot check it for a force")
	}
	if envDir {
		// The repository is not the working directory: nothing a nested shell
		// reads from the directory can be trusted either.
		saved := g.dirKnown
		g.dirKnown = false
		defer func() { g.dirKnown = saved }()
	}
	switch programName(cmd[0].text) {
	case "git":
		return g.git(args, envDir)
	case "cd", "pushd", "chdir", "set-location", "sl":
		g.cd(cmd)
	case "export", "declare", "typeset", "readonly":
		g.exported(args)
	case "bash", "sh", "zsh", "dash", "ksh":
		return g.shellString(args, depth, ShellPOSIX)
	case "pwsh", "powershell":
		return g.shellString(args, depth, ShellPowerShell)
	case "eval":
		for len(args) > 0 && args[0].text == "--" {
			args = args[1:]
		}
		return g.nested(joinTexts(args), depth, g.shell)
	case "npm", "pnpm", "yarn", "bun":
		if hasText(args, "land:push") {
			return refuse("land:push is the owner's landing mode; it is not run by an agent")
		}
	}
	return PushVerdict{}
}

// cd tracks a directory change, except inside a subshell or a substitution,
// where it does not carry out to the commands after it.
func (g *guard) cd(cmd command) {
	if cmd[0].sub {
		g.dirKnown = false
		return
	}
	g.chdir(cmd[1:])
}

// exported notes `export GIT_DIR=...` and friends, which move the repository for
// every later command of the shell.
func (g *guard) exported(args command) {
	for _, a := range args {
		if strings.Contains(a.text, "=") && movesRepo(a.text) {
			g.dirKnown = false
		}
	}
}

func hasText(args command, want string) bool {
	for _, a := range args {
		if a.text == want {
			return true
		}
	}
	return false
}

func joinTexts(args command) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.text
	}
	return strings.Join(parts, " ")
}

// shellString finds the `-c <string>` of a nested shell and checks the string.
func (g *guard) shellString(args command, depth int, sh Shell) PushVerdict {
	for i, a := range args {
		isFlag := strings.HasPrefix(a.text, "-") && !strings.HasPrefix(a.text, "--")
		isCmd := strings.EqualFold(a.text, "-command") || (isFlag && strings.ContainsRune(a.text, 'c'))
		if !isCmd {
			continue
		}
		j := i + 1
		for j < len(args) && args[j].text == "--" {
			j++
		}
		if j < len(args) && sh == ShellPowerShell {
			// PowerShell takes every word after -Command as the script.
			return g.nested(joinTexts(args[j:]), depth, sh)
		}
		if j < len(args) {
			return g.nested(args[j].text, depth, sh)
		}
	}
	return PushVerdict{}
}

func (g *guard) nested(s string, depth int, sh Shell) PushVerdict {
	inner := *g
	inner.shell = sh
	v := inner.run(s, depth+1)
	g.dir, g.dirKnown = inner.dir, inner.dirKnown
	return v
}

// --- directories ------------------------------------------------------------

var driveRaw = regexp.MustCompile(`^[A-Za-z]:\\`)

// dirText reads a directory argument: raw when it is a Windows drive path whose
// backslashes an unquoted POSIX read would have eaten.
func dirText(w word) string {
	if driveRaw.MatchString(w.raw) {
		return w.raw
	}
	return w.text
}

func normaliseDir(p string) string {
	if p == "" {
		return ""
	}
	return path.Clean(strings.ReplaceAll(p, `\`, "/"))
}

func isAbsDir(p string) bool {
	return strings.HasPrefix(p, "/") || (len(p) >= 2 && p[1] == ':')
}

// joinDir moves base by p the way a shell would.
func joinDir(base, p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	if isAbsDir(p) {
		return path.Clean(p)
	}
	return path.Join(base, p)
}

// chdir tracks `cd` so the branch is read from where the push will run. A target
// that cannot be known (`cd $X`, `cd ~`, `cd -`) makes the directory unknown,
// which refuses a lease push rather than guessing.
func (g *guard) chdir(args command) {
	for _, a := range args {
		if strings.HasPrefix(a.text, "-") && a.text != "-" {
			continue
		}
		if a.dynamic || a.text == "-" || strings.HasPrefix(a.text, "~") || !g.dirKnown {
			g.dirKnown = false
			return
		}
		g.dir = joinDir(g.dir, dirText(a))
		return
	}
	g.dirKnown = false
}

// --- git --------------------------------------------------------------------

// gitPrefix is what precedes the subcommand: the directory it runs in and
// whether `-c remote.*.push=+...` forces by configuration.
type gitPrefix struct {
	dir      string
	known    bool
	cfgForce bool
	rest     command
}

func (g *guard) git(args command, envDir bool) PushVerdict {
	p := g.gitGlobals(args, envDir)
	if len(p.rest) == 0 || p.rest[0].text != "push" {
		return PushVerdict{}
	}
	return g.push(p.rest[1:], p)
}

func (g *guard) gitGlobals(args command, envDir bool) gitPrefix {
	p := gitPrefix{dir: g.dir, known: g.dirKnown && !envDir}
	for i := 0; i < len(args); {
		n := p.option(args, i)
		if n == 0 {
			p.rest = args[i:]
			return p
		}
		i += n
	}
	return p
}

// option reads the git global option at args[i] and returns how many words it
// took; 0 means args[i] is the subcommand. Attached forms (`-C/dir`,
// `-ckey=v`) are as valid as separate ones.
func (p *gitPrefix) option(args command, i int) int {
	w := args[i]
	t := w.text
	switch {
	case t == "-C" && i+1 < len(args):
		p.setDir(args[i+1])
		return 2
	case t == "-c" && i+1 < len(args):
		p.cfgForce = p.cfgForce || forcingConfig(args[i+1].text)
		return 2
	case len(t) > 2 && (t[:2] == "-C" || t[:2] == "-c"):
		p.attached(w)
		return 1
	case strings.HasPrefix(t, "-"):
		return p.longOption(t)
	}
	return 0
}

// longOption reads a `-`-prefixed global that is not -C or -c.
func (p *gitPrefix) longOption(t string) int {
	switch {
	case strings.HasPrefix(t, "--git-dir") || strings.HasPrefix(t, "--work-tree"):
		p.known = false
		return 1 + btoi(!strings.Contains(t, "="))
	case t == "--namespace" || t == "--super-prefix" || t == "--config-env" || t == "--exec-path":
		return 2
	}
	return 1
}

func (p *gitPrefix) setDir(w word) {
	p.known = p.known && !w.dynamic
	p.dir = joinDir(p.dir, dirText(w))
}

// attached handles `-C<dir>` and `-c<key=value>` written without a space.
func (p *gitPrefix) attached(w word) {
	rest := w
	rest.text = w.text[2:]
	if len(w.raw) >= 2 {
		rest.raw = w.raw[2:]
	}
	if w.text[1] == 'C' {
		p.setDir(rest)
		return
	}
	p.cfgForce = p.cfgForce || forcingConfig(rest.text)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// forcingConfig reports a `-c remote.<name>.push=+...` setting, which makes a
// bare `git push` force without the command text saying so.
func forcingConfig(kv string) bool {
	k, v, _ := strings.Cut(kv, "=")
	k = strings.ToLower(k)
	return strings.HasPrefix(k, "remote.") && strings.HasSuffix(k, ".push") && strings.Contains(v, "+")
}

// --- push -------------------------------------------------------------------

// pushCall is a push's arguments, sorted into what matters to the guard.
type pushCall struct {
	force, fii, negate, mirror, plus, abbrev, dynamic bool
	leases                                            []string
	bareLeases                                        int
	deleteOpt                                         bool
	unknown                                           []string
	positional                                        []string
}

func (c *pushCall) forceLike(cfgForce bool) bool {
	return c.force || c.fii || c.mirror || c.plus || c.abbrev || cfgForce || c.bareLeases > 0 || len(c.leases) > 0
}

var forceNames = []string{"--force", "--force-with-lease", "--force-if-includes", "--mirror"}

// benignLong are push options that cannot change which refs move or how.
var benignLong = map[string]bool{
	"--set-upstream": true, "--quiet": true, "--verbose": true, "--dry-run": true,
	"--no-verify": true, "--verify": true, "--progress": true, "--no-progress": true,
	"--porcelain": true, "--atomic": true, "--follow-tags": true, "--no-follow-tags": true,
	"--thin": true, "--no-thin": true,
}

// abbreviatesForce reports a proper prefix of a force option. git accepts any
// unambiguous prefix of a long option, so `--force-w=b:sha` is a lease.
func abbreviatesForce(key string) bool {
	if len(key) < 3 {
		return false
	}
	for _, n := range forceNames {
		if key != n && strings.HasPrefix(n, key) {
			return true
		}
	}
	return false
}

func parsePush(args command) pushCall {
	var c pushCall
	afterDashes := false
	for _, a := range args {
		c.dynamic = c.dynamic || a.dynamic
		t := a.text
		switch {
		case afterDashes:
			c.addPositional(t)
		case t == "--":
			afterDashes = true
		case strings.HasPrefix(t, "--"):
			c.addLong(t)
		case strings.HasPrefix(t, "-") && len(t) > 1:
			c.addShort(t)
		default:
			c.addPositional(t)
		}
	}
	return c
}

func (c *pushCall) addPositional(t string) {
	c.plus = c.plus || strings.HasPrefix(t, "+")
	c.positional = append(c.positional, t)
}

func (c *pushCall) addLong(t string) {
	key, val, hasVal := strings.Cut(t, "=")
	switch key {
	case "--force":
		c.force = true
	case "--force-with-lease":
		if hasVal {
			c.leases = append(c.leases, val)
		} else {
			c.bareLeases++
		}
	case "--force-if-includes":
		c.fii = true
	case "--no-force-with-lease", "--no-force-if-includes":
		c.negate = true
	case "--mirror":
		c.mirror = true
	case "--delete":
		c.deleteOpt = true
	default:
		switch {
		case abbreviatesForce(key):
			c.abbrev = true
		case !benignLong[key]:
			c.unknown = append(c.unknown, t)
		}
	}
}

func (c *pushCall) addShort(t string) {
	for _, r := range t[1:] {
		switch r {
		case 'f':
			c.force = true
		case 'd':
			c.deleteOpt = true
		case 'u', 'q', 'v', 'n':
		default:
			c.unknown = append(c.unknown, t)
			return
		}
	}
}

func (g *guard) push(args command, p gitPrefix) PushVerdict {
	c := parsePush(args)
	if !c.forceLike(p.cfgForce) {
		return PushVerdict{}
	}
	if v := c.refuseForm(p.cfgForce); v.Refused {
		return v
	}
	return g.checkLeasePush(c, p)
}

// refuseForm refuses every force-like spelling that is not the one lease shape.
func (c *pushCall) refuseForm(cfgForce bool) PushVerdict {
	switch {
	case c.dynamic:
		return refuse("a variable or substitution inside a force-like push cannot be checked; write the branch and sha literally")
	case cfgForce:
		return refuse("a -c remote.*.push config with + forces every push; use --force-with-lease=<branch>:<sha>")
	case c.mirror:
		return refuse("--mirror overwrites every ref on the remote and is refused")
	case c.force:
		return refuse("--force/-f overwrites without checking the remote; use --force-with-lease=<branch>:<sha> on your own branch")
	case c.plus:
		return refuse("a +<refspec> forces the update; use --force-with-lease=<branch>:<sha> on your own branch")
	case c.abbrev:
		return refuse("an abbreviated force option is not accepted; spell out --force-with-lease=<branch>:<sha>")
	case c.negate:
		return refuse("a negated lease option beside a force-like push makes the outcome order-dependent; remove it")
	case len(c.leases)+c.bareLeases > 1:
		return refuse("exactly one lease is allowed per push")
	case c.bareLeases == 1:
		return refuse("a bare --force-with-lease names no branch and no sha; use --force-with-lease=<branch>:<sha>")
	case len(c.leases) == 0:
		return refuse("--force-if-includes is accepted only beside --force-with-lease=<branch>:<sha>; no lease given")
	case c.deleteOpt:
		return refuse("a deletion is not allowed through the lease path")
	case len(c.unknown) > 0:
		return refuse("option %s is not permitted on a lease push", c.unknown[0])
	}
	return PushVerdict{}
}

var shaPattern = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)

const refsHeads = "refs/heads/"

// checkLeasePush verifies the one allowed shape:
//
//	git push <remote> --force-with-lease=<branch>:<full sha> <refspec>
//
// where the refspec's destination, the lease branch and the checked-out branch
// are one name, and that name is not the remote's default branch.
func (g *guard) checkLeasePush(c pushCall, p gitPrefix) PushVerdict {
	leaseBranch, sha, hasSHA := strings.Cut(c.leases[0], ":")
	if !hasSHA || !shaPattern.MatchString(sha) {
		return refuse("the lease needs an explicit full 40 or 64 hex character sha: --force-with-lease=<branch>:<sha>")
	}
	leaseBranch = strings.TrimPrefix(leaseBranch, refsHeads)
	if len(c.positional) != 2 {
		return refuse("a force-with-lease push needs an explicit remote and exactly one refspec")
	}
	remote, spec := c.positional[0], c.positional[1]
	dst, v := destinationOf(spec)
	if v.Refused {
		return v
	}
	if !p.known {
		return refuse("the working directory cannot be resolved (variable, --git-dir or GIT_DIR); cannot read the checked-out branch")
	}
	if dst == "" { // `HEAD`: pushes to the checked-out branch's name
		dst = g.currentBranch(p.dir)
	}
	if dst != leaseBranch {
		return refuse("destination %q differs from the lease branch %q", dst, leaseBranch)
	}
	if v := g.refuseDefault(dst, remote, p.dir); v.Refused {
		return v
	}
	return g.refuseOtherBranch(spec, dst, p.dir)
}

// destinationOf returns the destination branch of a refspec, "" for a bare HEAD.
func destinationOf(spec string) (string, PushVerdict) {
	if strings.ContainsAny(spec, "*?[^~ \t\\") {
		return "", refuse("unsupported characters in the refspec %q; name one branch literally", spec)
	}
	src, dst, hasColon := strings.Cut(spec, ":")
	if !hasColon {
		if src == "HEAD" {
			return "", PushVerdict{}
		}
		dst = src
	}
	switch {
	case src == "":
		return "", refuse("a :<branch> refspec is a deletion, not allowed through the lease path")
	case dst == "" || dst == "HEAD":
		return "", refuse("the refspec %q has no usable destination branch", spec)
	case strings.HasPrefix(dst, "refs/") && !strings.HasPrefix(dst, refsHeads):
		return "", refuse("destination %q is not a branch", dst)
	}
	return strings.TrimPrefix(dst, refsHeads), PushVerdict{}
}

func (g *guard) refuseDefault(dst, remote, dir string) PushVerdict {
	if dst == "main" || dst == "master" {
		return refuse("%q is a default branch name; a forced update of the default branch is never allowed", dst)
	}
	if def := g.defaultBranch(dir, remote); def != "" && def == dst {
		return refuse("%q is the default branch of %q; a forced update of the default branch is never allowed", dst, remote)
	}
	return PushVerdict{}
}

// refuseOtherBranch requires the source to be HEAD or the destination itself
// and the destination to be the checked-out branch: the agent's own branch.
func (g *guard) refuseOtherBranch(spec, dst, dir string) PushVerdict {
	if src, _, hasColon := strings.Cut(spec, ":"); hasColon && src != "HEAD" && strings.TrimPrefix(src, refsHeads) != dst {
		return refuse("the source %q must be HEAD or the destination branch itself", src)
	}
	cur := g.currentBranch(dir)
	if cur == "" {
		return refuse("the checked-out branch could not be determined (detached HEAD?); a lease push is allowed only on it")
	}
	if cur != dst {
		return refuse("destination %q is not the checked-out branch %q; a lease push is allowed only on your own branch", dst, cur)
	}
	return PushVerdict{}
}
