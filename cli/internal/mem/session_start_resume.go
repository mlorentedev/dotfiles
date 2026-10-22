package mem

import (
	"fmt"
	"os"
	"strings"
)

// resumeInjector is the session-start-config.json flag that turns the resume
// block off. Absent means on, like every other injector.
const resumeInjector = "handoff_resume"

// sessionResume is what a SessionStart of the given source carries about this
// session's handoff thread (MEMORY-008 AC3). A session that starts fresh, was
// cleared or was compacted has lost the conversation, so it gets the full
// resume context. One that was resumed or forked still holds the conversation,
// so it gets one line saying where the thread is. An unknown source gets the
// line too: it costs a sentence, never a wrong context.
//
// It returns "" when the memory file is absent or no single thread belongs to
// this agent here: a session start never guesses at somebody else's work.
func sessionResume(source, memoryPath, key, agent string) string {
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return ""
	}
	content := string(data)
	thread, _ := SelectThread(content, key, agent)
	if thread == "" {
		return ""
	}
	switch normaliseSource(source) {
	case "startup", "clear", "compact":
		text, src := ThreadText(content, thread, memoryPath)
		if text == "" {
			return ""
		}
		out, err := RenderResume(text, src, "context")
		if err != nil {
			return ""
		}
		return out
	}
	return fmt.Sprintf("[resume] Thread %s has a handoff; `dotf mem resume --thread %s` renders it.\n", thread, thread)
}

// normaliseSource reads a hook's source as a harness may spell it: a payload
// saying "Compact" or " clear" still gets the resume, not the one line.
func normaliseSource(source string) string {
	return strings.ToLower(strings.TrimSpace(source))
}
