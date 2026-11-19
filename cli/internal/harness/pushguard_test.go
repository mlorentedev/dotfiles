package harness

import (
	"strings"
	"testing"
)

const leaseSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// fakeEnv models a machine with one repository on branch feat/x whose remote
// "origin" has default branch main. Other directories are named for what they
// hold: /onmain is on main, /ondev on develop (remote "upstream" defaults to
// develop), /detached has no branch checked out.
func fakeEnv(seen *[]string) PushEnv {
	return PushEnv{
		Dir: "/repo",
		CurrentBranch: func(dir string) string {
			if seen != nil {
				*seen = append(*seen, dir)
			}
			switch dir {
			case "/detached":
				return ""
			case "/onmain":
				return "main"
			case "/onmaster":
				return "master"
			case "/ondev":
				return "develop"
			}
			return "feat/x"
		},
		DefaultBranch: func(_, remote string) string {
			switch remote {
			case "origin":
				return "main"
			case "upstream":
				return "develop"
			}
			return ""
		},
	}
}

type pushRow struct {
	name   string
	cmd    string
	refuse bool
	reason string // substring the refusal must carry
}

func lease(branch string) string { return "--force-with-lease=" + branch + ":" + leaseSHA }

func allowedRows() []pushRow {
	l := lease("feat/x")
	return []pushRow{
		// Pushes that are not force-like stay exactly as they are today.
		{name: "plain push", cmd: "git push origin feat/x"},
		{name: "bare push", cmd: "git push"},
		{name: "upstream flag", cmd: "git push -u origin feat/x"},
		{name: "non-forced push to default branch is out of scope", cmd: "git push origin HEAD:refs/heads/main"},
		{name: "non-forced push of main", cmd: "git push origin main"},
		{name: "plain deletion is out of scope", cmd: "git push origin --delete old-branch"},
		{name: "plain colon deletion is out of scope", cmd: "git push origin :old-branch"},
		{name: "follow-tags is not a force abbreviation", cmd: "git push origin feat/x --follow-tags"},
		{name: "dry run short flags", cmd: "git push -n -v origin feat/x"},
		{name: "negated lease alone", cmd: "git push --no-force-with-lease origin feat/x"},
		{name: "status", cmd: "git status"},
		{name: "force inside a commit message", cmd: `git commit -m "git push --force origin main"`},
		{name: "force inside an echo", cmd: `echo "git push -f origin main"`},
		{name: "force inside single quotes", cmd: `echo 'git push --force'`},
		{name: "push word as an argument", cmd: "git log --grep push --oneline"},
		{name: "chained ordinary", cmd: "git push origin feat/x; echo done"},
		{name: "pipe", cmd: "git log --oneline | head"},
		{name: "empty", cmd: ""},
		{name: "blank", cmd: "   "},
		{name: "unterminated quote without push", cmd: `echo "unterminated`},
		{name: "npm test", cmd: "npm run test"},
		{name: "npm land without push", cmd: "npm run land"},
		{name: "land:push only named", cmd: "echo npm run land:push"},

		// The one allowed force shape and its spellings.
		{name: "lease after remote", cmd: "git push origin " + l + " feat/x"},
		{name: "lease before remote", cmd: "git push " + l + " origin feat/x"},
		{name: "lease last", cmd: "git push origin feat/x " + l},
		{name: "src:dst", cmd: "git push origin " + l + " feat/x:feat/x"},
		{name: "HEAD", cmd: "git push origin " + l + " HEAD"},
		{name: "HEAD:branch", cmd: "git push origin " + l + " HEAD:feat/x"},
		{name: "HEAD:refs/heads", cmd: "git push origin " + l + " HEAD:refs/heads/feat/x"},
		{name: "full refs", cmd: "git push origin " + l + " refs/heads/feat/x:refs/heads/feat/x"},
		{name: "lease with refs/heads", cmd: "git push origin --force-with-lease=refs/heads/feat/x:" + leaseSHA + " feat/x"},
		{name: "sha-256 lease", cmd: "git push origin --force-with-lease=feat/x:" + strings.Repeat("b", 64) + " feat/x"},
		{name: "force-if-includes beside a lease", cmd: "git push origin " + l + " --force-if-includes feat/x"},
		{name: "upstream flag with lease", cmd: "git push -u origin " + l + " feat/x"},
		{name: "git -C", cmd: "git -C /repo push origin " + l + " feat/x"},
		{name: "git -c", cmd: "git -c core.editor=true push origin " + l + " feat/x"},
		{name: "git.exe", cmd: "git.exe push origin " + l + " feat/x"},
		{name: "quoted full path to git", cmd: `"C:/Program Files/Git/cmd/git.exe" push origin ` + l + " feat/x"},
		{name: "env assignment prefix", cmd: "GIT_TRACE=0 git push origin " + l + " feat/x"},
		{name: "chained after cd", cmd: "cd /elsewhere && git push origin " + l + " feat/x"},
		{name: "chained after echo", cmd: "echo hi && git push origin " + l + " feat/x"},
		{name: "non-origin remote with another default", cmd: "git push upstream " + l + " feat/x"},
		{name: "unresolved default, branch is not main or master", cmd: "git -C /ondev push nohead " + lease("develop") + " develop"},
		{name: "bash -c", cmd: `bash -c 'git push origin ` + l + ` feat/x'`},
		{name: "quoted refspec", cmd: `git push origin "` + l + `" "feat/x"`},
	}
}

func refusedRows() []pushRow {
	l := lease("feat/x")
	return []pushRow{
		// Forms that force without a verified lease.
		{name: "--force", cmd: "git push --force origin feat/x", refuse: true, reason: "force"},
		{name: "--force last", cmd: "git push origin feat/x --force", refuse: true, reason: "force"},
		{name: "-f", cmd: "git push -f origin feat/x", refuse: true, reason: "force"},
		{name: "-f after remote", cmd: "git push origin -f feat/x", refuse: true, reason: "force"},
		{name: "cluster -uf", cmd: "git push -uf origin feat/x", refuse: true, reason: "force"},
		{name: "cluster -fu", cmd: "git push -fu origin feat/x", refuse: true, reason: "force"},
		{name: "bare lease", cmd: "git push --force-with-lease origin feat/x", refuse: true, reason: "sha"},
		{name: "lease without sha", cmd: "git push --force-with-lease=feat/x origin feat/x", refuse: true, reason: "sha"},
		{name: "lease with empty sha", cmd: "git push --force-with-lease=feat/x: origin feat/x", refuse: true, reason: "sha"},
		{name: "lease with short sha", cmd: "git push --force-with-lease=feat/x:abc1234 origin feat/x", refuse: true, reason: "sha"},
		{name: "lease with a symbolic expect", cmd: "git push --force-with-lease=feat/x:origin/feat/x origin feat/x", refuse: true, reason: "sha"},
		{name: "force-if-includes alone", cmd: "git push --force-if-includes origin feat/x", refuse: true, reason: "lease"},
		{name: "mirror", cmd: "git push --mirror origin", refuse: true, reason: "mirror"},
		{name: "plus refspec", cmd: "git push origin +feat/x", refuse: true, reason: "+"},
		{name: "plus HEAD to main", cmd: "git push origin +HEAD:main", refuse: true, reason: "+"},
		{name: "plus main", cmd: "git push origin +main", refuse: true, reason: "+"},
		{name: "plus with a lease", cmd: "git push origin " + l + " +feat/x", refuse: true, reason: "+"},
		{name: "force with a lease", cmd: "git push origin " + l + " --force feat/x", refuse: true, reason: "force"},
		{name: "two leases", cmd: "git push origin " + l + " " + lease("feat/x") + " feat/x", refuse: true, reason: "one lease"},
		{name: "lease then bare lease", cmd: "git push origin " + l + " --force-with-lease feat/x", refuse: true, reason: "one lease"},
		{name: "lease and negation", cmd: "git push origin " + l + " --no-force-with-lease feat/x", refuse: true, reason: "negat"},
		{name: "config forcing refspec", cmd: "git -c remote.origin.push=+refs/heads/*:refs/heads/* push origin", refuse: true, reason: "config"},

		// Abbreviated long options git accepts.
		{name: "abbreviated force", cmd: "git push --forc origin feat/x", refuse: true, reason: "option"},
		{name: "abbreviated lease", cmd: "git push --force-w=feat/x:" + leaseSHA + " origin feat/x", refuse: true, reason: "option"},
		{name: "abbreviated mirror", cmd: "git push --mir origin", refuse: true, reason: "option"},
		{name: "abbreviated delete with a lease", cmd: "git push origin " + l + " --del feat/x", refuse: true, reason: "option"},
		{name: "push option with a lease", cmd: "git push origin " + l + " -o ci.skip feat/x", refuse: true, reason: "option"},
		{name: "repo option with a lease", cmd: "git push --repo=origin " + l + " feat/x", refuse: true, reason: "option"},

		// Shape of the lease push.
		{name: "implicit remote", cmd: "git push " + l + " feat/x", refuse: true, reason: "remote"},
		{name: "no refspec", cmd: "git push origin " + l, refuse: true, reason: "refspec"},
		{name: "two refspecs", cmd: "git push origin " + l + " feat/x other", refuse: true, reason: "refspec"},
		{name: "dst differs from lease", cmd: "git push origin " + l + " feat/x:main", refuse: true, reason: "destination"},
		{name: "src differs from dst", cmd: "git push origin " + l + " other:feat/x", refuse: true, reason: "source"},
		{name: "lease names another branch", cmd: "git push origin " + lease("other") + " feat/x", refuse: true, reason: "lease"},
		{name: "destination is not the checked-out branch", cmd: "git push origin " + lease("other") + " other", refuse: true, reason: "checked-out"},
		{name: "main by refspec", cmd: "git push origin " + lease("main") + " HEAD:refs/heads/main", refuse: true, reason: "default branch"},
		{name: "main by name", cmd: "git push origin " + lease("main") + " main", refuse: true, reason: "default branch"},
		{name: "on main", cmd: "git -C /onmain push origin " + lease("main") + " main", refuse: true, reason: "default branch"},
		{name: "master with default unresolved", cmd: "git -C /onmaster push nohead " + lease("master") + " master", refuse: true, reason: "default branch"},
		{name: "resolved default other than main", cmd: "git -C /ondev push upstream " + lease("develop") + " develop", refuse: true, reason: "default branch"},
		{name: "deletion with a lease", cmd: "git push origin " + l + " --delete feat/x", refuse: true, reason: "delet"},
		{name: "colon deletion with a lease", cmd: "git push origin " + l + " :feat/x", refuse: true, reason: "delet"},
		{name: "prune with a lease", cmd: "git push origin " + l + " --prune feat/x", refuse: true, reason: "option"},
		{name: "all with a lease", cmd: "git push origin " + l + " --all", refuse: true, reason: "option"},
		{name: "tags with a lease", cmd: "git push origin " + l + " --tags", refuse: true, reason: "option"},
		{name: "wildcard refspec", cmd: "git push origin " + l + " feat/*", refuse: true, reason: "refspec"},
		{name: "tag destination", cmd: "git push origin " + l + " HEAD:refs/tags/feat/x", refuse: true, reason: "destination"},
		{name: "detached HEAD", cmd: "git -C /detached push origin " + l + " feat/x", refuse: true, reason: "checked-out"},

		// Directory and variable handling.
		{name: "git-dir", cmd: "git --git-dir=/x/.git push origin " + l + " feat/x", refuse: true, reason: "directory"},
		{name: "GIT_DIR env", cmd: "GIT_DIR=/x/.git git push origin " + l + " feat/x", refuse: true, reason: "directory"},
		{name: "GIT_WORK_TREE env", cmd: "GIT_WORK_TREE=/x git push -f origin feat/x", refuse: true, reason: "force"},
		{name: "cd to a variable", cmd: "cd $WT && git push origin " + l + " feat/x", refuse: true, reason: "directory"},
		{name: "variable in the sha", cmd: "git push origin --force-with-lease=feat/x:$SHA feat/x", refuse: true, reason: "variable"},
		{name: "variable in the branch", cmd: "git push origin " + l + " $BR", refuse: true, reason: "variable"},
		{name: "variable in a forced push", cmd: `git push -f origin "$B"`, refuse: true, reason: "force"},
		{name: "unterminated quote in a push", cmd: `git push --force "origin`, refuse: true, reason: "parse"},

		// Where the push hides.
		{name: "git -C", cmd: "git -C /repo push --force origin feat/x", refuse: true, reason: "force"},
		{name: "git -C plus refspec", cmd: "git -C /repo push origin +main", refuse: true, reason: "+"},
		{name: "git -c before push", cmd: "git -c core.editor=true push -f origin feat/x", refuse: true, reason: "force"},
		{name: "git.exe", cmd: "git.exe push -f origin feat/x", refuse: true, reason: "force"},
		{name: "quoted full path", cmd: `"C:/Program Files/Git/cmd/git.exe" push --force origin x`, refuse: true, reason: "force"},
		{name: "env prefix", cmd: "FOO=1 git push -f origin x", refuse: true, reason: "force"},
		{name: "command wrapper", cmd: "command git push -f origin x", refuse: true, reason: "force"},
		{name: "and chain", cmd: "true && git push --force origin x", refuse: true, reason: "force"},
		{name: "semicolon chain", cmd: "git status; git push -f origin x", refuse: true, reason: "force"},
		{name: "or chain", cmd: "git status || git push --force", refuse: true, reason: "force"},
		{name: "pipe", cmd: "echo y | git push --force", refuse: true, reason: "force"},
		{name: "background", cmd: "sleep 1 & git push -f origin x", refuse: true, reason: "force"},
		{name: "newline", cmd: "git status\ngit push -f origin x", refuse: true, reason: "force"},
		{name: "subshell", cmd: "(git push -f origin x)", refuse: true, reason: "force"},
		{name: "command substitution", cmd: "echo $(git push --force origin x)", refuse: true, reason: "force"},
		{name: "quoted command substitution", cmd: `echo "$(git push --force origin x)"`, refuse: true, reason: "force"},
		{name: "backticks", cmd: "echo `git push -f origin x`", refuse: true, reason: "force"},
		{name: "bash -c", cmd: `bash -c 'git push --force origin x'`, refuse: true, reason: "force"},
		{name: "sh -c double quotes", cmd: `sh -c "git push origin +main"`, refuse: true, reason: "+"},
		{name: "bash -lc", cmd: `bash -lc "git push -f"`, refuse: true, reason: "force"},
		{name: "eval", cmd: `eval "git push -f"`, refuse: true, reason: "force"},
		{name: "nested bash -c", cmd: `bash -c "bash -c 'git push -f'"`, refuse: true, reason: "force"},
		{name: "main by refspec after force", cmd: "git push origin HEAD:refs/heads/main --force", refuse: true, reason: "force"},
		{name: "lease with main refspec", cmd: "git push origin " + lease("main") + " HEAD:refs/heads/main", refuse: true, reason: "default branch"},

		// The owner's landing mode.
		{name: "land:push", cmd: "npm run land:push", refuse: true, reason: "land:push"},
		{name: "land:push with arguments", cmd: "npm run land:push -- --dry-run x", refuse: true, reason: "land:push"},
		{name: "land:push chained", cmd: "cd x && npm run land:push", refuse: true, reason: "land:push"},
		{name: "land:push via pnpm", cmd: "pnpm run land:push", refuse: true, reason: "land:push"},
		{name: "land:push via yarn", cmd: "yarn land:push", refuse: true, reason: "land:push"},
		{name: "land:push with prefix", cmd: "npm --prefix x run land:push", refuse: true, reason: "land:push"},
		{name: "land:push run-script", cmd: "npm run-script land:push", refuse: true, reason: "land:push"},
		{name: "land:push in bash -c", cmd: `bash -c "npm run land:push"`, refuse: true, reason: "land:push"},
	}
}

func TestGuardCommand_Table(t *testing.T) {
	rows := append(allowedRows(), refusedRows()...)
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			got := GuardCommand(r.cmd, ShellPOSIX, fakeEnv(nil))
			if got.Refused != r.refuse {
				t.Fatalf("GuardCommand(%q) refused=%v (%q), want refused=%v", r.cmd, got.Refused, got.Reason, r.refuse)
			}
			if r.refuse {
				if got.Reason == "" {
					t.Fatalf("GuardCommand(%q) refused without a reason", r.cmd)
				}
				if !strings.Contains(strings.ToLower(got.Reason), strings.ToLower(r.reason)) {
					t.Fatalf("GuardCommand(%q) reason %q does not mention %q", r.cmd, got.Reason, r.reason)
				}
			}
		})
	}
}

func TestGuardCommand_PowerShell(t *testing.T) {
	cases := []struct {
		cmd    string
		refuse bool
	}{
		{"git push -f origin x; git status", true},
		{"git status; git push origin +main", true},
		{"git push origin " + lease("feat/x") + " feat/x", false},
		{"git push origin feat/x", false},
	}
	for _, c := range cases {
		if got := GuardCommand(c.cmd, ShellPowerShell, fakeEnv(nil)); got.Refused != c.refuse {
			t.Errorf("GuardCommand(%q, powershell) refused=%v (%q), want %v", c.cmd, got.Refused, got.Reason, c.refuse)
		}
	}
}

// The directory the branch is read from is the one git would run in: the
// payload's cwd, moved by `cd` and `-C`, in the spellings a Windows box sends.
func TestGuardCommand_DirectoryResolution(t *testing.T) {
	l := lease("feat/x")
	cases := []struct {
		name string
		cmd  string
		want string
	}{
		{"cwd", "git push origin " + l + " feat/x", "/repo"},
		{"cd absolute", "cd /elsewhere && git push origin " + l + " feat/x", "/elsewhere"},
		{"cd relative", "cd sub && git push origin " + l + " feat/x", "/repo/sub"},
		{"cd dotdot", "cd .. && git push origin " + l + " feat/x", "/"},
		{"git -C absolute", "git -C /other push origin " + l + " feat/x", "/other"},
		{"git -C relative", "git -C sub push origin " + l + " feat/x", "/repo/sub"},
		{"git -C twice", "git -C /a -C b push origin " + l + " feat/x", "/a/b"},
		{"cd then -C", "cd /a && git -C b push origin " + l + " feat/x", "/a/b"},
		{"msys drive path", "cd /c/Users/u/repo && git push origin " + l + " feat/x", "/c/Users/u/repo"},
		{"windows backslashes", `cd C:\Users\u\repo && git push origin ` + l + " feat/x", "C:/Users/u/repo"},
		{"windows forward slashes", `cd "C:/Users/u/repo" && git push origin ` + l + " feat/x", "C:/Users/u/repo"},
		{"windows -C", `git -C C:\Users\u\repo push origin ` + l + " feat/x", "C:/Users/u/repo"},
		{"pushd", "pushd /elsewhere && git push origin " + l + " feat/x", "/elsewhere"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var seen []string
			got := GuardCommand(c.cmd, ShellPOSIX, fakeEnv(&seen))
			if got.Refused {
				t.Fatalf("refused: %s", got.Reason)
			}
			if len(seen) == 0 || seen[len(seen)-1] != c.want {
				t.Fatalf("branch read from %v, want %q", seen, c.want)
			}
		})
	}
}

// A guard that does work on input it has no business with becomes one whose
// normal state is slow or red; ordinary commands must never reach the resolvers.
func TestGuardCommand_OrdinaryCommandsDoNotTouchGit(t *testing.T) {
	var seen []string
	env := fakeEnv(&seen)
	env.DefaultBranch = func(_, _ string) string { t.Fatal("default branch resolved for an ordinary command"); return "" }
	for _, cmd := range []string{"git push origin feat/x", "ls -la", "git status && git diff", "echo hi"} {
		if got := GuardCommand(cmd, ShellPOSIX, env); got.Refused {
			t.Errorf("%q refused: %s", cmd, got.Reason)
		}
	}
	if len(seen) != 0 {
		t.Errorf("current branch read for ordinary commands: %v", seen)
	}
}

// A commit message or a file written through a heredoc is text: it may mention
// `git push --force` and carry an apostrophe without being either. A heredoc fed
// to a shell is code.
func TestGuardCommand_Heredocs(t *testing.T) {
	cases := []struct {
		name   string
		cmd    string
		refuse bool
	}{
		{"commit message mentioning a force with an apostrophe", "git commit -m \"$(cat <<'EOF'\nrefuse git push --force; don't touch main\nEOF\n)\"", false},
		{"cat to a file", "cat > notes.md <<'EOF'\ngit push -f origin main\nit's text\nEOF\ngit status", false},
		{"unquoted delimiter", "cat <<EOF\ngit push --force\nEOF", false},
		{"dash form with tabs", "cat <<-EOF\n\tgit push -f\n\tEOF\necho done", false},
		{"force after the heredoc is still seen", "cat <<'EOF'\ntext\nEOF\ngit push --force origin x", true},
		{"force before the heredoc is still seen", "git push -f origin x <<'EOF'\ntext\nEOF", true},
		{"heredoc fed to bash is code", "bash <<'EOF'\ngit push -f origin x\nEOF", true},
		{"heredoc fed to sh is code", "sh <<EOF\ngit push origin +main\nEOF", true},
		{"unterminated heredoc swallows the rest as text", "cat <<'EOF'\ngit push -f origin x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GuardCommand(c.cmd, ShellPOSIX, fakeEnv(nil)); got.Refused != c.refuse {
				t.Fatalf("refused=%v (%q), want %v for %q", got.Refused, got.Reason, c.refuse, c.cmd)
			}
		})
	}
}

// Bypasses found by the independent review: each of these reached a forced push
// (or a lease push read from the wrong directory) with the guard silent.
func TestGuardCommand_ReviewBypasses(t *testing.T) {
	sha := strings.Repeat("a", 40)
	lease := "--force-with-lease=feat/x:" + sha
	cases := []struct {
		name   string
		cmd    string
		refuse bool
	}{
		{"heredoc operator inside quotes", "echo \"<<EOF\"\ngit push --force origin main", true},
		{"heredoc piped to bash", "cat <<EOF | bash\ngit push --force origin main\nEOF", true},
		{"heredoc to a path-qualified shell", "/bin/bash <<EOF\ngit push --force origin main\nEOF", true},
		{"heredoc to a windows-style shell path", "C:/Git/bin/bash.exe <<EOF\ngit push -f origin x\nEOF", true},
		{"env with an option argument", "env -u FOO git push --force origin main", true},
		{"sudo with a user", "sudo -u root git push --force origin main", true},
		{"command with an option", "command -p git push --force origin main", true},
		{"exec with an option argument", "exec -a foo git push -f origin main", true},
		{"env sets GIT_DIR for a lease push", "env GIT_DIR=/x/.git git push origin " + lease + " feat/x", true},
		{"env with a plain assignment and a lease", "env FOO=1 git push origin " + lease + " feat/x", false},
		{"sudo -E keeps the program", "sudo -E git push origin " + lease + " feat/x", false},
		{"subshell inside a double-quoted substitution", "echo \"$( (true); git push --force origin main)\"", true},
		{"subshell inside a substitution, then a harmless command", "echo \"$( (true); echo ok)\"", false},
		{"attached -C directory", "git -C/onmain push origin " + lease + " feat/x", true},
		{"attached -c forcing config", "git -cremote.origin.push=+refs/heads/main:refs/heads/main push origin", true},
		{"attached -c harmless config", "git -cuser.name=x push origin feat/x", false},
		{"bash -c after --", "bash -c -- 'git push --force origin main'", true},
		{"bash -c after -- harmless", "bash -c -- 'git status'", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GuardCommand(c.cmd, ShellPOSIX, fakeEnv(nil)); got.Refused != c.refuse {
				t.Fatalf("refused=%v (%q), want %v for %q", got.Refused, got.Reason, c.refuse, c.cmd)
			}
		})
	}
}

// Second review round: more spellings of the same families.
func TestGuardCommand_SecondRoundBypasses(t *testing.T) {
	sha := strings.Repeat("a", 40)
	lease := "--force-with-lease=feat/x:" + sha
	cases := []struct {
		name   string
		cmd    string
		refuse bool
		shell  Shell
	}{
		{"heredoc operator in a multi-line quote", "echo \"first line\n<<EOF\"\ngit push --force origin main", true, ShellPOSIX},
		{"heredoc delimiter with a dash", "cat <<END-OF-MSG\ntext\nEND-OF-MSG\ngit push --force origin main", true, ShellPOSIX},
		{"heredoc operator in a comment", "# <<EOF\ngit push --force origin main", true, ShellPOSIX},
		{"no space between the shell and the operator", "bash<<EOF\ngit push --force origin main\nEOF", true, ShellPOSIX},
		{"heredoc piped to bash then more", "cat <<EOF | bash; echo done\ngit push --force origin main\nEOF", true, ShellPOSIX},
		{"heredoc piped to a parenthesised shell", "cat <<EOF | (bash)\ngit push --force origin main\nEOF", true, ShellPOSIX},
		{"heredoc with a dashed delimiter stays text", "cat <<END-OF-MSG\ngit push --force\nEND-OF-MSG\ngit status", false, ShellPOSIX},
		{"wrapper by path", "/usr/bin/env git push --force origin main", true, ShellPOSIX},
		{"env -C moves the repository", "env -C /onmain git push origin " + lease + " feat/x", true, ShellPOSIX},
		{"env --chdir moves the repository", "env --chdir=/onmain git push origin " + lease + " feat/x", true, ShellPOSIX},
		{"GIT_DIR before a nested shell", "GIT_DIR=/x/.git bash -c 'git push origin " + lease + " feat/x'", true, ShellPOSIX},
		{"export GIT_DIR then a lease push", "export GIT_DIR=/x/.git && git push origin " + lease + " feat/x", true, ShellPOSIX},
		{"GIT_DIR assigned alone then a lease push", "GIT_DIR=/x/.git; git push origin " + lease + " feat/x", true, ShellPOSIX},
		{"a cd outside any subshell still works", "cd /onmain && git push origin " + lease + " feat/x", true, ShellPOSIX},
		{"eval after --", "eval -- \"git push --force origin main\"", true, ShellPOSIX},
		{"powershell -Command with loose words", "pwsh -Command git push --force origin main", true, ShellPOSIX},
		{"powershell tool -Command with loose words", "powershell -Command git push -f origin x", true, ShellPowerShell},
		{"ANSI-C quote then land:push", `echo $'hello\'' ; npm run land:push`, true, ShellPOSIX},
		{"ANSI-C quote alone", `echo $'it\'s fine'`, false, ShellPOSIX},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GuardCommand(c.cmd, c.shell, fakeEnv(nil)); got.Refused != c.refuse {
				t.Fatalf("refused=%v (%q), want %v for %q", got.Refused, got.Reason, c.refuse, c.cmd)
			}
		})
	}
}

// A cd inside a subshell or a substitution does not change the directory of the
// commands after it, so the branch must not be read from there. The session is
// on main here; the subshell visits a feature branch.
func TestGuardCommand_CdInsideASubshellDoesNotCarryOut(t *testing.T) {
	lease := "--force-with-lease=feat/x:" + strings.Repeat("a", 40)
	env := fakeEnv(nil)
	env.Dir = "/onmain"
	for _, cmd := range []string{
		"(cd /repo) && git push origin " + lease + " feat/x",
		"echo $(cd /repo) && git push origin " + lease + " feat/x",
	} {
		if got := GuardCommand(cmd, ShellPOSIX, env); !got.Refused {
			t.Errorf("not refused: %q", cmd)
		}
	}
	if got := GuardCommand("cd /repo && git push origin "+lease+" feat/x", ShellPOSIX, env); got.Refused {
		t.Errorf("a plain cd must still move the directory: %q", got.Reason)
	}
}

// Third review round.
func TestGuardCommand_ThirdRoundBypasses(t *testing.T) {
	lease := "--force-with-lease=feat/x:" + strings.Repeat("a", 40)
	cases := []struct {
		name   string
		cmd    string
		refuse bool
		shell  Shell
	}{
		{"pwsh -C is -Command", "pwsh -C \"git push --force origin main\"", true, ShellPOSIX},
		{"pwsh -Co is -Command", "pwsh -Co \"git push --force origin main\"", true, ShellPOSIX},
		{"powershell -C in the PowerShell tool", "powershell -C \"git push -f origin x\"", true, ShellPowerShell},
		{"pwsh -C harmless", "pwsh -C \"git status\"", false, ShellPOSIX},
		{"quoted redirect target before the command", "> \"out.txt\" git push --force origin main", true, ShellPOSIX},
		{"attached quoted redirect target", ">\"out.txt\" git push --force origin main", true, ShellPOSIX},
		{"stderr redirect with a quoted target", "2>\"err.txt\" git push --force origin main", true, ShellPOSIX},
		{"a quoted redirect around a harmless command", ">\"out.txt\" echo hi", false, ShellPOSIX},
		{"env -S runs a script", "env -S \"git push --force origin main\"", true, ShellPOSIX},
		{"env --split-string runs a script", "env --split-string=\"git push -f origin x\"", true, ShellPOSIX},
		{"env -S harmless", "env -S \"git status\"", false, ShellPOSIX},
		{"env attached -C moves the repository", "env -C/onmain git push origin " + lease + " feat/x", true, ShellPOSIX},
		{"heredoc delimiter with a mid-word backslash", "cat <<E\\OF\nhi\nEOF\ngit push --force origin main", true, ShellPOSIX},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GuardCommand(c.cmd, c.shell, fakeEnv(nil)); got.Refused != c.refuse {
				t.Fatalf("refused=%v (%q), want %v for %q", got.Refused, got.Reason, c.refuse, c.cmd)
			}
		})
	}
}

// A child shell's cd is gone when it exits; eval runs in this shell, so its cd stays.
func TestGuardCommand_ChildShellCdDoesNotCarryBack(t *testing.T) {
	lease := "--force-with-lease=feat/x:" + strings.Repeat("a", 40)
	env := fakeEnv(nil)
	env.Dir = "/onmain"
	for _, cmd := range []string{
		"bash -c 'cd /repo' && git push origin " + lease + " feat/x",
		"sh -c \"cd /repo\" ; git push origin " + lease + " feat/x",
	} {
		if got := GuardCommand(cmd, ShellPOSIX, env); !got.Refused {
			t.Errorf("not refused: %q", cmd)
		}
	}
	if got := GuardCommand("eval 'cd /repo' && git push origin "+lease+" feat/x", ShellPOSIX, env); got.Refused {
		t.Errorf("eval's cd runs in this shell and must carry: %q", got.Reason)
	}
}

// Fourth review round.
func TestGuardCommand_FourthRoundBypasses(t *testing.T) {
	env := fakeEnv(nil)
	env.Dir = "/onmain"
	cases := []struct {
		name   string
		cmd    string
		refuse bool
		shell  Shell
	}{
		{"bundled env -vS", "env -vS \"git push --force origin main\"", true, ShellPOSIX},
		{"bundled env -iS attached", "env -iS\"git push -f origin x\"", true, ShellPOSIX},
		{"env -C names a directory called git", "/usr/bin/env -C /tmp/git git push --force origin main", true, ShellPOSIX},
		{"sudo -u names a user called git", "sudo -u git git push --force origin main", true, ShellPOSIX},
		{"exec -a names a name called git", "exec -a git git push -f origin x", true, ShellPOSIX},
		{"command -p keeps git as the program", "command -p git push --force origin main", true, ShellPOSIX},
		{"sudo -E keeps git as the program", "sudo -E git status", false, ShellPOSIX},
		{"powershell block comment before the command", "<# note #> git push --force origin main", true, ShellPowerShell},
		{"powershell block comment alone", "<# git push --force origin main #> git status", false, ShellPowerShell},
		{"escaped heredoc operator", "echo \\<<EOF\ngit push --force origin main", true, ShellPOSIX},
		{"quoted shell name fed a heredoc", "\"bash\" <<EOF\ngit push --force origin main\nEOF", true, ShellPOSIX},
		{"heredoc piped to a quoted shell name", "cat <<EOF | \"bash\"\ngit push --force origin main\nEOF", true, ShellPOSIX},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GuardCommand(c.cmd, c.shell, env); got.Refused != c.refuse {
				t.Fatalf("refused=%v (%q), want %v for %q", got.Refused, got.Reason, c.refuse, c.cmd)
			}
		})
	}
}

// The session is on a feature branch; an option moves the command to main. Reading
// the branch from the session's own directory would let the lease through.
func TestGuardCommand_OptionMovesTheDirectory(t *testing.T) {
	lease := "--force-with-lease=feat/x:" + strings.Repeat("a", 40)
	push := "git push origin " + lease + " feat/x"
	for _, c := range []struct {
		cmd   string
		shell Shell
	}{
		{"env -C /onmain -S \"" + push + "\"", ShellPOSIX},
		{"pwsh -WorkingDirectory /onmain -C \"" + push + "\"", ShellPowerShell},
		{"pwsh -wd /onmain -C \"" + push + "\"", ShellPowerShell},
	} {
		if got := GuardCommand(c.cmd, c.shell, fakeEnv(nil)); !got.Refused {
			t.Errorf("not refused: %q", c.cmd)
		}
	}
}
