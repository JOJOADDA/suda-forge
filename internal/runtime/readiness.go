package runtime

import (
	"os"
	"os/exec"
	"strings"
)

type Readiness struct {
	Application string `json:"application"`
	Runtime     string `json:"runtime"`
	Reason      string `json:"reason,omitempty"`
}

func HostReadiness() Readiness {
	result := Readiness{Application: "READY", Runtime: "READY"}
	if data, err := exec.Command("systemd-detect-virt").Output(); err == nil {
		virtualization := strings.TrimSpace(string(data))
		if virtualization == "docker" || virtualization == "podman" || virtualization == "lxc" || virtualization == "container" {
			result.Runtime = "BLOCKED"
			result.Reason = "runtime host is a nested or restricted container: " + virtualization
		}
	}
	if _, err := os.Stat("/etc/lxc/lxc-usernet"); err != nil {
		result.Runtime = "BLOCKED"
		if result.Reason == "" {
			result.Reason = "LXC user network policy is unavailable"
		}
	}
	return result
}
