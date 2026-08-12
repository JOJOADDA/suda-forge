package environment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"suda-forge/internal/runtime"
)

type RuntimeExecutor interface {
	Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error)
}

func Verify(ctx context.Context, exec RuntimeExecutor, runtimeID string, manifest Manifest, now time.Time) VerificationResult {
	result := VerificationResult{Status: VerificationReady, RuntimeID: runtimeID, Checks: []VerificationCheck{}, Fingerprint: BuildFingerprint(manifest, now)}
	if exec == nil || runtimeID == "" {
		result.Status = VerificationBlocked
		result.Checks = append(result.Checks, VerificationCheck{Name: "runtime", Required: true, Status: "BLOCKED_BY_ENVIRONMENT", Reason: "runtime executor unavailable"})
		return result
	}
	check := func(name string, required bool, cmd runtime.Command) {
		out, err := exec.Exec(ctx, runtimeID, cmd)
		status := "PASSED"
		reason := strings.TrimSpace(out.Stdout)
		if err != nil || out.ExitCode != 0 {
			status = "FAILED"
			if err != nil {
				reason = err.Error()
			} else {
				reason = out.Stderr
			}
			if required {
				result.Status = VerificationFailed
			}
		}
		result.Checks = append(result.Checks, VerificationCheck{Name: name, Required: required, Status: status, Evidence: reason})
	}
	check("runtime", true, runtime.Command{Argv: []string{"sh", "-lc", "printf runtime-ready"}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	check("filesystem", true, runtime.Command{Argv: []string{"sh", "-lc", "test -d /workspace"}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	check("git", true, runtime.Command{Argv: []string{"sh", "-lc", "git --version"}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	for _, lang := range manifest.Languages {
		check("language:"+lang.Name, lang.Required, runtime.Command{Argv: []string{"sh", "-lc", fmt.Sprintf("command -v %s", safeName(lang.Name))}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	}
	for _, tool := range manifest.PackageManagers {
		check("package-manager:"+tool.Name, tool.Required, runtime.Command{Argv: []string{"sh", "-lc", fmt.Sprintf("command -v %s", safeName(tool.Name))}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	}
	for _, agent := range manifest.AgentCLIs {
		check("agent:"+agent.AgentID, agent.Required, runtime.Command{Argv: []string{"sh", "-lc", fmt.Sprintf("command -v %s", agentBinary(agent.AgentID))}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	}
	for _, browser := range manifest.Browsers {
		check("browser:"+browser.Name, browser.Required, runtime.Command{Argv: []string{"sh", "-lc", "command -v chromium || command -v chromium-browser"}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	}
	if result.Status == VerificationReady {
		for _, c := range result.Checks {
			if c.Status == "FAILED" && !c.Required {
				result.Status = VerificationDegraded
			}
		}
	}
	return result
}
func safeName(v string) string {
	switch v {
	case "node", "go", "python", "python3", "pnpm", "npm", "yarn", "git":
		return v
	default:
		return "sh"
	}
}
func agentBinary(v string) string {
	switch v {
	case "claude_code":
		return "claude"
	default:
		return v
	}
}
