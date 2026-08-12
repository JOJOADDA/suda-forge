package aifabric

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPRuntime struct {
	spec   RuntimeSpec
	client *http.Client
	caps   RuntimeCapabilities
	kind   string
}

func NewOllamaRuntime(spec RuntimeSpec) *HTTPRuntime {
	if spec.Kind == "" {
		spec.Kind = "ollama"
	}
	return &HTTPRuntime{spec: spec, client: &http.Client{Timeout: 5 * time.Minute}, kind: "ollama", caps: RuntimeCapabilities{Generate: true, Stream: true, Embeddings: true, Install: true, Load: true, Unload: true, ModelDiscovery: true, ToolCalling: true}}
}
func NewOpenAICompatibleRuntime(spec RuntimeSpec) *HTTPRuntime {
	if spec.Kind == "" {
		spec.Kind = "openai-compatible"
	}
	return &HTTPRuntime{spec: spec, client: &http.Client{Timeout: 5 * time.Minute}, kind: "openai-compatible", caps: RuntimeCapabilities{Generate: true, Stream: true, Embeddings: true, ModelDiscovery: true, OpenAICompatible: true, StructuredOutput: true, ToolCalling: true}}
}
func NewVLLMRuntime(spec RuntimeSpec) *HTTPRuntime {
	if spec.Kind == "" {
		spec.Kind = "vllm"
	}
	return NewOpenAICompatibleRuntime(spec)
}
func NewLlamaCPPRuntime(spec RuntimeSpec) *HTTPRuntime {
	if spec.Kind == "" {
		spec.Kind = "llama.cpp"
	}
	return NewOpenAICompatibleRuntime(spec)
}
func (r *HTTPRuntime) Spec() RuntimeSpec                 { return r.spec }
func (r *HTTPRuntime) Capabilities() RuntimeCapabilities { return r.caps }
func (r *HTTPRuntime) Discover(ctx context.Context) ([]ModelDescriptor, error) {
	return r.ListModels(ctx)
}
func (r *HTTPRuntime) ListModels(ctx context.Context) ([]ModelDescriptor, error) {
	var raw struct {
		Models []struct {
			Name    string `json:"name"`
			Model   string `json:"model"`
			ID      string `json:"id"`
			Context int    `json:"context_length"`
			OwnedBy string `json:"owned_by"`
		} `json:"models"`
	}
	var err error
	if r.kind == "ollama" {
		err = r.get(ctx, "/api/tags", &raw)
	} else {
		err = r.get(ctx, "/v1/models", &raw)
	}
	if err != nil {
		return nil, err
	}
	out := []ModelDescriptor{}
	for _, m := range raw.Models {
		id := m.ID
		if id == "" {
			id = m.Name
		}
		if id == "" {
			id = m.Model
		}
		out = append(out, ModelDescriptor{ID: ModelID(id), ProviderID: string(r.spec.ID), RuntimeID: r.spec.ID, DisplayName: id, ContextWindow: m.Context, Capabilities: map[Capability]bool{CapabilityLocal: r.spec.Local, CapabilityGeneral: true, CapabilityCoding: true, CapabilityToolUse: r.caps.ToolCalling}, Local: r.spec.Local, Remote: !r.spec.Local, Availability: "AVAILABLE", Status: ModelAvailable})
	}
	return out, nil
}
func (r *HTTPRuntime) Health(ctx context.Context) (RuntimeHealth, error) {
	started := time.Now()
	var data map[string]any
	path := "/api/version"
	if r.kind != "ollama" {
		path = "/v1/models"
	}
	if err := r.get(ctx, path, &data); err != nil {
		return RuntimeHealth{RuntimeID: r.spec.ID, Status: RuntimeOffline, Endpoint: r.spec.Endpoint, Latency: time.Since(started), LastChecked: time.Now().UTC(), Error: err.Error()}, err
	}
	models, _ := r.ListModels(ctx)
	ids := make([]string, 0, len(models))
	for _, m := range models {
		ids = append(ids, string(m.ID))
	}
	return RuntimeHealth{RuntimeID: r.spec.ID, Status: RuntimeOnline, Endpoint: r.spec.Endpoint, Latency: time.Since(started), AvailableModels: ids, LastChecked: time.Now().UTC()}, nil
}
func (r *HTTPRuntime) Start(context.Context) error {
	return errors.New("runtime process lifecycle is managed externally")
}
func (r *HTTPRuntime) Stop(context.Context) error {
	return errors.New("runtime process lifecycle is managed externally")
}
func (r *HTTPRuntime) Restart(context.Context) error {
	return errors.New("runtime process lifecycle is managed externally")
}
func (r *HTTPRuntime) Install(ctx context.Context, req ModelInstallRequest) (ModelDescriptor, error) {
	if r.kind != "ollama" {
		return ModelDescriptor{}, errors.New("model installation unsupported by runtime")
	}
	body := map[string]any{"name": req.Source}
	var out map[string]any
	if err := r.post(ctx, "/api/pull", body, &out); err != nil {
		return ModelDescriptor{}, err
	}
	models, err := r.ListModels(ctx)
	if err != nil {
		return ModelDescriptor{}, err
	}
	for _, m := range models {
		if string(m.ID) == req.Source || string(m.ID) == string(req.ModelID) {
			m.Status = ModelInstalled
			return m, nil
		}
	}
	return ModelDescriptor{ID: req.ModelID, RuntimeID: r.spec.ID, ProviderID: string(r.spec.ID), Status: ModelInstalled, Local: r.spec.Local}, nil
}
func (r *HTTPRuntime) Remove(context.Context, ModelID) error {
	return errors.New("model removal is not exposed by this runtime adapter")
}
func (r *HTTPRuntime) LoadModel(ctx context.Context, req ModelLoadRequest) error {
	if r.kind == "ollama" {
		var out map[string]any
		return r.post(ctx, "/api/show", map[string]any{"name": req.ModelID}, &out)
	}
	return nil
}
func (r *HTTPRuntime) UnloadModel(context.Context, ModelLoadRequest) error { return nil }
func (r *HTTPRuntime) Generate(ctx context.Context, req InferenceRequest) (InferenceResponse, error) {
	started := time.Now()
	if r.kind == "ollama" {
		var out struct {
			Response        string `json:"response"`
			Done            bool   `json:"done"`
			PromptEvalCount int    `json:"prompt_eval_count"`
			EvalCount       int    `json:"eval_count"`
			Model           string `json:"model"`
		}
		body := map[string]any{"model": req.ModelID, "prompt": lastContent(req.Messages), "stream": false}
		if req.SystemPrompt != "" {
			body["system"] = req.SystemPrompt
		}
		if err := r.post(ctx, "/api/generate", body, &out); err != nil {
			return InferenceResponse{}, err
		}
		return InferenceResponse{RequestID: req.RequestID, ModelID: req.ModelID, RuntimeID: req.RuntimeID, Content: out.Response, FinishReason: "stop", Latency: time.Since(started), Usage: Usage{InputTokens: out.PromptEvalCount, OutputTokens: out.EvalCount, TotalTokens: out.PromptEvalCount + out.EvalCount}}, nil
	}
	body := map[string]any{"model": req.ModelID, "messages": req.Messages, "stream": false}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
			Finish  string  `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	if err := r.post(ctx, "/v1/chat/completions", body, &out); err != nil {
		return InferenceResponse{}, err
	}
	if len(out.Choices) == 0 {
		return InferenceResponse{}, errors.New("runtime returned no choices")
	}
	return InferenceResponse{RequestID: req.RequestID, ModelID: req.ModelID, RuntimeID: req.RuntimeID, Content: out.Choices[0].Message.Content, FinishReason: out.Choices[0].Finish, Usage: out.Usage, Latency: time.Since(started)}, nil
}
func (r *HTTPRuntime) Stream(ctx context.Context, req InferenceRequest) (<-chan StreamEvent, error) {
	req.Stream = true
	body := map[string]any{"model": req.ModelID, "messages": req.Messages, "stream": true}
	path := "/v1/chat/completions"
	if r.kind == "ollama" {
		body = map[string]any{"model": req.ModelID, "prompt": lastContent(req.Messages), "stream": true}
		path = "/api/generate"
	}
	raw, err := r.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	out := make(chan StreamEvent, 16)
	go func() {
		defer close(out)
		defer raw.Body.Close()
		scanner := bufio.NewScanner(raw.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			line = strings.TrimPrefix(line, "data:")
			if line == "" || line == "[DONE]" {
				continue
			}
			var value map[string]any
			if json.Unmarshal([]byte(line), &value) != nil {
				continue
			}
			delta := extractDelta(value, r.kind)
			out <- StreamEvent{RequestID: req.RequestID, Type: "token", Delta: delta}
		}
		out <- StreamEvent{RequestID: req.RequestID, Type: "completion", Done: true}
	}()
	return out, nil
}
func (r *HTTPRuntime) Embeddings(ctx context.Context, req InferenceRequest) ([][]float32, error) {
	if r.kind == "ollama" {
		var out struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := r.post(ctx, "/api/embeddings", map[string]any{"model": req.ModelID, "prompt": lastContent(req.Messages)}, &out); err != nil {
			return nil, err
		}
		return [][]float32{out.Embedding}, nil
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := r.post(ctx, "/v1/embeddings", map[string]any{"model": req.ModelID, "input": lastContent(req.Messages)}, &out); err != nil {
		return nil, err
	}
	result := make([][]float32, 0, len(out.Data))
	for _, item := range out.Data {
		result = append(result, item.Embedding)
	}
	return result, nil
}
func (r *HTTPRuntime) Resources(ctx context.Context) (HostResources, error) {
	return DiscoverHostResources(ctx)
}
func (r *HTTPRuntime) get(ctx context.Context, path string, out any) error {
	resp, err := r.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
func (r *HTTPRuntime) post(ctx context.Context, path string, body, out any) error {
	resp, err := r.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
func (r *HTTPRuntime) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if r.spec.Endpoint == "" {
		return nil, errors.New("runtime endpoint is empty")
	}
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(r.spec.Endpoint, "/")+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("runtime %s returned %d: %s", r.spec.ID, resp.StatusCode, string(raw))
	}
	return resp, nil
}
func lastContent(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Content != "" {
			return messages[i].Content
		}
	}
	return ""
}
func extractDelta(value map[string]any, kind string) string {
	if kind == "ollama" {
		if v, ok := value["response"].(string); ok {
			return v
		}
	}
	choices, _ := value["choices"].([]any)
	if len(choices) > 0 {
		choice, _ := choices[0].(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		text, _ := delta["content"].(string)
		return text
	}
	return ""
}
