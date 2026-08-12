package aifabric

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func DiscoverHostResources(ctx context.Context) (HostResources, error) {
	out := HostResources{CPUCores: runtime.NumCPU(), DetectedAt: time.Now().UTC()}
	if file, err := os.Open("/proc/meminfo"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) < 2 {
				continue
			}
			value, _ := strconv.ParseUint(parts[1], 10, 64)
			switch parts[0] {
			case "MemTotal:":
				out.MemoryTotal = value * 1024
			case "MemAvailable:":
				out.MemoryAvailable = value * 1024
			}
		}
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		out.DiskTotal = stat.Blocks * uint64(stat.Bsize)
		out.DiskAvailable = stat.Bavail * uint64(stat.Bsize)
	}
	if err := discoverNVIDIA(ctx, &out); err != nil && !isCommandMissing(err) {
		return out, err
	}
	return out, nil
}
func discoverNVIDIA(ctx context.Context, out *HostResources) error {
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=index,name,memory.total,memory.free,driver_version,compute_cap", "--format=csv,noheader,nounits")
	raw, err := cmd.Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		total, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
		free, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)
		out.GPUs = append(out.GPUs, GPUResource{ID: strings.TrimSpace(parts[0]), Vendor: "NVIDIA", Model: strings.TrimSpace(parts[1]), MemoryTotal: total * 1024 * 1024, MemoryAvailable: free * 1024 * 1024, Driver: strings.TrimSpace(parts[4]), ComputeCapability: strings.TrimSpace(parts[5]), Runtime: "CUDA"})
		out.GPUCount++
		out.GPUMemoryTotal += total * 1024 * 1024
		out.GPUMemoryAvailable += free * 1024 * 1024
	}
	return nil
}
func isCommandMissing(err error) bool {
	return strings.Contains(err.Error(), "executable file not found")
}
func ResourceSatisfies(resources HostResources, requirement ResourceRequirement) error {
	if requirement.MemoryBytes > resources.MemoryAvailable {
		return fmt.Errorf("insufficient memory")
	}
	if requirement.VRAMBytes > resources.GPUMemoryAvailable {
		return fmt.Errorf("insufficient GPU memory")
	}
	if requirement.GPURequired && resources.GPUCount == 0 {
		return fmt.Errorf("GPU is required but unavailable")
	}
	return nil
}
