package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) UpsertNode(ctx context.Context, n Node) (Node, error) {
	if s.DB == nil {
		return Node{}, errors.New("knowledge database unavailable")
	}
	attrs, _ := json.Marshal(n.Attributes)
	_, err := s.DB.Exec(ctx, `INSERT INTO knowledge_nodes(id,project_id,node_type,name,path,attributes,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,path=EXCLUDED.path,attributes=EXCLUDED.attributes,updated_at=EXCLUDED.updated_at`, string(n.ID), n.ProjectID, string(n.Type), n.Name, n.Path, attrs, n.CreatedAt, n.UpdatedAt)
	return n, err
}
func (s PostgresStore) UpsertEdge(ctx context.Context, e Edge) (Edge, error) {
	if s.DB == nil {
		return Edge{}, errors.New("knowledge database unavailable")
	}
	attrs, _ := json.Marshal(e.Attributes)
	err := s.DB.QueryRow(ctx, `INSERT INTO knowledge_edges(id,project_id,from_node,to_node,edge_type,attributes,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(project_id,from_node,to_node,edge_type) DO UPDATE SET attributes=EXCLUDED.attributes RETURNING id,created_at`, string(e.ID), e.ProjectID, string(e.From), string(e.To), string(e.Type), attrs, e.CreatedAt).Scan(&e.ID, &e.CreatedAt)
	return e, err
}
func (s PostgresStore) Graph(ctx context.Context, project string) (Graph, error) {
	if s.DB == nil {
		return Graph{}, errors.New("knowledge database unavailable")
	}
	g := Graph{ProjectID: project, Nodes: []Node{}, Edges: []Edge{}}
	rows, err := s.DB.Query(ctx, `SELECT id,project_id,node_type,name,path,attributes,created_at,updated_at FROM knowledge_nodes WHERE project_id=$1 ORDER BY id`, project)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var n Node
		var attrs []byte
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &n.Name, &n.Path, &attrs, &n.CreatedAt, &n.UpdatedAt); err != nil {
			rows.Close()
			return g, err
		}
		_ = json.Unmarshal(attrs, &n.Attributes)
		g.Nodes = append(g.Nodes, n)
	}
	rows.Close()
	rows, err = s.DB.Query(ctx, `SELECT id,project_id,from_node,to_node,edge_type,attributes,created_at FROM knowledge_edges WHERE project_id=$1 ORDER BY id`, project)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Edge
		var attrs []byte
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.From, &e.To, &e.Type, &attrs, &e.CreatedAt); err != nil {
			return g, err
		}
		_ = json.Unmarshal(attrs, &e.Attributes)
		g.Edges = append(g.Edges, e)
	}
	return g, rows.Err()
}
func (s PostgresStore) Neighbors(ctx context.Context, project string, id NodeID, typ EdgeType) ([]Node, error) {
	if s.DB == nil {
		return nil, errors.New("knowledge database unavailable")
	}
	query := `SELECT n.id,n.project_id,n.node_type,n.name,n.path,n.attributes,n.created_at,n.updated_at FROM knowledge_nodes n JOIN knowledge_edges e ON e.to_node=n.id AND e.project_id=n.project_id WHERE e.project_id=$1 AND e.from_node=$2`
	args := []any{project, string(id)}
	if typ != "" {
		query += " AND e.edge_type=$3"
		args = append(args, string(typ))
	}
	query += " ORDER BY n.id"
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Node{}
	for rows.Next() {
		var n Node
		var attrs []byte
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &n.Name, &n.Path, &attrs, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(attrs, &n.Attributes)
		out = append(out, n)
	}
	return out, rows.Err()
}

var _ = time.Time{}
