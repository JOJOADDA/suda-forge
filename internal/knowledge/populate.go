package knowledge

import (
	"context"
	"errors"
	"time"
)

// Snapshot is the canonical cross-domain input for graph population. It keeps
// the graph authoritative while allowing Project Intelligence, Design,
// Agents, Verification, Git, and Deployment projections to contribute nodes.
type Snapshot struct {
	ProjectID string
	Nodes     []Node
	Edges     []Edge
}

func Populate(ctx context.Context, store Store, snapshot Snapshot) (Graph, error) {
	if store == nil {
		return Graph{}, errors.New("knowledge store is required")
	}
	if snapshot.ProjectID == "" {
		return Graph{}, errors.New("project_id is required")
	}
	now := time.Now().UTC()
	for _, n := range snapshot.Nodes {
		if n.ProjectID == "" {
			n.ProjectID = snapshot.ProjectID
		}
		if n.ProjectID != snapshot.ProjectID || n.ID == "" || n.Type == "" || n.Name == "" {
			return Graph{}, errors.New("invalid knowledge node")
		}
		if n.CreatedAt.IsZero() {
			n.CreatedAt = now
		}
		if n.UpdatedAt.IsZero() {
			n.UpdatedAt = now
		}
		if _, err := store.UpsertNode(ctx, n); err != nil {
			return Graph{}, err
		}
	}
	for _, e := range snapshot.Edges {
		if e.ProjectID == "" {
			e.ProjectID = snapshot.ProjectID
		}
		if e.ProjectID != snapshot.ProjectID || e.ID == "" || e.From == "" || e.To == "" || e.Type == "" {
			return Graph{}, errors.New("invalid knowledge edge")
		}
		if e.CreatedAt.IsZero() {
			e.CreatedAt = now
		}
		if _, err := store.UpsertEdge(ctx, e); err != nil {
			return Graph{}, err
		}
	}
	return store.Graph(ctx, snapshot.ProjectID)
}
