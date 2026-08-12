package aifabric

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaRuntimeDiscoveryHealthGenerateAndStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "qwen:7b"}}})
		case "/api/version":
			json.NewEncoder(w).Encode(map[string]string{"version": "0.1"})
		case "/api/generate":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("missing content type")
			}
			json.NewEncoder(w).Encode(map[string]any{"response": "hello", "done": true, "prompt_eval_count": 2, "eval_count": 3})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	runtime := NewOllamaRuntime(RuntimeSpec{ID: "ollama-test", Endpoint: srv.URL, Kind: "ollama", Local: true})
	models, err := runtime.Discover(context.Background())
	if err != nil || len(models) != 1 || models[0].ID != "qwen:7b" {
		t.Fatalf("models=%+v err=%v", models, err)
	}
	health, err := runtime.Health(context.Background())
	if err != nil || health.Status != RuntimeOnline {
		t.Fatalf("health=%+v err=%v", health, err)
	}
	response, err := runtime.Generate(context.Background(), InferenceRequest{RequestID: "req", RuntimeID: "ollama-test", ModelID: "qwen:7b", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil || response.Content != "hello" || response.Usage.TotalTokens != 5 {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}
func TestResourceValidationAndRegistry(t *testing.T) {
	resources := HostResources{MemoryAvailable: 100, GPUMemoryAvailable: 200, GPUCount: 1}
	if err := ResourceSatisfies(resources, ResourceRequirement{MemoryBytes: 101}); err == nil {
		t.Fatal("expected memory rejection")
	}
	registry := NewRuntimeRegistry()
	runtime := NewOllamaRuntime(RuntimeSpec{ID: "r", Endpoint: "http://127.0.0.1:1"})
	if err := registry.RegisterRuntime(runtime); err != nil {
		t.Fatal(err)
	}
	registry.RegisterModel(ModelDescriptor{ID: "m", RuntimeID: "r", Status: ModelReady})
	if _, ok := registry.Model("m"); !ok {
		t.Fatal("model not registered")
	}
}
