package knowledge

import "time"

type NodeID string
type EdgeID string
type NodeType string

const (
	Project      NodeType = "Project"
	Requirement  NodeType = "Requirement"
	Architecture NodeType = "Architecture"
	Module       NodeType = "Module"
	File         NodeType = "File"
	Component    NodeType = "Component"
	Page         NodeType = "Page"
	API          NodeType = "API"
	Database     NodeType = "Database"
	Table        NodeType = "Table"
	DesignToken  NodeType = "DesignToken"
	Agent        NodeType = "Agent"
	Task         NodeType = "Task"
	Test         NodeType = "Test"
	Bug          NodeType = "Bug"
	Decision     NodeType = "Decision"
	Dependency   NodeType = "Dependency"
	Release      NodeType = "Release"
	Deployment   NodeType = "Deployment"
)

type EdgeType string

const (
	Contains    EdgeType = "contains"
	DependsOn   EdgeType = "depends_on"
	Uses        EdgeType = "uses"
	Calls       EdgeType = "calls"
	Reads       EdgeType = "reads"
	Writes      EdgeType = "writes"
	Implements  EdgeType = "implements"
	Tests       EdgeType = "tests"
	Owns        EdgeType = "owns"
	Modifies    EdgeType = "modifies"
	GeneratedBy EdgeType = "generated_by"
	DeployedAs  EdgeType = "deployed_as"
)

type Node struct {
	ID         NodeID            `json:"id"`
	ProjectID  string            `json:"project_id"`
	Type       NodeType          `json:"type"`
	Name       string            `json:"name"`
	Path       string            `json:"path,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}
type Edge struct {
	ID         EdgeID            `json:"id"`
	ProjectID  string            `json:"project_id"`
	From       NodeID            `json:"from"`
	To         NodeID            `json:"to"`
	Type       EdgeType          `json:"type"`
	Attributes map[string]string `json:"attributes,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}
type Graph struct {
	ProjectID string `json:"project_id"`
	Nodes     []Node `json:"nodes"`
	Edges     []Edge `json:"edges"`
}
type Store interface {
	UpsertNode(Node) (Node, error)
	UpsertEdge(Edge) (Edge, error)
	Graph(string) (Graph, error)
	Neighbors(string, NodeID, EdgeType) ([]Node, error)
}
