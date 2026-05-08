package models

import "time"

// State values for the soft-delete pattern. Resources are never hard-deleted by
// default — Delete sets state="deleted" and stamps DeletedAt / DeletedByUserID.
const (
	StateActive  = "active"
	StateDeleted = "deleted"
)

// Meta is embedded into every resource and provides the standard audit trail
// (createdAt, updatedAt, deletedAt) plus the "who did it" relationships.
//
// In deskapi these come from a shared Meta struct and are populated by helper
// methods MetaCreate / MetaUpdate / MetaDelete. We replicate that pattern here
// so candidates see how cross-cutting fields are handled in one place rather
// than re-implemented per resource.
type Meta struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"            db:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"            db:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"            db:"deletedAt"`

	// "Rel" prefix mirrors deskapi: in storage these are foreign-key IDs, but
	// in JSON they render as Relationship objects so the client can use the
	// `?include=` mechanism to inline the related record.
	RelCreatedBy *Relationship `json:"createdBy,omitempty"           db:"createdBy_users_id"`
	RelUpdatedBy *Relationship `json:"updatedBy,omitempty"           db:"updatedBy_users_id"`
	RelDeletedBy *Relationship `json:"deletedBy,omitempty"           db:"deletedBy_users_id"`

	State string `json:"state,omitempty" db:"state"`
}

// MetaCreate stamps fields when a new record is created. The actor is the user
// performing the request (carried on Session); we keep this method on Meta so
// every service writes the audit fields the same way.
func (m *Meta) MetaCreate(actorUserID int64) {
	now := time.Now().UTC()
	m.CreatedAt = &now
	m.UpdatedAt = &now
	m.RelCreatedBy = NewRelationship("users", actorUserID)
	m.RelUpdatedBy = NewRelationship("users", actorUserID)
	if m.State == "" {
		m.State = StateActive
	}
}

// MetaUpdate stamps fields on update. Note we deliberately do not touch
// CreatedAt / RelCreatedBy.
func (m *Meta) MetaUpdate(actorUserID int64) {
	now := time.Now().UTC()
	m.UpdatedAt = &now
	m.RelUpdatedBy = NewRelationship("users", actorUserID)
}

// MetaDelete stamps the soft-delete fields. The repository is still
// responsible for persisting the change — this only mutates the in-memory
// model.
func (m *Meta) MetaDelete(actorUserID int64) {
	now := time.Now().UTC()
	m.UpdatedAt = &now
	m.DeletedAt = &now
	m.RelUpdatedBy = NewRelationship("users", actorUserID)
	m.RelDeletedBy = NewRelationship("users", actorUserID)
	m.State = StateDeleted
}
