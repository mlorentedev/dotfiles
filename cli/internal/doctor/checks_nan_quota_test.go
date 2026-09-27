package doctor

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

const nanTestKey = "sk-nan-test-DO-NOT-PRINT"

const nanRegistryAge = "version: 1\nsecrets:\n" +
	"  - {id: NAN_API_KEY, plane: app, backend: age, age: nan.key, expose: {env: [NAN_API_KEY, HIVE_WORKER_API_KEY]}}\n"

const nanRegistryBW = "version: 1\nsecrets:\n" +
	"  - {id: NAN_API_KEY, plane: app, backend: bw, bw: {item: nan-api-key, field: api-key}, expose: {env: [NAN_API_KEY]}}\n"

// nanUsage415 is /v1/usage with qwen3.8-flash at 83% of its 500M quota.
const nanUsage415 = `{"start_date":"2026-06-01","end_date":"2026-06-23","totals":{"by_model":[` +
	`{"model":"qwen3.8-flash","total_tokens":415000000},{"model":"deepseek-v4-flash","total_tokens":300000000}]},"has_more":false}`

const nanServed = `{"data":[{"id":"deepseek-v4-flash"},{"id":"qwen3.8-flash"},{"id":"mimo-v2.5"},` +
	`{"id":"qwen3.6"},{"id":"qwen3-embedding"},{"id":"rerank"},{"id":"glm5.3-flash"}]}`

// nanCfg builds a dotfiles tree holding the registry, the repo's real model map
// and schema with services.rerank set to rerankID, and the repo's quota table
// with its checked date replaced.
func nanCfg(t *testing.T, registry, rerankID, checked string) *Config {
	t.Helper()
	repo := filepath.Join("..", "..", "..")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "secrets", "registry.yaml"), registry)

	schema, err := os.ReadFile(filepath.Join(repo, "harness", "model-map.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "harness", "model-map.schema.json"), string(schema))

	var m map[string]any
	readJSON(t, filepath.Join(repo, "harness", "model-map.json"), &m)
	m["services"].(map[string]any)["rerank"].(map[string]any)["model"] = rerankID
	writeJSON(t, filepath.Join(dir, "harness", "model-map.json"), m)

	var q map[string]any
	readJSON(t, filepath.Join(repo, "harness", "nan-quotas.json"), &q)
	q["checked"] = checked
	writeJSON(t, filepath.Join(dir, "harness", "nan-quotas.json"), q)
	return &Config{DotfilesDir: dir}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(raw))
}

// nanSys answers /v1/usage and /v1/models from the given bodies and records the
// Authorization header it was sent.
func nanSys(usage, models string, gotAuth *string) *System {
	sys := newSys(nil, nil, nil)
	sys.ResolveSecret = resolvesTo(nanTestKey)
	sys.HTTPGetBody = func(url string, h map[string]string) (int, []byte, error) {
		if gotAuth != nil {
			*gotAuth = h["Authorization"]
		}
		if strings.Contains(url, "/usage") {
			return 200, []byte(usage), nil
		}
		return 200, []byte(models), nil
	}
	return sys
}

func runNaNQuota(sys *System, cfg *Config) (string, int) {
	var buf bytes.Buffer
	rep := capture(&buf)
	checkNaNQuota(sys, cfg, rep)
	return buf.String(), rep.Failures()
}

// AC2's shape against fixtures: every bound metered model reported with used,
// quota and percent; 83% WARNs without failing doctor.
func TestCheckNaNQuota_ReportsEachBoundModel(t *testing.T) {
	var auth string
	out, fails := runNaNQuota(nanSys(nanUsage415, nanServed, &auth), nanCfg(t, nanRegistryAge, "rerank", "2026-06-10"))

	if fails != 0 {
		t.Errorf("failures = %d, want 0:\n%s", fails, out)
	}
	for _, want := range []string{"qwen3.8-flash: 415.0M / 500.0M tokens this month (83%)", "deepseek-v4-flash: 300.0M / 3000.0M", "rerank: unmetered"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if auth != "Bearer "+nanTestKey {
		t.Errorf("the request did not carry the resolved key as a bearer token")
	}
}

// AC3: a bound id NaN does not serve fails doctor, by name.
func TestCheckNaNQuota_FailsOnAnUnservedBinding(t *testing.T) {
	out, fails := runNaNQuota(nanSys(nanUsage415, nanServed, nil), nanCfg(t, nanRegistryAge, "qwen3-rerank", "2026-06-10"))
	if fails != 1 || !strings.Contains(out, "qwen3-rerank is bound in model-map.json but NaN does not serve it") {
		t.Errorf("failures = %d, want 1 naming qwen3-rerank:\n%s", fails, out)
	}
}

// AC4: NaN unreachable is a SKIP with a reason, never a failure.
func TestCheckNaNQuota_SkipsWhenNaNIsUnreachable(t *testing.T) {
	sys := newSys(nil, nil, nil)
	sys.ResolveSecret = resolvesTo(nanTestKey)
	out, fails := runNaNQuota(sys, nanCfg(t, nanRegistryAge, "rerank", "2026-06-10"))
	if fails != 0 || !strings.Contains(out, "NaN is unreachable") {
		t.Errorf("failures = %d, want a SKIP naming the outage:\n%s", fails, out)
	}
}

// A bw-backed key is not resolved while the serve daemon cannot serve it: that
// read is the ~1.5s shellout that made shell startup hang (BUG-080).
func TestCheckNaNQuota_DoesNotResolveABWKeyWithoutTheDaemon(t *testing.T) {
	sys := newSys(nil, nil, nil)
	resolved := false
	sys.ResolveSecret = func(secrets.Entry) (string, error) { resolved = true; return nanTestKey, nil }
	out, fails := runNaNQuota(sys, nanCfg(t, nanRegistryBW, "rerank", "2026-06-10"))
	if fails != 0 || resolved || !strings.Contains(out, "bw serve daemon") {
		t.Errorf("failures = %d, resolved = %v, want a SKIP naming the daemon:\n%s", fails, resolved, out)
	}
}

// CI runs doctor with no NaN key: the section SKIPs and makes no call.
func TestCheckNaNQuota_SkipsWithoutAKey(t *testing.T) {
	cases := map[string]string{
		"not provisioned":             nanRegistryAge,
		"bw-backed, daemon not ready": nanRegistryBW,
		"not declared at all":         "version: 1\nsecrets: []\n",
	}
	for name, registry := range cases {
		t.Run(name, func(t *testing.T) {
			sys := newSys(nil, nil, nil) // default resolver: secret absent
			called := false
			sys.HTTPGetBody = func(string, map[string]string) (int, []byte, error) {
				called = true
				return 0, nil, errors.New("unexpected")
			}
			out, fails := runNaNQuota(sys, nanCfg(t, registry, "rerank", "2026-06-10"))
			if fails != 0 || called || !strings.Contains(out, "not being watched") {
				t.Errorf("failures = %d, called = %v, want a SKIP and no request:\n%s", fails, called, out)
			}
		})
	}
}

// AC5: no branch prints the key, including the ones that echo what NaN sent.
func TestCheckNaNQuota_NeverPrintsTheKey(t *testing.T) {
	branches := map[string]func(string, map[string]string) (int, []byte, error){
		"rejected": func(string, map[string]string) (int, []byte, error) {
			return 401, []byte(`{"error":"invalid key ` + nanTestKey + `"}`), nil
		},
		"unparseable": func(string, map[string]string) (int, []byte, error) {
			return 200, []byte("<html>" + nanTestKey), nil
		},
		"transport": func(url string, _ map[string]string) (int, []byte, error) {
			return 0, nil, errors.New("GET " + url + ": dial tcp: connection refused")
		},
	}
	for name, get := range branches {
		t.Run(name, func(t *testing.T) {
			sys := newSys(nil, nil, nil)
			sys.ResolveSecret = resolvesTo(nanTestKey)
			sys.HTTPGetBody = get
			out, _ := runNaNQuota(sys, nanCfg(t, nanRegistryAge, "rerank", "2026-06-10"))
			if strings.Contains(out, nanTestKey) || strings.Contains(out, "sk-nan") {
				t.Errorf("the key reached the report:\n%s", out)
			}
		})
	}
}

func TestCheckNaNQuota_WarnsOnAStaleTable(t *testing.T) {
	out, fails := runNaNQuota(nanSys(nanUsage415, nanServed, nil), nanCfg(t, nanRegistryAge, "rerank", "2026-01-01"))
	if fails != 0 || !strings.Contains(out, "harness/nan-quotas.json was read 2026-01-01") {
		t.Errorf("want a WARN naming the stale date:\n%s", out)
	}
}
