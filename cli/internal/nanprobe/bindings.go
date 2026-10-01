package nanprobe

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
	"github.com/mlorentedev/dotfiles/cli/internal/nanquota"
)

// serviceAPIs names the routing map's services that are not chat models. The
// service key IS the API: `services.embeddings` is called on /embeddings. A
// service missing here is probed as chat, and a non-chat one then reports a
// refusal that names it, which is loud rather than silently unprobed.
var serviceAPIs = map[string]API{"embeddings": EmbeddingsAPI, "rerank": RerankAPI}

// Bindings records, for each bound model, the files that bind it (so a dead
// model's report names every file to change) and the API it is called through.
type Bindings struct {
	Files map[string][]string
	APIs  map[string]API
}

func newBindings() Bindings {
	return Bindings{Files: map[string][]string{}, APIs: map[string]API{}}
}

// Targets lists the bound models, sorted, each with its API (chat unless the
// map declares it as a non-chat service).
func (b Bindings) Targets() []Target {
	models := make([]string, 0, len(b.Files))
	for m := range b.Files {
		models = append(models, m)
	}
	sort.Strings(models)
	out := make([]Target, 0, len(models))
	for _, m := range models {
		api := b.APIs[m]
		if api == "" {
			api = ChatAPI
		}
		out = append(out, Target{Model: m, API: api})
	}
	return out
}

func (b Bindings) add(model, file string) {
	for _, f := range b.Files[model] {
		if f == file {
			return
		}
	}
	b.Files[model] = append(b.Files[model], file)
}

// Collect gathers every model bound to pool: the routing map's own bindings
// (tiers, chains, services), then every pin harness/model-pins.json declares for
// that pool.
//
// The map is the floor, since a pin that resolves names a model the map already
// routes. The pins add the files a reader has to edit. A deployed-scope site
// (under $HOME) that is absent is not an error, because a CI runner has none of
// them. It is returned as a note so the report says what was not read, rather
// than implying it was.
//
// A pin whose locator matches nothing IS an error: zero values would read as
// zero dead models, which is the false "all clear" model-pins.json exists to
// prevent.
func Collect(m map[string]any, pins *harness.ModelPins, repoRoot, home, pool string) (Bindings, []string, error) {
	b := newBindings()
	for _, id := range nanquota.BoundModels(m, pool) {
		b.add(id, harness.ModelMapFile)
	}
	services, _ := m["services"].(map[string]any)
	for name, v := range services {
		svc, _ := v.(map[string]any)
		id, _ := svc["model"].(string)
		if api, ok := serviceAPIs[name]; ok && svc["pool"] == pool && id != "" {
			b.APIs[id] = api
		}
	}

	var notes []string
	for _, site := range pins.Sites {
		path := sitePath(site, repoRoot, home)
		var wanted []harness.Pin
		for _, p := range site.Pins {
			if p.Pool == pool {
				wanted = append(wanted, p)
			}
		}
		if len(wanted) == 0 {
			continue
		}
		content, err := os.ReadFile(path) // #nosec G304 -- path is declared by the repository's own registry
		if err != nil {
			if os.IsNotExist(err) && site.Scope == "deployed" {
				notes = append(notes, fmt.Sprintf("%s is not present here, so its pins were not read", site.File))
				continue
			}
			return Bindings{}, nil, fmt.Errorf("read %s: %w", site.File, err)
		}
		for _, p := range wanted {
			values, err := harness.Extract(p, content)
			if err != nil {
				return Bindings{}, nil, fmt.Errorf("%s: %w", site.File, err)
			}
			for _, v := range values {
				if !harness.Spelled(p, v) {
					// Misspelled for its site, so it cannot be normalised to a
					// NaN id. `dotf doctor` reports it; probing a guess would not.
					continue
				}
				_, id, _ := strings.Cut(harness.Normalize(p, v), ":")
				b.add(id, site.File)
			}
		}
	}
	return b, notes, nil
}

func sitePath(site harness.PinSite, repoRoot, home string) string {
	if site.Scope == "deployed" {
		f := strings.NewReplacer("${HOME}", home, "$HOME", home).Replace(site.File)
		return filepath.FromSlash(f)
	}
	return filepath.Join(repoRoot, filepath.FromSlash(site.File))
}
