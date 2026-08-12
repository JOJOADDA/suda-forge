package productexperience

import (
	"context"
	"suda-forge/internal/events"
	"sync"
	"time"
)

type ActivityStore interface {
	SaveActivity(context.Context, Activity) error
}
type ActivityLog struct {
	mu    sync.RWMutex
	items map[string][]Activity
	now   func() time.Time
	Store ActivityStore
}

func NewActivityLog(now func() time.Time) *ActivityLog {
	if now == nil {
		now = time.Now
	}
	return &ActivityLog{items: map[string][]Activity{}, now: now}
}
func (l *ActivityLog) Append(a Activity) {
	if a.ProjectID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if a.ID == "" {
		a.ID = "activity_" + a.ProjectID + "_" + l.now().UTC().Format("20060102150405.000000000")
	}
	if a.Timestamp.IsZero() {
		a.Timestamp = l.now().UTC()
	}
	l.items[a.ProjectID] = append(l.items[a.ProjectID], a)
	if l.Store != nil {
		_ = l.Store.SaveActivity(context.Background(), a)
	}
}
func (l *ActivityLog) AppendEvent(e events.Event) {
	state := Completed
	if len(e.Type) > 0 {
		switch {
		case containsActivity(e.Type, "started"), containsActivity(e.Type, "running"):
			state = Executing
		case containsActivity(e.Type, "failed"), containsActivity(e.Type, "error"):
			state = Failed
		case containsActivity(e.Type, "blocked"):
			state = Blocked
		case containsActivity(e.Type, "waiting"), containsActivity(e.Type, "approval"):
			state = Waiting
		}
	}
	l.Append(Activity{ProjectID: e.ProjectID, Type: e.Type, State: state, Message: e.Type, Data: map[string]any{"runtime_id": e.RuntimeID, "data": e.Data}})
}
func (l *ActivityLog) List(projectID string) []Activity {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := append([]Activity(nil), l.items[projectID]...)
	return out
}
func containsActivity(v, s string) bool {
	for i := 0; i+len(s) <= len(v); i++ {
		if v[i:i+len(s)] == s {
			return true
		}
	}
	return false
}
