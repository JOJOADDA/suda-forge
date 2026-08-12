package agent

import (
	"encoding/json"
	"strings"
	"time"
)

func Normalize(sessionID SessionID, provider string, rawLine string, now time.Time) Event {
	line := strings.TrimSpace(rawLine)
	event := Event{SessionID: sessionID, Type: EventMessage, Timestamp: now.UTC(), Normalized: map[string]any{"text": line}, Raw: map[string]any{"provider": provider, "line": line}}
	var object map[string]any
	if json.Unmarshal([]byte(line), &object) == nil {
		event.Raw["payload"] = object
		if typ, ok := object["type"].(string); ok {
			switch typ {
			case "thinking":
				event.Type = EventThinking
			case "tool.started":
				event.Type = EventToolStarted
			case "tool.output":
				event.Type = EventToolOutput
			case "error":
				event.Type = EventError
			case "permission.required":
				event.Type = EventPermissionRequired
				event.RequiresApproval = event.Type == EventPermissionRequired
			}
		}
		if text, ok := object["text"].(string); ok {
			event.Normalized["text"] = text
		}
	}
	return event
}
