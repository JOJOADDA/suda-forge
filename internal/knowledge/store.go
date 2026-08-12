package knowledge

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.RWMutex
	nodes map[NodeID]Node
	edges map[EdgeID]Edge
	now   func() time.Time
}

func NewMemoryStore(now func() time.Time) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	return &MemoryStore{nodes: map[NodeID]Node{}, edges: map[EdgeID]Edge{}, now: now}
}
func (s *MemoryStore) UpsertNode(_ context.Context, n Node) (Node, error) {
	if n.ProjectID == "" || n.ID == "" || n.Type == "" || n.Name == "" {
		return Node{}, errors.New("node project, id, type, and name are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	if old, ok := s.nodes[n.ID]; ok {
		n.CreatedAt = old.CreatedAt
	} else {
		n.CreatedAt = now
	}
	n.UpdatedAt = now
	s.nodes[n.ID] = n
	return n, nil
}
func (s *MemoryStore) UpsertEdge(_ context.Context, e Edge) (Edge, error) {
	if e.ProjectID == "" || e.ID == "" || e.From == "" || e.To == "" || e.Type == "" {
		return Edge{}, errors.New("edge project, id, from, to, and type are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[e.From]; !ok {
		return Edge{}, errors.New("edge source node not found")
	}
	if _, ok := s.nodes[e.To]; !ok {
		return Edge{}, errors.New("edge target node not found")
	}
	if old, ok := s.edges[e.ID]; ok {
		e.CreatedAt = old.CreatedAt
	} else {
		e.CreatedAt = s.now().UTC()
	}
	s.edges[e.ID] = e
	return e, nil
}
func (s *MemoryStore) Graph(_ context.Context, project string) (Graph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g := Graph{ProjectID: project, Nodes: []Node{}, Edges: []Edge{}}
	for _, n := range s.nodes {
		if n.ProjectID == project {
			g.Nodes = append(g.Nodes, n)
		}
	}
	for _, e := range s.edges {
		if e.ProjectID == project {
			g.Edges = append(g.Edges, e)
		}
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool { return g.Edges[i].ID < g.Edges[j].ID })
	return g, nil
}
func (s *MemoryStore) Neighbors(ctx context.Context, project string, id NodeID, typ EdgeType) ([]Node, error) {
	g, _ := s.Graph(ctx, project)
	ids := map[NodeID]bool{}
	for _, e := range g.Edges {
		if e.From == id && (typ == "" || e.Type == typ) {
			ids[e.To] = true
		}
	}
	out := []Node{}
	for _, n := range g.Nodes {
		if ids[n.ID] {
			out = append(out, n)
		}
	}
	return out, nil
}

var _ Store = (*MemoryStore)(nil)
