package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"suda-forge/internal/runtime"
)

type CaddyProxy struct {
	AdminURL string
	Client   *http.Client
}

func (p CaddyProxy) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}
func (p CaddyProxy) CreateRoute(ctx context.Context, preview Preview) error {
	return p.put(ctx, "/config/apps/http/servers/suda/routes/"+string(preview.ID), map[string]any{"match": []any{map[string]any{"host": []string{preview.Hostname}}}, "handle": []any{map[string]any{"handler": "reverse_proxy", "upstreams": []any{map[string]any{"dial": string(preview.ServiceID)}}}}})
}
func (p CaddyProxy) UpdateRoute(ctx context.Context, preview Preview) error {
	return p.CreateRoute(ctx, preview)
}
func (p CaddyProxy) DeleteRoute(ctx context.Context, id ID) error {
	return p.request(ctx, http.MethodDelete, "/config/apps/http/servers/suda/routes/"+string(id), nil, nil)
}
func (p CaddyProxy) URL(preview Preview) string { return "https://" + preview.Hostname }
func (p CaddyProxy) put(ctx context.Context, path string, body any) error {
	return p.request(ctx, http.MethodPut, path, body, nil)
}
func (p CaddyProxy) request(ctx context.Context, method, path string, body, out any) error {
	if p.AdminURL == "" {
		return errors.New("caddy admin endpoint is not configured")
	}
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(p.AdminURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("caddy returned status %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

type CaddyCertificate struct{ Proxy CaddyProxy }

func (c CaddyCertificate) Issue(ctx context.Context, domain Domain) (Certificate, error) {
	if err := ValidateHostname(domain.Hostname); err != nil {
		return Certificate{}, err
	}
	return Certificate{ID: ID("cert_" + string(domain.ID)), DomainID: domain.ID, Status: "REQUESTED", Issuer: "Caddy/Let's Encrypt", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
}
func (c CaddyCertificate) Renew(ctx context.Context, certificate Certificate) (Certificate, error) {
	certificate.Status = "RENEWAL_REQUESTED"
	certificate.UpdatedAt = time.Now().UTC()
	return certificate, nil
}
func (c CaddyCertificate) Status(ctx context.Context, certificate Certificate) (Certificate, error) {
	return certificate, nil
}

type RuntimeHealthChecker struct{ Runtime runtime.Provider }

func (h RuntimeHealthChecker) Check(ctx context.Context, check HealthCheck) (HealthCheck, error) {
	if h.Runtime == nil {
		return check, errors.New("runtime provider unavailable")
	}
	if check.Timeout <= 0 {
		check.Timeout = 5 * time.Second
	}
	if check.Type == HealthHTTP || check.Type == HealthTCP {
		result, err := h.Runtime.Exec(ctx, check.RuntimeID, runtime.Command{Argv: []string{"sh", "-lc", check.Target}, WorkingDir: "/workspace", TimeoutSeconds: int(check.Timeout.Seconds())})
		if err != nil {
			return check, err
		}
		if result.ExitCode != 0 {
			check.Status = "FAILED"
			check.LastError = result.Stderr
			return check, errors.New(result.Stderr)
		}
		check.Status = "PASSED"
		check.CheckedAt = time.Now().UTC()
		return check, nil
	}
	return check, errors.New("health check type requires a runtime-backed implementation")
}

type LocalStorage struct{ Root string }

func (s LocalStorage) CreateVolume(ctx context.Context, projectID, name string) (string, error) {
	path, err := safeChild(s.Root, projectID, name)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(path, 0700); err != nil {
		return "", err
	}
	return path, nil
}
func (s LocalStorage) Snapshot(ctx context.Context, source, reference string) (Snapshot, error) {
	if source == "" || reference == "" {
		return Snapshot{}, errors.New("source and reference are required")
	}
	target, err := safeChild(s.Root, "snapshots", reference)
	if err != nil {
		return Snapshot{}, err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return Snapshot{}, err
	}
	if err = os.Rename(source, target); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{ID: ID("snapshot_" + reference), Kind: "local", Reference: reference, Source: source, Status: "CREATED", CreatedAt: time.Now().UTC()}, nil
}
func (s LocalStorage) Restore(ctx context.Context, snapshot Snapshot) error {
	if snapshot.Reference == "" || snapshot.Source == "" {
		return errors.New("invalid snapshot")
	}
	return nil
}
func safeChild(root string, parts ...string) (string, error) {
	if root == "" {
		return "", errors.New("storage root is empty")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "/\\") {
			return "", errors.New("unsafe storage path")
		}
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(append([]string{base}, parts...)...))
	if err != nil {
		return "", err
	}
	if path != base && !strings.HasPrefix(path, base+string(os.PathSeparator)) {
		return "", errors.New("storage path escapes root")
	}
	return path, nil
}
