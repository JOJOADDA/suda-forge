package knowledge

import (
	"context"
	"testing"
)

func TestGraphRelationshipsAreStructuredAndQueryable(t *testing.T) {
	s := NewMemoryStore(nil)
	_, err := s.UpsertNode(context.Background(), Node{ID: "page.home", ProjectID: "p1", Type: Page, Name: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpsertNode(context.Background(), Node{ID: "component.button", ProjectID: "p1", Type: Component, Name: "Button"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpsertNode(context.Background(), Node{ID: "token.primary", ProjectID: "p1", Type: DesignToken, Name: "color.primary"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpsertEdge(context.Background(), Edge{ID: "home.button", ProjectID: "p1", From: "page.home", To: "component.button", Type: Uses}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpsertEdge(context.Background(), Edge{ID: "button.token", ProjectID: "p1", From: "component.button", To: "token.primary", Type: Uses}); err != nil {
		t.Fatal(err)
	}
	neighbors, _ := s.Neighbors(context.Background(), "p1", "component.button", Uses)
	if len(neighbors) != 1 || neighbors[0].ID != "token.primary" {
		t.Fatalf("unexpected neighbors: %#v", neighbors)
	}
	g, _ := s.Graph(context.Background(), "p1")
	if len(g.Nodes) != 3 || len(g.Edges) != 2 {
		t.Fatalf("unexpected graph: %#v", g)
	}
}
