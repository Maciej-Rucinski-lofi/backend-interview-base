package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FilterOp enumerates the operators the API accepts in `?filter=`. We keep
// the set deliberately small here; deskapi supports more (gt, gte, in, like,
// etc.) but the pattern is identical — a parsed list of {field, op, value}
// triples that the repository translates into SQL.
type FilterOp string

const (
	OpEq    FilterOp = "$eq"
	OpNotEq FilterOp = "$ne"
	OpLike  FilterOp = "$like"
	OpGt    FilterOp = "$gt"
	OpLt    FilterOp = "$lt"
	OpIn    FilterOp = "$in"
)

// FilterClause is one parsed clause from the request.
type FilterClause struct {
	Name  string   `json:"name"`
	Op    FilterOp `json:"op"`
	Value any      `json:"val"`
}

// FilterQuery is the full set of filter clauses for a single request. It's
// embedded in RequestCommons so every Args struct gets it for free, and the
// repository layer iterates over it to build a WHERE clause.
//
// Wire format (URL-encoded JSON):
//
//	?filter=[{"name":"genre","op":"$eq","val":"fiction"}]
type FilterQuery struct {
	Clauses []FilterClause
}

// Add is the programmatic builder service code uses to append a clause from
// the inside, e.g. when one service deletes records that reference another:
//
//	args := &models.BookArgs{}
//	args.Filter.Add("authors_id", filter.OpEq, authorID)
//
// This mirrors deskapi's `args.Filter.Add(field, op, val)`.
func (f *FilterQuery) Add(name string, op FilterOp, value any) {
	f.Clauses = append(f.Clauses, FilterClause{Name: name, Op: op, Value: value})
}

// UnmarshalJSON accepts the wire format described above.
func (f *FilterQuery) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	return json.Unmarshal(data, &f.Clauses)
}

// UnmarshalParam lets Echo's query binder populate the field directly from
// `?filter=<json>`. Without this method, the default binder would fail
// because FilterQuery isn't a basic type.
func (f *FilterQuery) UnmarshalParam(s string) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), &f.Clauses)
}

// MarshalJSON keeps round-trip behaviour clean for tests.
func (f FilterQuery) MarshalJSON() ([]byte, error) {
	if len(f.Clauses) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(f.Clauses)
}

// SQL renders the parsed clauses as a parameterised WHERE fragment plus its
// args, taking a FilterFieldMap so a model can rename or join a user-friendly
// field name onto the real database column. Returns ("", nil) when there are
// no clauses.
//
// Note: this implementation is intentionally simple. In production deskapi
// uses a richer query builder (sqlgen / squirrel). The pattern of "API field
// name -> DB column via FilterFieldMap" is the part candidates should
// recognise.
func (f FilterQuery) SQL(fieldMap map[string]string) (string, []any, error) {
	if len(f.Clauses) == 0 {
		return "", nil, nil
	}
	var (
		parts []string
		args  []any
	)
	for _, c := range f.Clauses {
		col, ok := fieldMap[c.Name]
		if !ok {
			// If a model didn't whitelist the field we reject the request
			// rather than silently allow arbitrary columns to be queried.
			return "", nil, fmt.Errorf("filter: field %q not allowed", c.Name)
		}
		switch c.Op {
		case OpEq:
			parts = append(parts, fmt.Sprintf("%s = ?", col))
			args = append(args, c.Value)
		case OpNotEq:
			parts = append(parts, fmt.Sprintf("%s <> ?", col))
			args = append(args, c.Value)
		case OpLike:
			parts = append(parts, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, c.Value)
		case OpGt:
			parts = append(parts, fmt.Sprintf("%s > ?", col))
			args = append(args, c.Value)
		case OpLt:
			parts = append(parts, fmt.Sprintf("%s < ?", col))
			args = append(args, c.Value)
		case OpIn:
			vals, ok := c.Value.([]any)
			if !ok || len(vals) == 0 {
				return "", nil, fmt.Errorf("filter: $in needs a non-empty array for %q", c.Name)
			}
			parts = append(parts, fmt.Sprintf("%s IN (%s)", col, strings.Repeat("?,", len(vals)-1)+"?"))
			args = append(args, vals...)
		default:
			return "", nil, fmt.Errorf("filter: unsupported op %q", c.Op)
		}
	}
	return strings.Join(parts, " AND "), args, nil
}
