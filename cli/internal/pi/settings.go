package pi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

// SettingsRel is the repo's copy of pi's settings.json, relative to the
// checkout.
const SettingsRel = "ai/pi/settings.json"

// SettingsPath is pi's settings.json under home.
func SettingsPath(home string) string {
	return filepath.Join(home, ".pi", "agent", "settings.json")
}

// SettingsState is how pi's settings.json differs from the repo's copy.
//
// pi rewrites the file at runtime: the model picked in its TUI, the theme,
// lastChangelogVersion. So the repo's copy is a seed, written only when the
// file is absent, and once the file exists the repo owns one key of it,
// enabledModels, which is made equal to the repo's list (AI-032, #1247). A
// merge would not do: it unions lists, so a model the repo drops would stay.
type SettingsState struct {
	Missing      bool // absent: seed the repo's copy whole
	ModelsDiffer bool // present, and enabledModels is not the repo's list

	models any // the repo's enabledModels, when ModelsDiffer
}

// Changes counts what ApplySettings would write: at most one.
func (s SettingsState) Changes() int {
	if s.Missing || s.ModelsDiffer {
		return 1
	}
	return 0
}

// InspectSettings compares dst with the repo's copy at src. A dst that is not
// a JSON object is an error, never a seed: it may hold the user's choices.
func InspectSettings(src, dst string) (SettingsState, error) {
	repo, err := readObject(src)
	if err != nil {
		return SettingsState{}, fmt.Errorf("pi settings source %s: %w", src, err)
	}
	if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) {
		return SettingsState{Missing: true}, nil
	}
	have, err := readObject(dst)
	if err != nil {
		return SettingsState{}, fmt.Errorf("pi settings %s: %w", dst, err)
	}
	want, managed := repo["enabledModels"]
	if !managed || reflect.DeepEqual(have["enabledModels"], want) {
		return SettingsState{}, nil
	}
	return SettingsState{ModelsDiffer: true, models: want}, nil
}

// ApplySettings writes what st found: the repo's copy byte for byte when dst
// is absent, or dst with enabledModels replaced, every other key and the
// file's mode kept.
func ApplySettings(src, dst string, st SettingsState) error {
	switch {
	case st.Missing:
		raw, err := os.ReadFile(src) //nolint:gosec // the checkout's own file
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil { //nolint:gosec // pi's own config dir, as pi creates it
			return err
		}
		return secrets.AtomicWriteMode(dst, raw, 0o644)
	case st.ModelsDiffer:
		have, err := readObject(dst)
		if err != nil {
			return fmt.Errorf("pi settings %s: %w", dst, err)
		}
		fi, err := os.Stat(dst)
		if err != nil {
			return err
		}
		have["enabledModels"] = st.models
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(have); err != nil {
			return err
		}
		return secrets.AtomicWriteMode(dst, out.Bytes(), fi.Mode().Perm())
	}
	return nil
}

// readObject decodes a JSON object, numbers kept as written.
func readObject(p string) (map[string]any, error) {
	raw, err := os.ReadFile(p) //nolint:gosec // a pi settings path
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}
	if m == nil {
		return nil, errors.New("not a JSON object: null")
	}
	return m, nil
}
