package project

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ID string

type Status string

const (
	StatusCreating   Status = "CREATING"
	StatusPreparing  Status = "PREPARING"
	StatusReady      Status = "READY"
	StatusRunning    Status = "RUNNING"
	StatusStopping   Status = "STOPPING"
	StatusStopped    Status = "STOPPED"
	StatusRecovering Status = "RECOVERING"
	StatusFailed     Status = "FAILED"
	StatusDestroyed  Status = "DESTROYED"
)

type Project struct {
	ID        ID        `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    Status    `json:"status"`
	RuntimeID string    `json:"runtime_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var ErrInvalidTransition = errors.New("invalid project status transition")

func New(name string, now time.Time) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, errors.New("project name is required")
	}
	slug := slugify(name)
	return Project{
		ID:        ID(fmt.Sprintf("proj_%d", now.UnixNano())),
		Name:      name,
		Slug:      slug,
		Status:    StatusCreating,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *Project) Transition(next Status, now time.Time) error {
	if !allowed(p.Status, next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, p.Status, next)
	}
	p.Status = next
	p.UpdatedAt = now
	return nil
}

func allowed(from, to Status) bool {
	switch from {
	case StatusCreating:
		return to == StatusPreparing || to == StatusFailed
	case StatusPreparing:
		return to == StatusReady || to == StatusFailed
	case StatusReady:
		return to == StatusRunning || to == StatusStopped || to == StatusFailed
	case StatusRunning:
		return to == StatusStopping || to == StatusRecovering || to == StatusFailed
	case StatusStopping:
		return to == StatusStopped || to == StatusFailed
	case StatusStopped:
		return to == StatusRunning || to == StatusRecovering || to == StatusDestroyed
	case StatusRecovering:
		return to == StatusReady || to == StatusRunning || to == StatusFailed
	case StatusFailed:
		return to == StatusRecovering || to == StatusDestroyed
	default:
		return false
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
