package aifabric

import "suda-forge/internal/routing"

type ProjectAISettings struct {
	ProjectID        string                `json:"project_id"`
	PreferredAgent   string                `json:"preferred_agent,omitempty"`
	PreferredModel   string                `json:"preferred_model,omitempty"`
	RoutingPolicy    routing.RoutingPolicy `json:"routing_policy"`
	PrivacyPolicy    routing.PrivacyClass  `json:"privacy_policy"`
	LocalOnly        bool                  `json:"local_only"`
	Budget           float64               `json:"budget"`
	AllowedProviders []string              `json:"allowed_providers,omitempty"`
	AllowedRuntimes  []string              `json:"allowed_runtimes,omitempty"`
	AllowedModels    []string              `json:"allowed_models,omitempty"`
}

func ApplyProjectPolicy(settings ProjectAISettings, models []routing.ModelProfile) []routing.ModelProfile {
	out := make([]routing.ModelProfile, 0, len(models))
	for _, model := range models {
		if len(settings.AllowedProviders) > 0 && !contains(settings.AllowedProviders, model.ProviderID) {
			continue
		}
		if len(settings.AllowedRuntimes) > 0 && !contains(settings.AllowedRuntimes, model.RuntimeID) {
			continue
		}
		if len(settings.AllowedModels) > 0 && !contains(settings.AllowedModels, model.ModelID) {
			continue
		}
		if settings.LocalOnly && !model.Local {
			continue
		}
		out = append(out, model)
	}
	return out
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
