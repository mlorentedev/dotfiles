package tools

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const sampleCatalog = `{
  "tools": [
    {
      "name": "sops",
      "version": "3.13.1",
      "profile": "full",
      "source": {
        "type": "github-release",
        "repo": "getsops/sops",
        "asset": {
          "linux": "sops-v{version}.linux.{goarch}",
          "darwin": "sops-v{version}.darwin.{goarch}",
          "windows": "sops-v{version}.{goarch}.exe"
        },
        "checksums": "sops-v{version}.checksums.txt"
      }
    }
  ]
}`

func writeCatalog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "packages.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	cat, err := Load(writeCatalog(t, sampleCatalog))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cat.Tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(cat.Tools))
	}
	got := cat.Tools[0]
	if got.Name != "sops" || got.Version != "3.13.1" || got.Profile != "full" {
		t.Fatalf("unexpected tool: %+v", got)
	}
	if got.Source.Type != "github-release" || got.Source.Repo != "getsops/sops" {
		t.Fatalf("unexpected source: %+v", got.Source)
	}
	if got.Source.Checksums != "sops-v{version}.checksums.txt" {
		t.Fatalf("unexpected checksums template: %q", got.Source.Checksums)
	}
}

func TestChecksumsName(t *testing.T) {
	tool := Tool{Version: "3.13.1", Source: Source{Checksums: "sops-v{version}.checksums.txt"}}
	if got, want := tool.ChecksumsName("amd64"), "sops-v3.13.1.checksums.txt"; got != want {
		t.Errorf("ChecksumsName = %q, want %q", got, want)
	}
	if got := (Tool{Version: "1.0.0"}).ChecksumsName("amd64"); got != "" {
		t.Errorf("ChecksumsName with no template = %q, want empty", got)
	}
}

// A tool listed twice is a catalog error, not two installs: the duplicate
// copilot entry of AI-038 (#1321) installed once and then reported "already
// installed; skipping" for its twin, which read as idempotence.
func TestLoad_RejectsDuplicateToolNames(t *testing.T) {
	dup := `{"tools":[{"name":"copilot","version":"1.0.81","profile":"full","source":{"type":"npm","package":"@github/copilot"}},{"name":"copilot","version":"1.0.81","profile":"full","source":{"type":"npm","package":"@github/copilot"}}]}`
	_, err := Load(writeCatalog(t, dup))
	if err == nil {
		t.Fatal("a duplicated tool name must be a load error")
	}
	if got := err.Error(); !strings.Contains(got, `tool "copilot" is listed more than once`) {
		t.Fatalf("error must name the tool, got: %v", err)
	}
}

func TestLoad_Errors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error for missing catalog")
	}
	if _, err := Load(writeCatalog(t, "{ not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAssetName(t *testing.T) {
	tool := Tool{
		Version: "3.13.1",
		Source: Source{Asset: map[string]string{
			"linux":   "sops-v{version}.linux.{goarch}",
			"darwin":  "sops-v{version}.darwin.{goarch}",
			"windows": "sops-v{version}.{goarch}.exe",
		}},
	}
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "sops-v3.13.1.linux.amd64"},
		{"linux", "arm64", "sops-v3.13.1.linux.arm64"},
		{"darwin", "arm64", "sops-v3.13.1.darwin.arm64"},
		{"windows", "amd64", "sops-v3.13.1.amd64.exe"}, // irregular: goarch before .exe, no OS token
		{"plan9", "amd64", ""},                         // unsupported OS → empty
	}
	for _, tc := range cases {
		if got := tool.AssetName(tc.goos, tc.goarch); got != tc.want {
			t.Errorf("AssetName(%q,%q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestLoad_Platforms(t *testing.T) {
	ok := `{"tools":[{"name":"hive","version":"4.2.2","profile":"full","source":{"type":"uv-tool","package":"hive-vault","platforms":["linux","darwin"]}}]}`
	c, err := Load(writeCatalog(t, ok))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	hive := c.Tools[0]
	for goos, want := range map[string]bool{"linux": true, "darwin": true, "windows": false} {
		if got := hive.SupportsOS(goos); got != want {
			t.Errorf("SupportsOS(%s) = %v, want %v", goos, got, want)
		}
	}
	if !(Tool{}).SupportsOS("windows") {
		t.Error("a tool that lists no platforms must support every OS")
	}

	// A misspelt platform would make the tool unsupported everywhere, in silence.
	typo := strings.Replace(ok, `"darwin"`, `"macos"`, 1)
	if _, err := Load(writeCatalog(t, typo)); err == nil || !strings.Contains(err.Error(), "macos") {
		t.Errorf("Load accepted an unknown platform: %v", err)
	}
}

// The shipped catalog declares hive as a uv tool on every OS, which hive
// installs itself on Windows: there hive owns its install layout (hive
// ADR-019), so the catalog must never put a uv copy there. A misspelt
// self_install key is dropped by the decoder, and this is what notices.
func TestTheRepoCatalogDeclaresHiveSelfInstalledOnWindows(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "..", "packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range c.Tools {
		if tool.Name != "hive" {
			continue
		}
		if tool.Source.Type != "uv-tool" || tool.Source.Package != "hive-vault" {
			t.Errorf("hive source = %+v, want uv-tool hive-vault", tool.Source)
		}
		for goos, self := range map[string]bool{"linux": false, "darwin": false, "windows": true} {
			if !tool.SupportsOS(goos) || tool.Source.SelfInstalls(goos) != self {
				t.Errorf("hive on %s: supported %v, self-installed %v; want supported, self-installed %v", goos, tool.SupportsOS(goos), tool.Source.SelfInstalls(goos), self)
			}
		}
		return
	}
	t.Fatal("packages.json declares no hive tool")
}

// A goos/goarch key states the exact asset for one platform and wins over the
// goos key; the goos key still answers alone. mise needs this: it names its
// arches x64/arm64 and its OS macos, which {goarch} cannot express.
func TestAssetName_PrefersGoosGoarchOverGoos(t *testing.T) {
	tool := Tool{Version: "2026.10.3", Source: Source{Asset: map[string]string{
		"linux/amd64":  "mise-v{version}-linux-x64",
		"darwin/arm64": "mise-v{version}-macos-arm64",
		"linux":        "generic-{goarch}",
	}}}
	cases := []struct{ goos, goarch, want string }{
		{"linux", "amd64", "mise-v2026.10.3-linux-x64"},
		{"darwin", "arm64", "mise-v2026.10.3-macos-arm64"},
		{"linux", "arm64", "generic-arm64"}, // no specific key: the goos key answers
		{"darwin", "amd64", ""},             // neither: unsupported
	}
	for _, tc := range cases {
		if got := tool.AssetName(tc.goos, tc.goarch); got != tc.want {
			t.Errorf("AssetName(%q,%q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}

// With a missing asset now a skip, a misspelt key would silently install
// nothing anywhere; Load refuses it instead.
func TestLoad_RejectsAnUnknownAssetKey(t *testing.T) {
	for _, key := range []string{"macos", "linux/x64", "darwin/arm64/extra"} {
		path := filepath.Join(t.TempDir(), "packages.json")
		body := `{"tools":[{"name":"t","version":"1.0.0","source":{"type":"github-release","repo":"o/r","asset":{"` + key + `":"a"},"checksums":"s"}}]}`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("asset key %q: want an error naming it, got %v", key, err)
		}
	}
}

func TestLoad_RejectsAReleaseToolWithNoAsset(t *testing.T) {
	for name, source := range map[string]string{
		"no asset map":       `{"type":"github-release","repo":"o/r","checksums":"s"}`,
		"misspelt as assets": `{"type":"github-release","repo":"o/r","assets":{"linux":"a"},"checksums":"s"}`,
	} {
		path := filepath.Join(t.TempDir(), "packages.json")
		body := `{"tools":[{"name":"t","version":"1.0.0","source":` + source + `}]}`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "declares no asset") {
			t.Errorf("%s: want a load error, got %v", name, err)
		}
	}
}

// A system entry names its package per OS manager. One that names none can never
// install anywhere, one that names an unknown manager (pacman, dnf) is a typo the
// decoder would drop in silence, and one that carries a version asks for a pin a
// system package does not have: each is an error naming the entry.
func TestLoad_RejectsMalformedSystemEntries(t *testing.T) {
	cases := []struct {
		name, source, version, want string
	}{
		{"no manager at all", `{"type":"system"}`, "", "names no package manager"},
		{"a platforms list is not a manager", `{"type":"system","platforms":["linux"]}`, "", "names no package manager"},
		{"an unknown manager key", `{"type":"system","apt":"gh","pacman":"github-cli"}`, "", `"pacman"`},
		{"a key of another source kind", `{"type":"system","apt":"gh","package":"gh"}`, "", `"package"`},
		{"platforms that exclude every named manager", `{"type":"system","apt":"gh","platforms":["darwin","windows"]}`, "", "names no package for any"},
		{"a version", `{"type":"system","apt":"gh"}`, `"version":"2.40.0",`, "not pinned"},
		{"a name that is a flag", `{"type":"system","apt":"-y"}`, "", "apt"},
		{"a name with a space", `{"type":"system","brew":"gh cli"}`, "", "brew"},
		{"a cask name that is a flag", `{"type":"system","cask":"--force"}`, "", "cask"},
		{"an apt name with a command separator", `{"type":"system","apt":"x;rm"}`, "", "apt"},
		{"a brew name with a substitution", `{"type":"system","brew":"x$(id)"}`, "", "brew"},
		{"a winget id with a backtick", "{\"type\":\"system\",\"winget\":\"x`id\"}", "", "winget"},
		{"a cask name with a redirect", `{"type":"system","cask":"x>y"}`, "", "cask"},
		{"a command with a pipe", `{"type":"system","apt":"gh","command":"gh|id"}`, "", "command"},
		{"a formula and a cask for one OS", `{"type":"system","brew":"gh","cask":"gh"}`, "", "both a brew formula and a cask"},
		{"needs_sudo on an entry with no cask", `{"type":"system","apt":"gh","brew":"gh","needs_sudo":true}`, "", "needs_sudo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"tools":[{"name":"gh",` + tc.version + `"source":` + tc.source + `}]}`
			_, err := Load(writeCatalog(t, body))
			if err == nil || !strings.Contains(err.Error(), `"gh"`) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want an error naming gh and %q, got %v", tc.want, err)
			}
		})
	}
}

func TestLoad_AcceptsASystemEntry(t *testing.T) {
	body := `{"tools":[{"name":"gh","source":{"type":"system","apt":"gh","brew":"gh","winget":"GitHub.cli","command":"gh","platforms":["linux","darwin","windows"]}}]}`
	c, err := Load(writeCatalog(t, body))
	if err != nil {
		t.Fatalf("a well-formed system entry must load: %v", err)
	}
	s := c.Tools[0].Source
	if s.Apt != "gh" || s.Brew != "gh" || s.Winget != "GitHub.cli" || s.Command != "gh" {
		t.Errorf("source = %+v", s)
	}
}

func TestLoad_AcceptsACaskEntry(t *testing.T) {
	body := `{"tools":[{"name":"obsidian","source":{"type":"system","cask":"obsidian"}}]}`
	c, err := Load(writeCatalog(t, body))
	if err != nil {
		t.Fatalf("a cask-only system entry must load: %v", err)
	}
	if m, pkg := c.Tools[0].Source.SystemPackage("darwin"); m != "brew-cask" || pkg != "obsidian" {
		t.Errorf("SystemPackage(darwin) = %s, %s; want brew-cask, obsidian", m, pkg)
	}
	if c.Tools[0].SupportsOS("linux") || c.Tools[0].SupportsOS("windows") || !c.Tools[0].SupportsOS("darwin") {
		t.Error("a cask installs on darwin only")
	}

	marked := `{"tools":[{"name":"veracrypt","source":{"type":"system","cask":"veracrypt","needs_sudo":true}}]}`
	m, err := Load(writeCatalog(t, marked))
	if err != nil || !m.Tools[0].Source.NeedsSudo {
		t.Errorf("a cask marked needs_sudo must load with the mark: %+v, %v", m, err)
	}

	// The platforms reachability check resolves through SystemPackage, so a
	// cask is a package darwin can install.
	withPlatforms := `{"tools":[{"name":"obsidian","source":{"type":"system","cask":"obsidian","platforms":["darwin"]}}]}`
	if _, err := Load(writeCatalog(t, withPlatforms)); err != nil {
		t.Errorf("a cask entry with platforms [darwin] must load: %v", err)
	}
}

// The skip for an OS the entry names no manager for is the platform skip every
// other source takes, so Install, Plan and `tools list` need no branch of their own.
func TestSystemSupportsOnlyTheOSesItNamesAPackageFor(t *testing.T) {
	gh := Tool{Name: "gh", Source: Source{Type: "system", Apt: "gh", Winget: "GitHub.cli"}}
	for goos, want := range map[string]bool{"linux": true, "darwin": false, "windows": true} {
		if got := gh.SupportsOS(goos); got != want {
			t.Errorf("SupportsOS(%s) = %v, want %v", goos, got, want)
		}
	}
	listed := Tool{Name: "gh", Source: Source{Type: "system", Apt: "gh", Brew: "gh", Platforms: []string{"darwin"}}}
	if listed.SupportsOS("linux") || !listed.SupportsOS("darwin") {
		t.Error("a platforms list narrows a system entry further")
	}
}

// An unknown source type is a skip with a warning at install time, so a typo in
// the shipped catalog ("npn") would skip a tool on every machine and exit 0. The
// reader cannot tell it from a type a newer dotf adds; this guard can, for the
// one catalog the repository ships.
func TestTheRepoCatalogUsesOnlyKnownSourceTypes(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "..", "packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, k := range KnownSourceTypes {
		known[k] = true
	}
	for _, tool := range c.Tools {
		if !known[tool.Source.Type] {
			t.Errorf("tool %q has source type %q, which this dotf would skip (known: %v)", tool.Name, tool.Source.Type, KnownSourceTypes)
		}
	}
}

// macOS ships /bin/bash 3.2, and `env bash` there needs brew's bash 5 (#2202).
// The entry must not declare `command: bash`: a command found on PATH satisfies
// a system entry, and /bin/bash is always found, so the install would never
// run. Presence is brew's own record instead. Only brew names it: Linux ships a
// current bash, and Windows has none to replace.
func TestTheRepoCatalogDeclaresBrewBashWithoutACommand(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "..", "packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range c.Tools {
		if tool.Name != "bash" {
			continue
		}
		if tool.Source.Type != "system" || tool.Source.Brew != "bash" || tool.Source.Apt != "" || tool.Source.Winget != "" {
			t.Errorf("bash source = %+v, want system, brew only", tool.Source)
		}
		if tool.Source.Command != "" {
			t.Errorf("bash declares command %q, which /bin/bash 3.2 would satisfy", tool.Source.Command)
		}
		return
	}
	t.Fatal("packages.json declares no bash entry")
}

// The charset is the shipping catalog's and the managers' own forms: a rule that
// refused a real name would break a converge, so each form is a case.
func TestLoad_AcceptsEveryFormAManagerDefines(t *testing.T) {
	for _, name := range []string{"gh", "libstdc++", "node@20", "owner/tap/formula", "GitHub.cli", "python3.12", "pkg:amd64", "a_b-c"} {
		body := `{"tools":[{"name":"t","source":{"type":"system","apt":"` + name + `"}}]}`
		if _, err := Load(writeCatalog(t, body)); err != nil {
			t.Errorf("%q must load: %v", name, err)
		}
	}
}

// The catalog this repository ships is the other half of the acceptance: the
// stricter rule must not turn a working entry into a load error.
func TestLoad_TheShippingCatalogStillLoads(t *testing.T) {
	if _, err := Load(filepath.Join("..", "..", "..", "packages.json")); err != nil {
		t.Fatalf("packages.json: %v", err)
	}
}

// Gitea's CLI is `tea` in brew and `Gitea.tea` in winget, but apt's `tea` is
// an unrelated graphical text editor (Debian, measured on msi 2026-10-17), so an
// apt name would install the wrong program and `command: tea` would then report
// it satisfied. Linux has no packaged channel and skips the entry, said so.
func TestTheRepoCatalogDeclaresGiteaTeaWithoutApt(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "..", "packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range c.Tools {
		if tool.Name != "tea" {
			continue
		}
		s := tool.Source
		if s.Type != "system" || s.Brew != "tea" || s.Winget != "Gitea.tea" || s.Command != "tea" {
			t.Errorf("tea source = %+v, want system, brew tea, winget Gitea.tea, command tea", s)
		}
		if s.Apt != "" {
			t.Errorf("tea names apt %q: apt's tea is a text editor, not Gitea's CLI", s.Apt)
		}
		if tool.SupportsOS("linux") {
			t.Error("tea must skip linux, where no channel packages Gitea's CLI")
		}
		return
	}
	t.Fatal("packages.json declares no tea entry")
}

// self_install hands a uv-tool's install to the tool itself on the OSes it
// lists (hive self-upgrade on Windows, ADR-036 amendment 2026-10-21). Each
// refusal is a catalog that would install something other than the pin, or
// nothing, without saying so.
func TestLoad_SelfInstall(t *testing.T) {
	const ok = `{"tools":[{"name":"hive","version":"4.4.1","profile":"full","source":{"type":"uv-tool","package":"hive-vault","self_install":{"platforms":["windows"],"args":["self-upgrade","{version}"],"launcher":"%LOCALAPPDATA%\\hive\\bin\\hive.cmd"}}}]}`
	c, err := Load(writeCatalog(t, ok))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if si := c.Tools[0].Source.SelfInstall; si == nil || !c.Tools[0].Source.SelfInstalls("windows") || c.Tools[0].Source.SelfInstalls("linux") {
		t.Fatalf("self_install = %+v, want it on windows only", si)
	}
	nb := strings.Replace(ok, `"args":`, `"no_build":["cryptography"],"args":`, 1)
	if c, err := Load(writeCatalog(t, nb)); err != nil || !slices.Equal(c.Tools[0].Source.SelfInstall.NoBuild, []string{"cryptography"}) {
		t.Fatalf("no_build: Load = %+v, %v; want [cryptography]", c.Tools, err)
	}
	for name, tc := range map[string]struct{ from, to, want string }{
		// An install with no version in it is an upgrade to whatever is newest:
		// unattended, which hive ADR-020 rules out.
		"no {version}":        {`"{version}"`, `"latest"`, "{version}"},
		"no args":             {`"args":["self-upgrade","{version}"]`, `"args":[]`, "args"},
		"a misspelt platform": {`["windows"]`, `["win"]`, "win"},
		"no platform":         {`"platforms":["windows"],`, ``, "platforms"},
		// Without it the probe has nowhere to look but PATH, where a uv copy
		// answers for an install that never happened (#2991).
		"no launcher":   {`,"launcher":"%LOCALAPPDATA%\\hive\\bin\\hive.cmd"`, ``, "launcher"},
		"not a uv tool": {`"type":"uv-tool"`, `"type":"npm"`, "uv-tool"},
		// The entry's own platforms exclude the OS, so the field never runs.
		// An empty name would reach uv as a bare --no-build-package.
		"an empty no_build name": {`"args":`, `"no_build":[""],"args":`, "no_build"},
		"an OS the tool skips":   {`"package":"hive-vault",`, `"package":"hive-vault","platforms":["linux"],`, "windows"},
	} {
		t.Run(name, func(t *testing.T) {
			bad := strings.Replace(ok, tc.from, tc.to, 1)
			if bad == ok {
				t.Fatalf("the fixture has no %s", tc.from)
			}
			if _, err := Load(writeCatalog(t, bad)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load = %v, want an error naming %q", err, tc.want)
			}
		})
	}
}

// The launcher expands %NAME% from the environment, and a name that is unset
// leaves no path at all rather than a guess.
func TestSelfInstall_LauncherPath(t *testing.T) {
	si := SelfInstall{Launcher: `%LOCALAPPDATA%\hive\bin\hive.cmd`}
	env := map[string]string{"LOCALAPPDATA": `C:\Users\u\AppData\Local`}
	if got, want := si.LauncherPath(func(k string) string { return env[k] }), `C:\Users\u\AppData\Local\hive\bin\hive.cmd`; got != want {
		t.Errorf("LauncherPath = %q, want %q", got, want)
	}
	if got := si.LauncherPath(func(string) string { return "" }); got != "" {
		t.Errorf("LauncherPath with LOCALAPPDATA unset = %q, want \"\"", got)
	}
}
