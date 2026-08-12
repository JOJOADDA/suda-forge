package routing

import "context"

type ProviderAdapter interface {
	ID() string
	ListModels(context.Context) ([]ModelProfile, error)
	GetModel(context.Context, string) (ModelProfile, error)
	Health(context.Context) (ProviderHealth, error)
	Capabilities(context.Context) []Capability
}
type DiscoveryState string

const (
	Discovered        DiscoveryState = "DISCOVERED"
	Registered        DiscoveryState = "REGISTERED"
	Enabled           DiscoveryState = "ENABLED"
	DiscoveryDisabled DiscoveryState = "DISABLED"
)

type DiscoveredModel struct {
	Profile ModelProfile   `json:"profile"`
	State   DiscoveryState `json:"state"`
}
