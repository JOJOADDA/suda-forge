package lifecycle

import (
	"context"
	"fmt"
	"time"

	"suda-forge/domain/project"
	"suda-forge/internal/postgres"
	"suda-forge/internal/runtime"
)

type Service struct {
	Projects postgres.Projects
	Runtime  runtime.Provider
	Now      func() time.Time
}

func (s Service) Create(ctx context.Context, name string) (project.Project, error) {
	now := s.Now()
	p, err := project.New(name, now)
	if err != nil {
		return project.Project{}, err
	}
	if err := s.Projects.Create(ctx, p); err != nil {
		return project.Project{}, err
	}
	if err := p.Transition(project.StatusPreparing, s.Now()); err != nil {
		return project.Project{}, err
	}
	if err := s.Projects.Update(ctx, p); err != nil {
		return project.Project{}, err
	}
	rt, err := s.Runtime.Create(ctx, runtime.Spec{Name: "suda-" + p.Slug, CPU: 1, MemoryBytes: 1073741824, DiskBytes: 10737418240, NetworkMode: "controlled"})
	if err != nil {
		_ = p.Transition(project.StatusFailed, s.Now())
		_ = s.Projects.Update(ctx, p)
		return project.Project{}, fmt.Errorf("create project runtime: %w", err)
	}
	p.RuntimeID = rt.ID
	if err := p.Transition(project.StatusReady, s.Now()); err != nil {
		return project.Project{}, err
	}
	if err := s.Projects.Update(ctx, p); err != nil {
		return project.Project{}, err
	}
	return p, nil
}

func (s Service) ChangeStatus(ctx context.Context, id project.ID, next project.Status) (project.Project, error) {
	p, err := s.Projects.Get(ctx, id)
	if err != nil {
		return project.Project{}, err
	}
	if p.RuntimeID == "" {
		return project.Project{}, fmt.Errorf("project %s has no runtime", id)
	}
	switch next {
	case project.StatusRunning:
		if err := s.Runtime.Start(ctx, p.RuntimeID); err != nil {
			return project.Project{}, err
		}
	case project.StatusStopped:
		if err := s.Runtime.Stop(ctx, p.RuntimeID); err != nil {
			return project.Project{}, err
		}
	default:
		return project.Project{}, fmt.Errorf("unsupported requested transition: %s", next)
	}
	if err := p.Transition(next, s.Now()); err != nil {
		return project.Project{}, err
	}
	if err := s.Projects.Update(ctx, p); err != nil {
		return project.Project{}, err
	}
	return p, nil
}
