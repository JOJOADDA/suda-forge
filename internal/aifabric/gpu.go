package aifabric

import (
	"context"
	"errors"
	"fmt"
)

type GPUAllocator interface {
	Allocate(context.Context, string, RuntimeID, ResourceRequirement) (GPUAllocation, error)
	Release(context.Context, string) error
}
type ContractGPUAllocator struct {
	Discover func(context.Context) (HostResources, error)
}

func (a ContractGPUAllocator) Allocate(ctx context.Context, projectID string, runtimeID RuntimeID, requirement ResourceRequirement) (GPUAllocation, error) {
	if a.Discover == nil {
		return GPUAllocation{}, errors.New("GPU allocation capability is unavailable")
	}
	resources, err := a.Discover(ctx)
	if err != nil {
		return GPUAllocation{}, err
	}
	if err = ResourceSatisfies(resources, requirement); err != nil {
		return GPUAllocation{}, err
	}
	if !requirement.GPURequired {
		return GPUAllocation{ProjectID: projectID, RuntimeID: runtimeID, Status: "NOT_REQUIRED"}, nil
	}
	ids := make([]string, 0, len(resources.GPUs))
	for _, gpu := range resources.GPUs {
		ids = append(ids, gpu.ID)
	}
	if len(ids) == 0 {
		return GPUAllocation{}, fmt.Errorf("GPU allocation blocked by environment")
	}
	return GPUAllocation{ID: projectID + "/" + string(runtimeID), ProjectID: projectID, RuntimeID: runtimeID, GPUIDs: ids, MemoryBytes: requirement.VRAMBytes, Status: "ALLOCATED_BY_POLICY"}, nil
}
func (a ContractGPUAllocator) Release(context.Context, string) error { return nil }
