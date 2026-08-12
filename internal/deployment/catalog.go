package deployment

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Catalog struct {
	mu           sync.RWMutex
	domains      map[ID]Domain
	certificates map[ID]Certificate
	previews     map[ID]Preview
	environments map[string]EnvironmentConfig
	snapshots    map[ID]Snapshot
	Certificates CertificateProvider
	Storage      StorageProvider
}

func NewCatalog() *Catalog {
	return &Catalog{domains: map[ID]Domain{}, certificates: map[ID]Certificate{}, previews: map[ID]Preview{}, environments: map[string]EnvironmentConfig{}, snapshots: map[ID]Snapshot{}}
}
func (c *Catalog) SaveDomain(_ context.Context, d Domain) (Domain, error) {
	if err := ValidateHostname(d.Hostname); err != nil {
		return Domain{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if d.ID == "" {
		d.ID = ID("domain_" + d.ProjectID + "_" + d.Hostname)
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	d.UpdatedAt = d.CreatedAt
	if d.Status == "" {
		d.Status = "PENDING"
	}
	if d.TLSStatus == "" {
		d.TLSStatus = "NOT_REQUESTED"
	}
	c.domains[d.ID] = d
	return d, nil
}
func (c *Catalog) Domains(_ context.Context, projectID string) []Domain {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []Domain{}
	for _, d := range c.domains {
		if d.ProjectID == projectID {
			out = append(out, d)
		}
	}
	return out
}
func (c *Catalog) IssueCertificate(ctx context.Context, d Domain) (Certificate, error) {
	if c.Certificates == nil {
		return Certificate{}, errors.New("certificate provider unavailable")
	}
	certificate, err := c.Certificates.Issue(ctx, d)
	if err != nil {
		return Certificate{}, err
	}
	c.mu.Lock()
	c.certificates[certificate.ID] = certificate
	c.mu.Unlock()
	return certificate, nil
}
func (c *Catalog) SavePreview(_ context.Context, p Preview) (Preview, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p.ID == "" {
		p.ID = ID("preview_" + p.ProjectID + "_" + p.Hostname)
	}
	c.previews[p.ID] = p
	return p, nil
}
func (c *Catalog) Previews(_ context.Context, projectID string) []Preview {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []Preview{}
	for _, p := range c.previews {
		if p.ProjectID == projectID {
			out = append(out, p)
		}
	}
	return out
}
