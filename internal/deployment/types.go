package deployment

import (
	"context"
	"time"
)

type ID string
type Environment string

const (
	Development Environment = "development"
	PreviewEnv  Environment = "preview"
	Production  Environment = "production"
)

type ServiceStatus string

const (
	ServiceDiscovered ServiceStatus = "DISCOVERED"
	ServiceRunning    ServiceStatus = "RUNNING"
	ServiceUnhealthy  ServiceStatus = "UNHEALTHY"
	ServiceStopped    ServiceStatus = "STOPPED"
	ServiceFailed     ServiceStatus = "FAILED"
)

type Service struct {
	ID              ID             `json:"id"`
	ProjectID       string         `json:"project_id"`
	Name            string         `json:"name"`
	RuntimeID       string         `json:"runtime_id"`
	ProcessIdentity string         `json:"process_identity,omitempty"`
	Protocol        string         `json:"protocol"`
	Host            string         `json:"host"`
	Port            int            `json:"port"`
	HealthEndpoint  string         `json:"health_endpoint,omitempty"`
	Status          ServiceStatus  `json:"status"`
	Environment     Environment    `json:"environment"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
type ExposureMode string

const (
	Internal        ExposureMode = "INTERNAL"
	PreviewExposure ExposureMode = "PREVIEW"
	Public          ExposureMode = "PUBLIC"
)

type PortBinding struct {
	ID           ID           `json:"id"`
	ProjectID    string       `json:"project_id"`
	RuntimeID    string       `json:"runtime_id"`
	ServiceID    ID           `json:"service_id"`
	InternalPort int          `json:"internal_port"`
	ExternalPort int          `json:"external_port"`
	Protocol     string       `json:"protocol"`
	Exposure     ExposureMode `json:"exposure_mode"`
	Status       string       `json:"status"`
	Health       string       `json:"health"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
type Preview struct {
	ID           ID          `json:"id"`
	ProjectID    string      `json:"project_id"`
	ServiceID    ID          `json:"service_id"`
	Environment  Environment `json:"environment"`
	Hostname     string      `json:"hostname"`
	URL          string      `json:"url"`
	Status       string      `json:"status"`
	AccessPolicy string      `json:"access_policy"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
type Domain struct {
	ID                 ID          `json:"id"`
	ProjectID          string      `json:"project_id"`
	Hostname           string      `json:"hostname"`
	ServiceID          ID          `json:"service_id"`
	Environment        Environment `json:"environment"`
	Status             string      `json:"status"`
	TLSStatus          string      `json:"tls_status"`
	VerificationStatus string      `json:"verification_status"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}
type Certificate struct {
	ID        ID         `json:"id"`
	DomainID  ID         `json:"domain_id"`
	Status    string     `json:"status"`
	Issuer    string     `json:"issuer,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
type EnvironmentConfig struct {
	ProjectID        string            `json:"project_id"`
	Environment      Environment       `json:"environment"`
	Variables        map[string]string `json:"variables,omitempty"`
	SecretReferences []string          `json:"secret_references,omitempty"`
	CPU              int               `json:"cpu"`
	MemoryBytes      int64             `json:"memory_bytes"`
	DiskBytes        int64             `json:"disk_bytes"`
	DeploymentPolicy map[string]any    `json:"deployment_policy,omitempty"`
}
type ReleaseStatus string

const (
	ReleaseCreated ReleaseStatus = "CREATED"
	ReleaseReady   ReleaseStatus = "READY"
	ReleaseFailed  ReleaseStatus = "FAILED"
)

type Release struct {
	ID                   ID             `json:"id"`
	ProjectID            string         `json:"project_id"`
	GitRevision          string         `json:"git_revision"`
	ArtifactReferences   []string       `json:"artifact_references,omitempty"`
	BuildMetadata        map[string]any `json:"build_metadata,omitempty"`
	Environment          Environment    `json:"environment"`
	ConfigurationVersion string         `json:"configuration_version,omitempty"`
	VerificationRunID    string         `json:"verification_run_id,omitempty"`
	Status               ReleaseStatus  `json:"status"`
	CreatedAt            time.Time      `json:"created_at"`
}
type DeploymentStatus string

const (
	DeploymentPending     DeploymentStatus = "PENDING"
	DeploymentPreparing   DeploymentStatus = "PREPARING"
	DeploymentBuilding    DeploymentStatus = "BUILDING"
	DeploymentTesting     DeploymentStatus = "TESTING"
	DeploymentVerifying   DeploymentStatus = "VERIFYING"
	DeploymentDeploying   DeploymentStatus = "DEPLOYING"
	DeploymentHealthCheck DeploymentStatus = "HEALTH_CHECK"
	DeploymentActive      DeploymentStatus = "ACTIVE"
	DeploymentFailed      DeploymentStatus = "FAILED"
	DeploymentRolledBack  DeploymentStatus = "ROLLED_BACK"
	DeploymentCancelled   DeploymentStatus = "CANCELLED"
)

type DeploymentStrategy string

const (
	StrategyRecreate  DeploymentStrategy = "RECREATE"
	StrategyBlueGreen DeploymentStrategy = "BLUE_GREEN"
	StrategyRolling   DeploymentStrategy = "ROLLING"
)

type Deployment struct {
	ID             ID                 `json:"id"`
	ProjectID      string             `json:"project_id"`
	Environment    Environment        `json:"environment"`
	Version        string             `json:"version"`
	SourceRevision string             `json:"source_revision"`
	RuntimeTarget  string             `json:"runtime_target"`
	ReleaseID      ID                 `json:"release_id"`
	Strategy       DeploymentStrategy `json:"strategy"`
	Status         DeploymentStatus   `json:"status"`
	HealthStatus   string             `json:"health_status,omitempty"`
	FailureReason  string             `json:"failure_reason,omitempty"`
	Metadata       map[string]any     `json:"metadata,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	StartedAt      *time.Time         `json:"started_at,omitempty"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
}
type HealthCheckType string

const (
	HealthHTTP     HealthCheckType = "HTTP"
	HealthTCP      HealthCheckType = "TCP"
	HealthProcess  HealthCheckType = "PROCESS"
	HealthCommand  HealthCheckType = "COMMAND"
	HealthDatabase HealthCheckType = "DATABASE"
)

type HealthCheck struct {
	ID        ID     `json:"id"`
	RuntimeID string `json:"runtime_id"`

	ProjectID        string          `json:"project_id"`
	ServiceID        ID              `json:"service_id"`
	Type             HealthCheckType `json:"type"`
	Target           string          `json:"target"`
	Timeout          time.Duration   `json:"timeout"`
	Interval         time.Duration   `json:"interval"`
	Retries          int             `json:"retries"`
	FailureThreshold int             `json:"failure_threshold"`
	SuccessThreshold int             `json:"success_threshold"`
	Status           string          `json:"status"`
	LastError        string          `json:"last_error,omitempty"`
	CheckedAt        time.Time       `json:"checked_at"`
}
type Snapshot struct {
	ID        ID        `json:"id"`
	ProjectID string    `json:"project_id"`
	Kind      string    `json:"kind"`
	Reference string    `json:"reference"`
	Source    string    `json:"source"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type ServiceDiscovery interface {
	Discover(context.Context, string) ([]Service, error)
}
type PortRegistry interface {
	Reserve(context.Context, PortBinding) (PortBinding, error)
	Release(context.Context, ID) error
	List(context.Context, string) ([]PortBinding, error)
}
type DeploymentProvider interface {
	Deploy(context.Context, Deployment, Release, EnvironmentConfig) error
	Stop(context.Context, Deployment) error
	Status(context.Context, Deployment) (DeploymentStatus, error)
}
type NetworkProvider interface {
	CheckPortConflict(context.Context, PortBinding) error
}
type ProxyProvider interface {
	CreateRoute(context.Context, Preview) error
	UpdateRoute(context.Context, Preview) error
	DeleteRoute(context.Context, ID) error
	URL(Preview) string
}
type CertificateProvider interface {
	Issue(context.Context, Domain) (Certificate, error)
	Renew(context.Context, Certificate) (Certificate, error)
	Status(context.Context, Certificate) (Certificate, error)
}
type StorageProvider interface {
	CreateVolume(context.Context, string, string) (string, error)
	Snapshot(context.Context, string, string) (Snapshot, error)
	Restore(context.Context, Snapshot) error
}
type HealthChecker interface {
	Check(context.Context, HealthCheck) (HealthCheck, error)
}
type AuditSink interface{ Publish(string, string, any) }
