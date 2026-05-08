package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"library-api/models"
)

// AuthorRepository is the SQLite-backed implementation of services.AuthorRepository.
//
// In deskapi the production repos are generated wrappers around gorp; here we
// hand-write the SQL so candidates can read it directly. The shape — one
// repo per resource, with List/Create/Update/Delete — is the same.
type AuthorRepository struct {
	db *sql.DB
}

// NewAuthorRepository returns an AuthorRepository backed by db.
func NewAuthorRepository(db *sql.DB) *AuthorRepository {
	return &AuthorRepository{db: db}
}

// List returns the authors matching args, plus a total count for pagination.
//
// The flow mirrors deskapi's repo.List: apply defaults, build WHERE / ORDER BY
// / LIMIT from Args, run two queries (records + count), return both.
func (r *AuthorRepository) List(ctx context.Context, args *models.AuthorArgs) ([]*models.Author, int64, error) {
	args.ApplyDefaults(ctx)

	where, params, err := buildAuthorWhere(args)
	if err != nil {
		return nil, 0, err
	}

	order := buildOrderBy(&args.RequestCommons, models.Author{}.FilterFieldMap(), "authors.id")

	q := `SELECT id, name, bio, state, createdAt, updatedAt, deletedAt,
		createdBy_users_id, updatedBy_users_id, deletedBy_users_id
		FROM authors ` + where + ` ` + order + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, append(params, args.PageSize, args.Offset())...)
	if err != nil {
		return nil, 0, fmt.Errorf("authors list: %w", err)
	}
	defer rows.Close()

	out := []*models.Author{}
	for rows.Next() {
		a := &models.Author{
			Meta: models.Meta{
				RelCreatedBy: &models.Relationship{Type: "users"},
				RelUpdatedBy: &models.Relationship{Type: "users"},
				RelDeletedBy: &models.Relationship{Type: "users"},
			},
		}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Bio, &a.State,
			&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
			a.RelCreatedBy, a.RelUpdatedBy, a.RelDeletedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("authors scan: %w", err)
		}
		clearEmptyRelationships(a)
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("authors rows: %w", err)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM authors `+where, params...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("authors count: %w", err)
	}
	return out, total, nil
}

// ListByIDs is the eager-load helper used by includes. It bypasses the
// Args/Filter machinery because we always want the lookup keyed on a known
// ID set.
func (r *AuthorRepository) ListByIDs(ctx context.Context, ids []int64) ([]*models.Author, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	q := `SELECT id, name, bio, state, createdAt, updatedAt, deletedAt,
		createdBy_users_id, updatedBy_users_id, deletedBy_users_id
		FROM authors WHERE id IN (` + placeholders + `) AND state = 'active'`
	params := make([]any, 0, len(ids))
	for _, id := range ids {
		params = append(params, id)
	}
	rows, err := r.db.QueryContext(ctx, q, params...)
	if err != nil {
		return nil, fmt.Errorf("authors by ids: %w", err)
	}
	defer rows.Close()
	out := []*models.Author{}
	for rows.Next() {
		a := &models.Author{
			Meta: models.Meta{
				RelCreatedBy: &models.Relationship{Type: "users"},
				RelUpdatedBy: &models.Relationship{Type: "users"},
				RelDeletedBy: &models.Relationship{Type: "users"},
			},
		}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Bio, &a.State,
			&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
			a.RelCreatedBy, a.RelUpdatedBy, a.RelDeletedBy,
		); err != nil {
			return nil, err
		}
		clearEmptyRelationships(a)
		out = append(out, a)
	}
	return out, rows.Err()
}

// Create inserts a new author and updates a.ID on success.
func (r *AuthorRepository) Create(ctx context.Context, a *models.Author) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO authors
			(name, bio, state, createdAt, updatedAt,
			 createdBy_users_id, updatedBy_users_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.Name, a.Bio, a.State, a.CreatedAt, a.UpdatedAt,
		a.RelCreatedBy, a.RelUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("authors insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("authors last id: %w", err)
	}
	a.ID = id
	return nil
}

// Update writes back every business-relevant column. The repo trusts that
// the service has populated the Meta fields correctly.
func (r *AuthorRepository) Update(ctx context.Context, a *models.Author) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE authors SET
			name = ?, bio = ?, state = ?,
			updatedAt = ?, deletedAt = ?,
			updatedBy_users_id = ?, deletedBy_users_id = ?
		 WHERE id = ?`,
		a.Name, a.Bio, a.State,
		a.UpdatedAt, a.DeletedAt,
		a.RelUpdatedBy, a.RelDeletedBy,
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("authors update: %w", err)
	}
	return nil
}

// Delete is the hard-delete escape hatch. The default delete path goes
// through Update with state="deleted".
func (r *AuthorRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM authors WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("authors delete: %w", err)
	}
	return nil
}

// buildAuthorWhere assembles the WHERE clause from Args. It always scopes to
// state="active" unless the caller is filtering on state explicitly — this
// keeps soft-deleted records hidden by default, the same default deskapi
// applies via RequestCommons.
func buildAuthorWhere(args *models.AuthorArgs) (string, []any, error) {
	parts, params, hasStateFilter, err := commonClauses(args.RequestCommons, models.Author{}.FilterFieldMap())
	if err != nil {
		return "", nil, err
	}
	if !hasStateFilter {
		parts = append(parts, "authors.state = ?")
		params = append(params, models.StateActive)
	}
	if args.ID != 0 {
		parts = append(parts, "authors.id = ?")
		params = append(params, args.ID)
	}
	if len(parts) == 0 {
		return "", nil, nil
	}
	return "WHERE " + strings.Join(parts, " AND "), params, nil
}

// clearEmptyRelationships nils out Rel* fields whose underlying ID is zero
// so the JSON `omitempty` tag works correctly.
func clearEmptyRelationships(a *models.Author) {
	if a.RelCreatedBy != nil && a.RelCreatedBy.ID == 0 {
		a.RelCreatedBy = nil
	}
	if a.RelUpdatedBy != nil && a.RelUpdatedBy.ID == 0 {
		a.RelUpdatedBy = nil
	}
	if a.RelDeletedBy != nil && a.RelDeletedBy.ID == 0 {
		a.RelDeletedBy = nil
	}
}
