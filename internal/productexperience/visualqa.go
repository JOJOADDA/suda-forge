package productexperience

import (
	"context"
	"errors"
	"suda-forge/internal/projectcomputer"
)

type VisualQAStatus string

const (
	VisualQAPassed      VisualQAStatus = "PASSED"
	VisualQAFailed      VisualQAStatus = "FAILED"
	VisualQABlocked     VisualQAStatus = "BLOCKED_BY_ENVIRONMENT"
	VisualQAUnsupported VisualQAStatus = "UNSUPPORTED"
)

type VisualQARequest struct {
	ProjectID  string             `json:"project_id"`
	ComputerID projectcomputer.ID `json:"computer_id"`
	Viewports  []Viewport         `json:"viewports"`
}
type Viewport struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
type VisualQAResult struct {
	Status    VisualQAStatus    `json:"status"`
	Reason    string            `json:"reason"`
	Viewports []Viewport        `json:"viewports"`
	Findings  []string          `json:"findings"`
	Evidence  map[string]string `json:"evidence"`
}
type CapabilityReader interface {
	Get(context.Context, projectcomputer.ID) (projectcomputer.ProjectComputer, error)
}
type VisualQABoundary struct{ Computers CapabilityReader }

func (v VisualQABoundary) Run(ctx context.Context, in VisualQARequest) (VisualQAResult, error) {
	if v.Computers == nil {
		return VisualQAResult{}, errors.New("project computer capability reader unavailable")
	}
	c, err := v.Computers.Get(ctx, in.ComputerID)
	if err != nil {
		return VisualQAResult{}, err
	}
	for _, cap := range c.Capabilities {
		if cap.Capability == projectcomputer.Browser {
			if cap.Status == projectcomputer.Blocked {
				return VisualQAResult{Status: VisualQABlocked, Reason: cap.Evidence, Viewports: in.Viewports, Findings: []string{}, Evidence: map[string]string{}}, nil
			}
			if cap.Status != projectcomputer.Supported {
				return VisualQAResult{Status: VisualQAUnsupported, Reason: "browser capability is not supported", Viewports: in.Viewports, Findings: []string{}, Evidence: map[string]string{}}, nil
			}
		}
	}
	return VisualQAResult{Status: VisualQAUnsupported, Reason: "browser capability was not verified", Viewports: in.Viewports, Findings: []string{}, Evidence: map[string]string{}}, nil
}
