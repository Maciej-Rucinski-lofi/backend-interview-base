package sqlite

import (
	"fmt"

	"library-api/models"
)

// commonClauses turns the shared filter fields on RequestCommons into a slice
// of SQL fragments + params. Per-resource buildXWhere functions extend the
// returned slice with anything model-specific (typed args, ID lookup, etc.).
//
// It also returns hasStateFilter so the caller knows whether it still needs
// to add the default `state = 'active'` clause.
func commonClauses(rc models.RequestCommons, fieldMap map[string]string) ([]string, []any, bool, error) {
	var (
		parts          []string
		params         []any
		hasStateFilter bool
	)

	frag, fparams, err := rc.Filter.SQL(fieldMap)
	if err != nil {
		return nil, nil, false, err
	}
	if frag != "" {
		parts = append(parts, "("+frag+")")
		params = append(params, fparams...)
		for _, c := range rc.Filter.Clauses {
			if c.Name == "state" {
				hasStateFilter = true
				break
			}
		}
	}
	return parts, params, hasStateFilter, nil
}

// buildOrderBy translates the OrderBy/OrderMode pair into a safe ORDER BY
// fragment, falling back to fallback when OrderBy is empty or unknown. It
// uses the model's FilterFieldMap as the whitelist of sortable columns —
// candidates should immediately spot that the same map gates filtering and
// sorting (which is exactly the deskapi default).
func buildOrderBy(rc *models.RequestCommons, fieldMap map[string]string, fallback string) string {
	col := fallback
	if rc.OrderBy != "" {
		if mapped, ok := fieldMap[rc.OrderBy]; ok {
			col = mapped
		}
	}
	dir := "ASC"
	if rc.OrderMode == "desc" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s", col, dir)
}
