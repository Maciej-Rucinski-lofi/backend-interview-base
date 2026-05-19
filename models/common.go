package models

import (
	"context"
	"strings"
)

// RequestCommons is embedded in every {Resource}Args struct. It carries the
// pagination, sorting, filtering, and "include" parameters shared by every
// list endpoint. This is the single most important pattern in the codebase
// to recognise: a controller binds request parameters straight into Args, the
// service forwards Args to the repository, and the repository uses Args to
// shape the SQL it generates.
//
// Compare to deskapi's models.RequestCommons — same shape, much smaller.
type RequestCommons struct {
	// Single id lookup (used by Get) or filter alongside list filters.
	ID int64 `json:"id"     query:"id"`

	// Pagination. The repo enforces a sensible default and a max.
	Page     int `json:"page"     query:"page"`
	PageSize int `json:"pageSize" query:"pageSize"`

	// Sorting.
	OrderBy   string `json:"orderBy"   query:"orderBy"`
	OrderMode string `json:"orderMode" query:"orderMode"` // "asc" or "desc"

	// Filtering — see filter.go.
	Filter FilterQuery `json:"filter" query:"filter"`

	// Includes is the comma-separated list of relationships the caller wants
	// inlined under the response's `included` field. e.g. ?include=authors.
	Includes Includer `json:"include" query:"include"`

	// HardDelete is set on DELETE endpoints to skip the soft-delete and
	// permanently remove the record. Most callers should leave this off.
	HardDelete bool `json:"-" query:"hardDelete"`
}

// ApplyDefaults mutates the args so the rest of the pipeline can rely on
// sensible values. Call this once at the top of the repository's List
// method (or earlier in the service if validation needs the defaults).
func (r *RequestCommons) ApplyDefaults(_ context.Context) {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = 50
	}
	if r.PageSize > 200 {
		r.PageSize = 200
	}
	if r.OrderMode == "" {
		r.OrderMode = "asc"
	}
	r.OrderMode = strings.ToLower(r.OrderMode)
	if r.OrderMode != "asc" && r.OrderMode != "desc" {
		r.OrderMode = "asc"
	}
}

// Offset is a small helper so the repository doesn't repeat the math.
func (r *RequestCommons) Offset() int {
	return (r.Page - 1) * r.PageSize
}

// Pagination is what gets returned to the client alongside the records.
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// Included is the response container for eager-loaded relationships. Each
// resource type lives under its own key. We only put what the caller asked
// for via `?include=` here.
type Included struct {
	Authors    []*Author    `json:"authors,omitempty"`
	Publishers []*Publisher `json:"publishers,omitempty"`
}

// Includer parses the comma-separated `include` query value into a small set.
// Keep the surface area tiny: Set / IsOn / String. This keeps the common
// pattern `if args.Includes.IsOn(IncludeAuthors) { ... }` very readable.
type Includer struct {
	values map[string]struct{}
}

// Include constants — every model exports the strings that callers can pass
// in `?include=`.
const (
	IncludeAuthors    = "authors"
	IncludePublishers = "publishers"
)

// NewIncluder is sugar for service code that wants to ask for an include
// programmatically.
func NewIncluder(names ...string) Includer {
	i := Includer{}
	for _, n := range names {
		i.Set(n)
	}
	return i
}

// Set adds an include name.
func (i *Includer) Set(name string) {
	if i.values == nil {
		i.values = map[string]struct{}{}
	}
	i.values[name] = struct{}{}
}

// IsOn reports whether the caller asked for `name` to be included.
func (i Includer) IsOn(name string) bool {
	_, ok := i.values[name]
	return ok
}

// UnmarshalJSON accepts a comma-separated string from `?include=...&...`
// (Echo passes query params as JSON-decoded strings).
func (i *Includer) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	return i.parse(s)
}

// UnmarshalParam lets Echo's query binder populate the field directly.
func (i *Includer) UnmarshalParam(s string) error {
	return i.parse(s)
}

func (i *Includer) parse(s string) error {
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			i.Set(part)
		}
	}
	return nil
}
