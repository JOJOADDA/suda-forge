package agent

import (
	"testing"
	"time"
)

func TestNormalizePreservesRawAndMapsTypes(t *testing.T) {
	message := Normalize("s1", "codex", `{"type":"permission.required","text":"git push"}`, time.Now())
	if message.Type != EventPermissionRequired || !message.RequiresApproval {
		t.Fatalf("event = %+v", message)
	}
	if message.Raw["provider"] != "codex" {
		t.Fatalf("raw = %+v", message.Raw)
	}
	if message.Normalized["text"] != "git push" {
		t.Fatalf("normalized = %+v", message.Normalized)
	}
}
