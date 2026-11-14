package mem

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// agyHookDir is where the first-invocation markers live, under the OS temp
// directory: a marker is only worth keeping for the length of a conversation.
const agyHookDir = "dotf-agy-session-start"

// agyMarkerTTL is how long a marker outlives its conversation before the next
// first invocation removes it.
const agyMarkerTTL = 7 * 24 * time.Hour

// AgyHookPayload is the part of agy's hook JSON the session line reads.
type AgyHookPayload struct {
	ConversationID string `json:"conversationId"`
	Cwd            string `json:"cwd"`
}

// ParseAgyHook reads agy's hook JSON; a field absent or malformed is "".
func ParseAgyHook(payload []byte) AgyHookPayload {
	var p AgyHookPayload
	_ = json.Unmarshal(payload, &p)
	return p
}

// FirstInvocation reports whether this is the first PreInvocation of the
// conversation, and marks it so the next one is not. agy has no session-start
// event, only PreInvocation, which fires before every turn; the marker is
// created with O_EXCL, so two concurrent hooks cannot both be first. An empty
// id is never first: a hook that cannot tell conversations apart must say
// nothing rather than repeat itself on every turn.
func FirstInvocation(stateDir, conversationID string) (bool, error) {
	if conversationID == "" {
		return false, nil
	}
	dir := filepath.Join(stateDir, agyHookDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}
	pruneMarkers(dir, time.Now())
	f, err := os.OpenFile(filepath.Join(dir, markerName(conversationID)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- name is sanitised by markerName
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, f.Close()
}

// markerName keeps the characters a conversation id is made of and replaces the
// rest, so an id can never name a path outside the state directory.
func markerName(id string) string {
	name := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return r
		}
		return '_'
	}, id)
	return name[:min(len(name), 96)]
}

// pruneMarkers removes the markers of conversations that ended long ago. A
// failure to remove one is ignored: it costs a few bytes, not the hook.
func pruneMarkers(dir string, now time.Time) {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > agyMarkerTTL {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// AgyInjection is the hook's stdout: agy requires a JSON object, and injects
// text as a systemMessage step that persists in the conversation. "" injects
// nothing, as `{}`.
func AgyInjection(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "{}\n", nil
	}
	step := map[string]any{"systemMessage": map[string]string{"systemMessage": text}}
	out, err := json.Marshal(map[string]any{"injectSteps": []any{step}})
	return string(out) + "\n", err
}
