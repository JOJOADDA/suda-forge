package productexperience

import (
	"context"
	"suda-forge/internal/projectcomputer"
	"testing"
)

type fakeComputerReader struct {
	computer projectcomputer.ProjectComputer
}

func (f fakeComputerReader) Get(context.Context, projectcomputer.ID) (projectcomputer.ProjectComputer, error) {
	return f.computer, nil
}
func TestVisualQABlocksUnavailableBrowser(t *testing.T) {
	r := VisualQABoundary{Computers: fakeComputerReader{computer: projectcomputer.ProjectComputer{ID: "pc1", Capabilities: []projectcomputer.CapabilityCheck{{Capability: projectcomputer.Browser, Status: projectcomputer.Blocked, Evidence: "LXC runtime unavailable"}}}}}
	out, err := r.Run(context.Background(), VisualQARequest{ProjectID: "p1", ComputerID: "pc1", Viewports: []Viewport{{Name: "mobile", Width: 390, Height: 844}}})
	if err != nil || out.Status != VisualQABlocked {
		t.Fatalf("expected blocked visual QA, got %#v %v", out, err)
	}
}
