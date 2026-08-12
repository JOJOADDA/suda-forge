package designintelligence

import (
	"context"
	"encoding/json"
	"errors"
)

func (s PostgresStore) Get(ctx context.Context, projectID string) (DesignSystem, error) {
	if s.DB == nil {
		return DesignSystem{}, errors.New("design intelligence database unavailable")
	}
	var d DesignSystem
	var typography, colors, spacing, radius, shadows, breakpoints []byte
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,name,version,typography,color_palette,spacing,radius,shadows,breakpoints,created_at,updated_at FROM design_systems WHERE project_id=$1 ORDER BY updated_at DESC LIMIT 1`, projectID).Scan(&d.ID, &d.ProjectID, &d.Name, &d.Version, &typography, &colors, &spacing, &radius, &shadows, &breakpoints, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, err
	}
	_ = json.Unmarshal(typography, &d.Typography)
	_ = json.Unmarshal(colors, &d.ColorPalette)
	_ = json.Unmarshal(spacing, &d.Spacing)
	_ = json.Unmarshal(radius, &d.Radius)
	_ = json.Unmarshal(shadows, &d.Shadows)
	_ = json.Unmarshal(breakpoints, &d.Breakpoints)
	tokenRows, err := s.DB.Query(ctx, `SELECT id,name,kind,value,description,semantic FROM design_tokens WHERE design_system_id=$1 ORDER BY id`, string(d.ID))
	if err != nil {
		return d, err
	}
	for tokenRows.Next() {
		var t DesignToken
		if err := tokenRows.Scan(&t.ID, &t.Name, &t.Kind, &t.Value, &t.Description, &t.Semantic); err != nil {
			tokenRows.Close()
			return d, err
		}
		d.Tokens = append(d.Tokens, t)
	}
	tokenRows.Close()
	componentRows, err := s.DB.Query(ctx, `SELECT id,name,description,dependencies,used_by,pages,tests FROM design_components WHERE design_system_id=$1 ORDER BY id`, string(d.ID))
	if err != nil {
		return d, err
	}
	for componentRows.Next() {
		var c ComponentDefinition
		var deps, used, pages, tests []byte
		if err := componentRows.Scan(&c.ID, &c.Name, &c.Description, &deps, &used, &pages, &tests); err != nil {
			componentRows.Close()
			return d, err
		}
		_ = json.Unmarshal(deps, &c.Dependencies)
		_ = json.Unmarshal(used, &c.UsedBy)
		_ = json.Unmarshal(pages, &c.Pages)
		_ = json.Unmarshal(tests, &c.Tests)
		d.Components = append(d.Components, c)
	}
	componentRows.Close()
	return d, nil
}
