package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Relationship is the JSON-API-style "pointer" we use to refer to another
// resource without inlining it. In deskapi the equivalent type is in the
// shared models package and is used for every foreign-key field.
//
// On the wire it looks like: {"id": 123, "type": "authors"}
// In storage it's just an int64 column (e.g. authors_id).
//
// We implement sql.Scanner and driver.Valuer so a *Relationship can be used
// directly as a column target in queries.
type Relationship struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// NewRelationship is a tiny helper so callers don't accidentally forget Type.
func NewRelationship(typ string, id int64) *Relationship {
	if id == 0 {
		return nil
	}
	return &Relationship{ID: id, Type: typ}
}

// GetID returns the relationship's ID, or 0 if the relationship is nil. This
// matches the deskapi helper of the same name and lets callers avoid nil
// checks at every call site.
func (r *Relationship) GetID() int64 {
	if r == nil {
		return 0
	}
	return r.ID
}

// Scan implements sql.Scanner. If the column is NULL we leave the relationship
// as a zero-id pointer so JSON omits it cleanly via omitempty on the parent.
func (r *Relationship) Scan(src any) error {
	var n sql.NullInt64
	if err := n.Scan(src); err != nil {
		return err
	}
	if !n.Valid {
		r.ID = 0
		return nil
	}
	r.ID = n.Int64
	return nil
}

// Value implements driver.Valuer.
func (r *Relationship) Value() (driver.Value, error) {
	if r == nil || r.ID == 0 {
		return nil, nil
	}
	return r.ID, nil
}

// MarshalJSON outputs nil for a zero-id relationship so the caller can use
// `omitempty` on their field tag without the type leaking onto the wire.
func (r *Relationship) MarshalJSON() ([]byte, error) {
	if r == nil || r.ID == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	}{r.ID, r.Type})
}

// UnmarshalJSON accepts both the object form ({"id":1,"type":"authors"}) and
// a bare integer (1) so legacy clients work.
func (r *Relationship) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '{' {
		aux := struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		}{}
		if err := json.Unmarshal(data, &aux); err != nil {
			return err
		}
		r.ID, r.Type = aux.ID, aux.Type
		return nil
	}
	var id int64
	if err := json.Unmarshal(data, &id); err != nil {
		return fmt.Errorf("relationship: %w", err)
	}
	r.ID = id
	return nil
}
