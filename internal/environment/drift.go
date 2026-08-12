package environment

import "strings"

type DriftStatus string

const (
	EnvironmentUnchanged DriftStatus = "UNCHANGED"
	EnvironmentDrift     DriftStatus = "ENVIRONMENT_DRIFT"
	EnvironmentUnknown   DriftStatus = "UNKNOWN"
)

type DriftResult struct {
	Status   DriftStatus `json:"status"`
	Reasons  []string    `json:"reasons"`
	Expected Fingerprint `json:"expected"`
	Actual   Fingerprint `json:"actual"`
}

func Compare(expected, actual Fingerprint) DriftResult {
	out := DriftResult{Status: EnvironmentUnchanged, Expected: expected, Actual: actual, Reasons: []string{}}
	if expected.OS != actual.OS {
		out.Reasons = append(out.Reasons, "operating system changed")
	}
	if expected.Image != actual.Image {
		out.Reasons = append(out.Reasons, "base image changed")
	}
	if expected.Browser != actual.Browser {
		out.Reasons = append(out.Reasons, "browser version changed")
	}
	for name, want := range expected.Languages {
		if got := actual.Languages[name]; got != want {
			out.Reasons = append(out.Reasons, "language "+name+" expected "+want+" got "+got)
		}
	}
	for name, want := range expected.Tools {
		if got := actual.Tools[name]; got != want {
			out.Reasons = append(out.Reasons, "tool "+name+" expected "+want+" got "+got)
		}
	}
	for name, want := range expected.Agents {
		if got := actual.Agents[name]; got != want {
			out.Reasons = append(out.Reasons, "agent "+name+" expected "+want+" got "+got)
		}
	}
	if len(out.Reasons) > 0 {
		out.Status = EnvironmentDrift
	}
	if expected.Value == "" || actual.Value == "" {
		out.Status = EnvironmentUnknown
	}
	return out
}
func IsVersionCompatible(expected, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	return expected == "" || expected == "latest-compatible" || expected == actual || strings.HasSuffix(expected, ".x") && strings.HasPrefix(actual, strings.TrimSuffix(expected, ".x"))
}
