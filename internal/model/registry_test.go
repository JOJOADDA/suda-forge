package model

import (
	"testing"

	"suda-forge/internal/agent"
)

func TestRegistrySeparatesProviderAndModel(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterModel(agent.Model{ID: "m1", ProviderID: "p1", ModelID: "model"}); err == nil {
		t.Fatal("model without provider must be rejected")
	}
	if err := r.RegisterProvider(agent.Provider{ID: "p1", Name: "Provider", Type: "custom"}); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterModel(agent.Model{ID: "m1", ProviderID: "p1", ModelID: "model", DisplayName: "Model", Remote: true}); err != nil {
		t.Fatal(err)
	}
	if got, ok := r.Model("m1"); !ok || got.ProviderID != "p1" {
		t.Fatalf("model lookup = %+v, %v", got, ok)
	}
}
