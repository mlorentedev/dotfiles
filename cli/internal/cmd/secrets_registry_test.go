package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// fakeBW is a map-backed secrets.BWReader ("item/field" → value) so command tests
// resolve bw secrets with no unlocked Bitwarden.
type fakeBW map[string]string

func (f fakeBW) Field(item, field string) (string, error) {
	if v, ok := f[item+"/"+field]; ok {
		return v, nil
	}
	return "", fmt.Errorf("fake bw: no %s/%s", item, field)
}

// useBwReader points the bwReader seam at a fake for the duration of a test.
func useBwReader(t *testing.T, r secrets.BWReader) {
	t.Helper()
	old := bwReader
	bwReader = r
	t.Cleanup(func() { bwReader = old })
}

const testRegistry = `
version: 1
secrets:
  - {id: nan-api-key, plane: app, backend: age, age: nan.api-key, expose: {env: NAN_API_KEY}}
  - {id: x-twitter, plane: app, backend: age, expose: {env: {X_API_KEY: {age: x.api-key}, X_BEARER_TOKEN: {age: x.bearer-token}}}}
  - {id: kubelab-kubeconfig, plane: infra, backend: age, age: kubelab.kubeconfig, expose: {file: {var: KUBECONFIG, path: "~/.kube/c", mode: "0600"}}}
`

// useTempRegistry points both the read seam (registryPath) and the write seam
// (registryWritePath) at one fixture for the duration of a test, so command tests
// exercise migrate's flip without needing a real checkout.
func useTempRegistry(t *testing.T, body string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "registry.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	oldRead, oldWrite := registryPath, registryWritePath
	registryPath = func() string { return p }
	registryWritePath = func() (string, error) { return p, nil }
	t.Cleanup(func() { registryPath, registryWritePath = oldRead, oldWrite })
}

func TestSecretsLs_ListsIdsAndVars_NoValues(t *testing.T) {
	useTempRegistry(t, testRegistry)
	var out bytes.Buffer
	cmd := newSecretsLsCmd()
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"nan-api-key", "x-twitter", "kubelab-kubeconfig", "NAN_API_KEY", "KUBECONFIG"} {
		if !strings.Contains(got, want) {
			t.Errorf("ls output missing %q:\n%s", want, got)
		}
	}
}

func TestSecretsShow_SingleEnv_ScrubbedNoTrailingNewline(t *testing.T) {
	useTempRegistry(t, testRegistry)
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return []byte("the-value\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })

	var out bytes.Buffer
	cmd := newSecretsShowCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"nan-api-key"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "the-value" {
		t.Errorf("show = %q, want %q (no trailing newline)", out.String(), "the-value")
	}
}

func TestSecretsShow_RejectsFileAndMultiAndUnknown(t *testing.T) {
	useTempRegistry(t, testRegistry)
	for _, id := range []string{"x-twitter", "kubelab-kubeconfig", "nope"} {
		cmd := newSecretsShowCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{id})
		if err := cmd.Execute(); err == nil {
			t.Errorf("show %q: expected an error (must point at run / unknown)", id)
		}
	}
}

func TestSecretsShow_ResolvesBwBackend(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-one, plane: app, backend: bw, bw: {item: bw-item, field: password}, expose: {env: B_KEY}}\n")
	useBwReader(t, fakeBW{"bw-item/password": "bw-secret"})
	var out bytes.Buffer
	cmd := newSecretsShowCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"bw-one"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("show bw-one: %v", err)
	}
	if out.String() != "bw-secret" {
		t.Errorf("show bw = %q, want %q (resolved from Bitwarden, no trailing newline)", out.String(), "bw-secret")
	}
}

// TestSecretsRun_ResolvesBwBackend exercises run's wiring: registry → secretLoader →
// EnvFor → bwResolver → the bw reader. --only selects the bw secret by id.
func TestSecretsRun_ResolvesBwBackend(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-foo, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: FOO}}\n")
	useBwReader(t, fakeBW{"it/password": "bw-secret-value"})

	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	sel, err := resolveOnly(reg, "bw-foo") // --only by id → the secret's vars
	if err != nil {
		t.Fatal(err)
	}
	env, err := buildChildEnv(reg, sel)
	if err != nil {
		t.Fatalf("buildChildEnv: %v", err)
	}
	found := false
	for _, kv := range env {
		if kv == "FOO=bw-secret-value" {
			found = true
		}
	}
	if !found {
		t.Errorf("run did not inject the bw secret FOO=bw-secret-value into the child env")
	}
}

func TestStripBackendAuth(t *testing.T) {
	in := []string{"PATH=/bin", "BW_SESSION=tok", "FOO=bar", "OP_SERVICE_ACCOUNT_TOKEN=x", "BW_CLIENTSECRET=y", "EMPTY="}
	got := strings.Join(stripBackendAuth(in), " ")
	for _, banned := range []string{"BW_SESSION", "OP_SERVICE_ACCOUNT_TOKEN", "BW_CLIENTSECRET"} {
		if strings.Contains(got, banned) {
			t.Errorf("%s must be stripped from the child env, got: %s", banned, got)
		}
	}
	for _, kept := range []string{"PATH=/bin", "FOO=bar", "EMPTY="} {
		if !strings.Contains(got, kept) {
			t.Errorf("non-auth var %q must be kept, got: %s", kept, got)
		}
	}
}

// TestBuildChildEnv_StripsBackendAuth proves the resolved secret reaches the child
// while the backend unlock token (BW_SESSION) does not — the child gets what it was
// granted, never the key that opens the whole vault.
func TestBuildChildEnv_StripsBackendAuth(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-foo, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: FOO}}\n")
	useBwReader(t, fakeBW{"it/password": "granted-value"})
	t.Setenv("BW_SESSION", "unlock-token-must-not-leak")

	reg, err := loadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	env, err := buildChildEnv(reg, nil)
	if err != nil {
		t.Fatalf("buildChildEnv: %v", err)
	}
	var grantedFound bool
	for _, kv := range env {
		if strings.HasPrefix(kv, "BW_SESSION=") {
			t.Error("BW_SESSION must NOT be passed to the child environment")
		}
		if kv == "FOO=granted-value" {
			grantedFound = true
		}
	}
	if !grantedFound {
		t.Error("the granted secret FOO=granted-value must still reach the child env")
	}
}

func TestSecretsRender_SubstitutesInPlace(t *testing.T) {
	useTempRegistry(t, testRegistry)
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return []byte("rendered-value\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })

	cfg := filepath.Join(t.TempDir(), "opencode.jsonc")
	if err := os.WriteFile(cfg, []byte(`{"k":"{env:NAN_API_KEY}","h":"{env:HOME}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newSecretsRenderCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{cfg})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(cfg)
	// NAN_API_KEY is registry-mapped → substituted; HOME is not → left for runtime.
	if want := `{"k":"rendered-value","h":"{env:HOME}"}`; string(got) != want {
		t.Errorf("render = %q, want %q", got, want)
	}
}

func TestSecretsShow_EmptyValue_Errors(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n  - {id: bw-empty, plane: app, backend: bw, bw: {item: it, field: password}, expose: {env: E_KEY}}\n")
	useBwReader(t, fakeBW{"it/password": ""})
	cmd := newSecretsShowCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"bw-empty"})
	if err := cmd.Execute(); err == nil {
		t.Error("show must error when the secret resolves to an empty value")
	}
}

// render default stays non-fatal on one failing secret (placeholder left, exit 0);
// --strict turns the same real failure into a non-zero exit (#612 A2).
func TestSecretsRender_Strict_NonZeroOnFailure(t *testing.T) {
	old := ageDecryptor
	ageDecryptor = func(_, _ string) ([]byte, error) { return nil, fmt.Errorf("age: vault locked") }
	t.Cleanup(func() { ageDecryptor = old })

	render := func(args ...string) error {
		useTempRegistry(t, testRegistry)
		cfg := filepath.Join(t.TempDir(), "c.jsonc")
		if err := os.WriteFile(cfg, []byte(`{"k":"{env:NAN_API_KEY}"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		c := newSecretsRenderCmd()
		c.SetOut(io.Discard)
		c.SetErr(io.Discard)
		c.SetArgs(append(args, cfg))
		return c.Execute()
	}
	if err := render(); err != nil {
		t.Fatalf("render default must not fail on one unresolved secret: %v", err)
	}
	if err := render("--strict"); err == nil {
		t.Error("render --strict must exit non-zero when a mapped secret fails to resolve")
	}
}

func TestResolveOnly_EmptyTokens_Errors(t *testing.T) {
	reg, err := secrets.ParseRegistry([]byte(testRegistry))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveOnly(reg, ","); err == nil {
		t.Error("--only with only-empty tokens must error (it selects nothing)")
	}
	if _, err := resolveOnly(reg, "  ,  "); err == nil {
		t.Error("--only with whitespace-only tokens must error")
	}
}

func TestResolveOnly_IdSelectsAllVars_NameSelectsOne(t *testing.T) {
	reg, err := secrets.ParseRegistry([]byte(testRegistry))
	if err != nil {
		t.Fatal(err)
	}
	set, err := resolveOnly(reg, "x-twitter") // id → all its vars
	if err != nil || len(set) != 2 || !set["X_API_KEY"] || !set["X_BEARER_TOKEN"] {
		t.Errorf("id selector = %v (err %v), want {X_API_KEY,X_BEARER_TOKEN}", set, err)
	}
	set, _ = resolveOnly(reg, "X_BEARER_TOKEN") // var name → just itself
	if len(set) != 1 || !set["X_BEARER_TOKEN"] {
		t.Errorf("var selector = %v, want {X_BEARER_TOKEN}", set)
	}
	if s, _ := resolveOnly(reg, "  "); s != nil {
		t.Error("empty --only must resolve to nil (= all)")
	}
	if _, err := resolveOnly(reg, "bogus"); err == nil {
		t.Error("unknown --only token must error")
	}
}

// The ambiguity the hive daemon's ExecStart depends on, pinned.
//
// The real NAN_API_KEY secret has an ID equal to one of the two vars it exposes,
// so `--only NAN_API_KEY` is simultaneously an id token and a var token.
// Selector resolves IDs FIRST, which is what delivers HIVE_WORKER_API_KEY to
// hive's worker from a flag that names only NAN_API_KEY
// (systemd/hive.service.d/10-dotf-secrets.conf).
//
// Swapping those two loops would still pass every other test here — and would
// silently strip hive's credential, restoring the unconfigured-worker state
// CLI-042 AC9 exists to catch. That failure would surface as a daemon that runs
// and answers nothing, which is precisely the shape that went unnoticed once.
func TestResolveOnly_IdWinsWhenIdAndVarNameCollide(t *testing.T) {
	const collides = `version: 1
secrets:
  - id: NAN_API_KEY
    plane: app
    backend: bw
    bw: { item: nan-api-key, field: api-key }
    expose: { env: [NAN_API_KEY, HIVE_WORKER_API_KEY] }
`
	reg, err := secrets.ParseRegistry([]byte(collides))
	if err != nil {
		t.Fatal(err)
	}
	set, err := resolveOnly(reg, "NAN_API_KEY")
	if err != nil {
		t.Fatalf("resolveOnly: %v", err)
	}
	if len(set) != 2 || !set["NAN_API_KEY"] || !set["HIVE_WORKER_API_KEY"] {
		t.Errorf("--only NAN_API_KEY = %v, want BOTH vars: the token matches the "+
			"secret's id, and an id selects every var it exposes. Selecting only "+
			"the same-named var would leave hive's worker without a credential.", set)
	}
}

func TestSecretsVerify_ReportsStatuses_NoValues(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: age-ok, plane: app, backend: age, age: a.key, expose: {env: AGE_OK}}\n"+
		"  - {id: age-missing, plane: app, backend: age, age: gone, expose: {env: AGE_MISSING}}\n"+
		"  - {id: bw-fail, plane: app, backend: bw, bw: {item: nope, field: password}, expose: {env: BW_FAIL}}\n")
	old := ageDecryptor
	ageDecryptor = func(ageFile, _ string) ([]byte, error) {
		if strings.Contains(ageFile, "gone") {
			return nil, fmt.Errorf("%w: gone", secrets.ErrSecretAbsent)
		}
		return []byte("the-secret-value\n"), nil
	}
	t.Cleanup(func() { ageDecryptor = old })
	useBwReader(t, fakeBW{}) // every bw lookup fails

	var out bytes.Buffer
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()

	got := out.String()
	for _, want := range []string{"OK", "AGE_OK", "MISSING", "AGE_MISSING", "FAILED", "BW_FAIL"} {
		if !strings.Contains(got, want) {
			t.Errorf("verify output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "the-secret-value") {
		t.Error("verify must NOT print secret values")
	}
	if err == nil {
		t.Error("verify must exit non-zero when a secret FAILED")
	}
}

func TestSecretsVerify_ScopesById(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: a, plane: app, backend: age, age: a, expose: {env: AAA}}\n"+
		"  - {id: b, plane: app, backend: age, age: b, expose: {env: BBB}}\n")
	old := ageDecryptor
	ageDecryptor = func(string, string) ([]byte, error) { return []byte("v\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })

	var out bytes.Buffer
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"a"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("verify a: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "AAA") {
		t.Errorf("verify a must check AAA: %s", got)
	}
	if strings.Contains(got, "BBB") {
		t.Errorf("verify a must NOT check BBB: %s", got)
	}
}

func TestSecretsVerify_UnknownId_Errors(t *testing.T) {
	useTempRegistry(t, testRegistry)
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"no-such-id"})
	if err := cmd.Execute(); err == nil {
		t.Error("verify with an unknown id must error")
	}
}

// --- BUG-086 (#1004): a malformed entry must not take down the whole report ---------

// malformedRegistry has one bad secret ("broken": an unknown backend) between two good
// ones, so a test can prove the good ones still report.
const malformedRegistry = "version: 1\nsecrets:\n" +
	"  - {id: good-one, plane: app, backend: age, age: a, expose: {env: GOOD_ONE}}\n" +
	"  - {id: broken, plane: app, backend: nonsense, age: b, expose: {env: BROKEN}}\n" +
	"  - {id: good-two, plane: app, backend: age, age: c, expose: {env: GOOD_TWO}}\n"

// alwaysResolves points the age seam at a value so the well-formed entries report OK,
// isolating the registry-defect behaviour from resolution behaviour.
func alwaysResolves(t *testing.T) {
	t.Helper()
	old := ageDecryptor
	ageDecryptor = func(string, string) ([]byte, error) { return []byte("v\n"), nil }
	t.Cleanup(func() { ageDecryptor = old })
}

func runVerify(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newSecretsVerifyCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	// Explicit, because Go evaluates return operands left to right: `return
	// out.String(), cmd.Execute()` would read the buffer before the command wrote it.
	err := cmd.Execute()
	return out.String(), err
}

// TestSecretsVerify_MalformedEntryDoesNotHideTheRest is BUG-086's AC1.
//
// Before this, ParseRegistry failed on the first bad secret, so ONE typo made
// `dotf secrets verify` unable to report anything at all — a health check taken down by
// the class of breakage it exists to find, and by an entry the caller never asked about.
func TestSecretsVerify_MalformedEntryDoesNotHideTheRest(t *testing.T) {
	useTempRegistry(t, malformedRegistry)
	alwaysResolves(t)

	out, err := runVerify(t)

	for _, want := range []string{"OK", "GOOD_ONE", "GOOD_TWO", "FAILED", "broken"} {
		if !strings.Contains(out, want) {
			t.Errorf("verify output missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "registry:") {
		t.Errorf("a malformed entry should be reported as a registry defect:\n%s", out)
	}
	if err == nil {
		t.Error("verify must exit non-zero when an entry is malformed")
	}
}

// TestSecretsVerify_ScopedAwayFromDefect is AC2: an entry the caller did not name is
// neither validated nor resolved, so a defect elsewhere cannot fail a scoped check.
func TestSecretsVerify_ScopedAwayFromDefect(t *testing.T) {
	useTempRegistry(t, malformedRegistry)
	alwaysResolves(t)

	out, err := runVerify(t, "good-one")

	if err != nil {
		t.Errorf("a scoped verify must not fail on an out-of-scope defect: %v\n%s", err, out)
	}
	if strings.Contains(out, "broken") {
		t.Errorf("an out-of-scope defect must not be reported:\n%s", out)
	}
	if !strings.Contains(out, "GOOD_ONE") {
		t.Errorf("the named secret should still be reported:\n%s", out)
	}
	if strings.Contains(out, "GOOD_TWO") {
		t.Errorf("scoping must exclude unnamed secrets:\n%s", out)
	}
}

// TestSecretsVerify_ScopedToTheDefect: naming the malformed entry must report it, not
// claim it is unknown. It was excluded from the registry precisely because it is the
// thing being asked about, so the lookup has to consult the defects too.
func TestSecretsVerify_ScopedToTheDefect(t *testing.T) {
	useTempRegistry(t, malformedRegistry)
	alwaysResolves(t)

	out, err := runVerify(t, "broken")

	if err == nil {
		t.Errorf("verify must exit non-zero when the named entry is malformed:\n%s", out)
	}
	if strings.Contains(out, "unknown id") {
		t.Errorf("a malformed entry must be reported as FAILED, not as unknown:\n%s", out)
	}
	if !strings.Contains(out, "FAILED") || !strings.Contains(out, "broken") {
		t.Errorf("the named defect should be a FAILED row:\n%s", out)
	}
}

// TestSecretsVerify_DuplicateIdBlamesTheSecondOne: the FIRST definition is valid and
// must still verify; only the later one is the defect. A rejected secret must not
// reserve its id, or it would make the valid one look like the duplicate.
func TestSecretsVerify_DuplicateIdBlamesTheSecondOne(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: dup, plane: app, backend: age, age: a, expose: {env: FIRST}}\n"+
		"  - {id: dup, plane: app, backend: age, age: b, expose: {env: SECOND}}\n")
	alwaysResolves(t)

	out, err := runVerify(t)

	if err == nil {
		t.Errorf("a duplicate id must fail verify:\n%s", out)
	}
	if !strings.Contains(out, "FIRST") {
		t.Errorf("the first definition is valid and must still be verified:\n%s", out)
	}
	if !strings.Contains(out, "duplicate secret id") {
		t.Errorf("the defect should name the duplication:\n%s", out)
	}
}

// TestSecretsVerify_StructuralFailureStillAborts: the partial door is for "this ENTRY is
// wrong", not for "the document is unreadable". With no parseable version there is
// nothing to report per-secret, so failing loudly is correct.
func TestSecretsVerify_StructuralFailureStillAborts(t *testing.T) {
	useTempRegistry(t, "version: 99\nsecrets: []\n")
	if _, err := runVerify(t); err == nil {
		t.Error("an unsupported registry version must still abort, not degrade to a report")
	}
}

// TestSecretsVerify_ScopedToDuplicateIdReportsBothHalves is CodeRabbit's Major finding
// on #1020, and it was right.
//
// A rejected secret does not reserve its id, so a duplicate id leaves TWO things sharing
// one name: the first definition, which validated and is in the registry, and the second,
// which is a defect. Scoping to that id must report both — the original code matched the
// defect and short-circuited, so `verify dup` reported the defect and silently skipped
// the entry that actually resolves, which is the half the caller most needs.
func TestSecretsVerify_ScopedToDuplicateIdReportsBothHalves(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: dup, plane: app, backend: age, age: a, expose: {env: VALID_HALF}}\n"+
		"  - {id: dup, plane: app, backend: age, age: b, expose: {env: DEFECT_HALF}}\n")
	alwaysResolves(t)

	out, err := runVerify(t, "dup")

	if !strings.Contains(out, "VALID_HALF") {
		t.Errorf("the valid definition sharing the id must still be resolved:\n%s", out)
	}
	if !strings.Contains(out, "duplicate secret id") {
		t.Errorf("the defect sharing the id must still be reported:\n%s", out)
	}
	if err == nil {
		t.Errorf("a defect in scope must still exit non-zero:\n%s", out)
	}
}

// TestSecretsVerify_SeveralDefectsOnOneId: defects are keyed to a slice, so two bad
// definitions of one id are both reported rather than one overwriting the other.
func TestSecretsVerify_SeveralDefectsOnOneId(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: trip, plane: app, backend: age, age: a, expose: {env: FIRST_OK}}\n"+
		"  - {id: trip, plane: app, backend: age, age: b, expose: {env: SECOND}}\n"+
		"  - {id: trip, plane: app, backend: age, age: c, expose: {env: THIRD}}\n")
	alwaysResolves(t)

	out, _ := runVerify(t, "trip")

	if n := strings.Count(out, "duplicate secret id"); n != 2 {
		t.Errorf("want both duplicate definitions reported, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "FIRST_OK") {
		t.Errorf("the one valid definition must still resolve:\n%s", out)
	}
}

// TestSecretsVerify_TokenThatIsBothADefectIdAndAValidVar is the adversarial review's
// second Minor (THEORETICAL) on BUG-086, pinned as behaviour rather than left implicit.
//
// One token can name two different secrets: a malformed entry whose ID is `SHARED`, and
// a well-formed entry that EXPOSES a var called `SHARED`. `scopeVerify` matches the
// first against the defect table and the second through reg.Selector, so both report.
//
// That is correct — they are genuinely two secrets and the caller named both — but it
// was undocumented and untested, which is the reviewer's point. Reporting only one of
// them would hide a real answer behind a name collision.
func TestSecretsVerify_TokenThatIsBothADefectIdAndAValidVar(t *testing.T) {
	useTempRegistry(t, "version: 1\nsecrets:\n"+
		"  - {id: SHARED, plane: app, backend: nonsense, age: a, expose: {env: OTHER}}\n"+
		"  - {id: exposer, plane: app, backend: age, age: b, expose: {env: SHARED}}\n")
	alwaysResolves(t)

	out, err := runVerify(t, "SHARED")

	if !strings.Contains(out, "registry:") {
		t.Errorf("the defect whose id is SHARED must be reported:\n%s", out)
	}
	if !strings.Contains(out, "OK") || !strings.Contains(out, "SHARED") {
		t.Errorf("the well-formed secret exposing a var named SHARED must resolve:\n%s", out)
	}
	if err == nil {
		t.Errorf("a defect in scope must still exit non-zero:\n%s", out)
	}
}
