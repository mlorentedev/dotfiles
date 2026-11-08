package agent

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Fruits is the canonical 64-entry dictionary of lowercase fruit names used to
// build deterministic, human-memorable ephemeral worker identifiers of the form
// `meeseek-<role>-<fruit>`.
var Fruits = [64]string{
	"apple", "apricot", "avocado", "banana",
	"blackberry", "blueberry", "boysenberry", "cantaloupe",
	"cherry", "clementine", "coconut", "cranberry",
	"currant", "date", "dragonfruit", "durian",
	"elderberry", "feijoa", "fig", "gooseberry",
	"grape", "grapefruit", "guava", "honeydew",
	"huckleberry", "jabuticaba", "jackfruit", "jujube",
	"kiwi", "kumquat", "lemon", "lime",
	"loganberry", "longan", "loquat", "lychee",
	"mandarin", "mango", "mangosteen", "marionberry",
	"melon", "mulberry", "nectarine", "olive",
	"orange", "papaya", "passionfruit", "peach",
	"pear", "persimmon", "pineapple", "plum",
	"pomegranate", "pomelo", "quince", "rambutan",
	"raspberry", "redcurrant", "salak", "sapote",
	"soursop", "starfruit", "strawberry", "tangerine",
}

var fruitSet = func() map[string]bool {
	m := make(map[string]bool, len(Fruits))
	for _, f := range Fruits {
		m[f] = true
	}
	return m
}()

// FruitDeck draws fruits sequentially without replacement from a seeded
// permutation of the 64-fruit dictionary, reshuffling only after all 64 have
// been consumed. This guarantees zero collisions across up to 64 worker
// dispatches in a session.
type FruitDeck struct {
	mu    sync.Mutex
	rng   *rand.Rand
	order [64]int
	pos   int
}

// NewFruitDeck creates a deck seeded with seed.
func NewFruitDeck(seed int64) *FruitDeck {
	d := &FruitDeck{
		rng: rand.New(rand.NewSource(seed)),
	}
	d.shuffle()
	return d
}

func (d *FruitDeck) shuffle() {
	for i := range d.order {
		d.order[i] = i
	}
	d.rng.Shuffle(len(d.order), func(i, j int) {
		d.order[i], d.order[j] = d.order[j], d.order[i]
	})
	d.pos = 0
}

// NextWorkerID returns `meeseek-<role>-<fruit>` using the next fruit in the
// permutation.
func (d *FruitDeck) NextWorkerID(role string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pos >= len(d.order) {
		d.shuffle()
	}
	fruit := Fruits[d.order[d.pos]]
	d.pos++
	return fmt.Sprintf("meeseek-%s-%s", strings.ToLower(strings.TrimSpace(role)), fruit)
}

var defaultDeck = NewFruitDeck(time.Now().UnixNano())

// NextWorkerID draws the next `meeseek-<role>-<fruit>` from the process-wide deck.
func NextWorkerID(role string) string {
	return defaultDeck.NextWorkerID(role)
}

// ParseWorkerID parses a canonical `meeseek-<role>-<fruit>` identifier (singular
// prefix `meeseek-`, lowercase `[a-z]+` role, and a known fruit from Fruits).
func ParseWorkerID(id string) (role, fruit string, ok bool) {
	if !strings.HasPrefix(id, "meeseek-") {
		return "", "", false
	}
	rest := strings.TrimPrefix(id, "meeseek-")
	role, fruit, found := strings.Cut(rest, "-")
	if !found || role == "" || fruit == "" || strings.Contains(fruit, "-") {
		return "", "", false
	}
	for i := 0; i < len(role); i++ {
		if role[i] < 'a' || role[i] > 'z' {
			return "", "", false
		}
	}
	if !fruitSet[fruit] {
		return "", "", false
	}
	return role, fruit, true
}
