package environment

import (
	"fmt"
)

type CapacityProvider interface{ Capacity() ResourceRequirement }

func PlanResources(required ResourceRequirement, available ResourceRequirement) ResourcePlan {
	p := ResourcePlan{Required: required, Available: available, Decision: Approved, Reasons: []string{}}
	if required.CPU > available.CPU || required.MemoryBytes > available.MemoryBytes || required.DiskBytes > available.DiskBytes || required.GPU && available.GPU == false || required.GPUMemoryBytes > available.GPUMemoryBytes {
		p.Decision = Rejected
		p.Reasons = append(p.Reasons, "required resources exceed available capacity")
		return p
	}
	if required.CPU == available.CPU || required.MemoryBytes > available.MemoryBytes*8/10 || required.DiskBytes > available.DiskBytes*8/10 {
		p.Decision = Degraded
		p.Reasons = append(p.Reasons, "capacity is available but operating headroom is limited")
	}
	if len(p.Reasons) == 0 {
		p.Reasons = append(p.Reasons, fmt.Sprintf("capacity satisfies %d CPU and %d bytes memory", required.CPU, required.MemoryBytes))
	}
	return p
}
