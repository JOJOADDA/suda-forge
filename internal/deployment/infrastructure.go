package deployment

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"suda-forge/internal/runtime"
)

var ErrPortConflict = errors.New("external port is already reserved")
var ErrInvalidRouteTarget = errors.New("route target is not a trusted project service")

type RuntimeServiceDiscovery struct{ Runtime runtime.Provider }

func (d RuntimeServiceDiscovery) Discover(ctx context.Context, runtimeID string) ([]Service, error) {
	if d.Runtime == nil {
		return nil, errors.New("runtime provider unavailable")
	}
	result, err := d.Runtime.Exec(ctx, runtimeID, runtime.Command{Argv: []string{"sh", "-lc", "ss -ltnH"}, WorkingDir: "/workspace", TimeoutSeconds: 10})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("service discovery failed: %s", result.Stderr)
	}
	services := []Service{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		host, portText, splitErr := net.SplitHostPort(fields[3])
		if splitErr != nil {
			continue
		}
		port, _ := strconv.Atoi(portText)
		services = append(services, Service{RuntimeID: runtimeID, Protocol: "tcp", Host: host, Port: port, Status: ServiceRunning, Environment: Development, UpdatedAt: time.Now().UTC()})
	}
	return services, nil
}

type MemoryPortRegistry struct {
	mu         sync.Mutex
	bindings   map[ID]PortBinding
	byExternal map[string]ID
}

func NewMemoryPortRegistry() *MemoryPortRegistry {
	return &MemoryPortRegistry{bindings: map[ID]PortBinding{}, byExternal: map[string]ID{}}
}
func (r *MemoryPortRegistry) Reserve(_ context.Context, binding PortBinding) (PortBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := fmt.Sprintf("%s/%d", binding.Protocol, binding.ExternalPort)
	if owner, ok := r.byExternal[key]; ok && owner != binding.ID {
		return PortBinding{}, ErrPortConflict
	}
	if binding.ID == "" {
		binding.ID = ID(fmt.Sprintf("port_%s_%d", binding.ProjectID, binding.ExternalPort))
	}
	binding.Status = "RESERVED"
	binding.UpdatedAt = time.Now().UTC()
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = binding.UpdatedAt
	}
	r.bindings[binding.ID] = binding
	r.byExternal[key] = binding.ID
	return binding, nil
}
func (r *MemoryPortRegistry) Release(_ context.Context, id ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	binding, ok := r.bindings[id]
	if !ok {
		return errors.New("port binding not found")
	}
	delete(r.bindings, id)
	delete(r.byExternal, fmt.Sprintf("%s/%d", binding.Protocol, binding.ExternalPort))
	return nil
}
func (r *MemoryPortRegistry) List(_ context.Context, projectID string) ([]PortBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []PortBinding{}
	for _, binding := range r.bindings {
		if binding.ProjectID == projectID {
			out = append(out, binding)
		}
	}
	return out, nil
}

type TrustedRouteValidator struct {
	Services func(context.Context, string) ([]Service, error)
}

func (v TrustedRouteValidator) Validate(ctx context.Context, preview Preview) error {
	if v.Services == nil {
		return ErrInvalidRouteTarget
	}
	services, err := v.Services(ctx, preview.ProjectID)
	if err != nil {
		return err
	}
	for _, service := range services {
		if service.ID == preview.ServiceID && service.ProjectID == preview.ProjectID && service.Status == ServiceRunning {
			return nil
		}
	}
	return ErrInvalidRouteTarget
}
func ValidateHostname(hostname string) error {
	if hostname == "" || len(hostname) > 253 {
		return errors.New("invalid hostname")
	}
	if strings.Contains(hostname, "/") || strings.Contains(hostname, "\\") || strings.Contains(hostname, "..") || strings.EqualFold(hostname, "localhost") {
		return errors.New("unsafe hostname")
	}
	if net.ParseIP(hostname) != nil {
		return errors.New("IP hostnames are not allowed")
	}
	return nil
}
func ValidateTargetURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidRouteTarget
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "169.254.") || strings.HasSuffix(host, ".internal") {
		return ErrInvalidRouteTarget
	}
	return nil
}
