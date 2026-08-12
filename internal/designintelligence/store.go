package designintelligence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) Save(ctx context.Context, d DesignSystem) error {
	if s.DB == nil {
		return errors.New("design intelligence database unavailable")
	}
	typ, _ := json.Marshal(d.Typography)
	colors, _ := json.Marshal(d.ColorPalette)
	spacing, _ := json.Marshal(d.Spacing)
	radius, _ := json.Marshal(d.Radius)
	shadows, _ := json.Marshal(d.Shadows)
	breakpoints, _ := json.Marshal(d.Breakpoints)
	_, err := s.DB.Exec(ctx, `INSERT INTO design_systems(id,project_id,name,version,typography,color_palette,spacing,radius,shadows,breakpoints,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(id) DO UPDATE SET version=EXCLUDED.version,typography=EXCLUDED.typography,color_palette=EXCLUDED.color_palette,spacing=EXCLUDED.spacing,radius=EXCLUDED.radius,shadows=EXCLUDED.shadows,breakpoints=EXCLUDED.breakpoints,updated_at=EXCLUDED.updated_at`, string(d.ID), d.ProjectID, d.Name, d.Version, typ, colors, spacing, radius, shadows, breakpoints, d.CreatedAt, d.UpdatedAt)
	if err != nil {
		return err
	}
	for _, t := range d.Tokens {
		raw := t
		data := map[string]any{"id": string(raw.ID), "name": raw.Name, "kind": raw.Kind, "value": raw.Value, "description": raw.Description, "semantic": raw.Semantic}
		if err := s.saveToken(ctx, d.ID, data); err != nil {
			return err
		}
	}
	for _, c := range d.Components {
		if err := s.saveComponent(ctx, d.ID, c); err != nil {
			return err
		}
	}
	return nil
}
func (s PostgresStore) saveToken(ctx context.Context, system ID, v map[string]any) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO design_tokens(id,design_system_id,name,kind,value,description,semantic) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(design_system_id,id) DO UPDATE SET value=EXCLUDED.value,description=EXCLUDED.description,semantic=EXCLUDED.semantic`, v["id"], string(system), v["name"], v["kind"], v["value"], v["description"], v["semantic"])
	return err
}
func (s PostgresStore) saveComponent(ctx context.Context, system ID, c ComponentDefinition) error {
	deps, _ := json.Marshal(c.Dependencies)
	used, _ := json.Marshal(c.UsedBy)
	pages, _ := json.Marshal(c.Pages)
	tests, _ := json.Marshal(c.Tests)
	if _, err := s.DB.Exec(ctx, `INSERT INTO design_components(id,design_system_id,name,description,dependencies,used_by,pages,tests) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(design_system_id,id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,dependencies=EXCLUDED.dependencies,used_by=EXCLUDED.used_by,pages=EXCLUDED.pages,tests=EXCLUDED.tests`, string(c.ID), string(system), c.Name, c.Description, deps, used, pages, tests); err != nil {
		return err
	}
	for _, v := range c.Variants {
		props, _ := json.Marshal(v.Props)
		tokens, _ := json.Marshal(v.Tokens)
		if _, err := s.DB.Exec(ctx, `INSERT INTO design_component_variants(component_id,design_system_id,name,props,tokens) VALUES($1,$2,$3,$4,$5)`, string(c.ID), string(system), v.Name, props, tokens); err != nil {
			return err
		}
	}
	return nil
}
