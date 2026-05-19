package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"library-api/models"
)

// PublisherRepository is the SQLite-backed implementation of services.PublisherRepository.
type PublisherRepository struct {
	db *sql.DB
}

func NewPublisherRepository(db *sql.DB) *PublisherRepository {
	return &PublisherRepository{db: db}
}

func (r *PublisherRepository) List(ctx context.Context, args *models.PublisherArgs) ([]*models.Publisher, int64, error) {
	args.ApplyDefaults(ctx)

	where, params, err := buildPublisherWhere(args)
	if err != nil {
		return nil, 0, err
	}

	order := buildOrderBy(&args.RequestCommons, models.Publisher{}.FilterFieldMap(), "publishers.id")

	q := `SELECT id, name, country, state, createdAt, updatedAt, deletedAt,
		createdBy_users_id, updatedBy_users_id, deletedBy_users_id
		FROM publishers ` + where + ` ` + order + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, append(params, args.PageSize, args.Offset())...)
	if err != nil {
		return nil, 0, fmt.Errorf("publishers list: %w", err)
	}
	defer rows.Close()

	out := []*models.Publisher{}
	for rows.Next() {
		p := &models.Publisher{
			Meta: models.Meta{
				RelCreatedBy: &models.Relationship{Type: "users"},
				RelUpdatedBy: &models.Relationship{Type: "users"},
				RelDeletedBy: &models.Relationship{Type: "users"},
			},
		}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Country, &p.State,
			&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
			p.RelCreatedBy, p.RelUpdatedBy, p.RelDeletedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("publishers scan: %w", err)
		}
		clearPublisherEmptyRelationships(p)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("publishers rows: %w", err)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM publishers `+where, params...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("publishers count: %w", err)
	}
	return out, total, nil
}

func (r *PublisherRepository) ListByIDs(ctx context.Context, ids []int64) ([]*models.Publisher, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	q := `SELECT id, name, country, state, createdAt, updatedAt, deletedAt,
		createdBy_users_id, updatedBy_users_id, deletedBy_users_id
		FROM publishers WHERE id IN (` + placeholders + `) AND state = 'active'`
	params := make([]any, 0, len(ids))
	for _, id := range ids {
		params = append(params, id)
	}
	rows, err := r.db.QueryContext(ctx, q, params...)
	if err != nil {
		return nil, fmt.Errorf("publishers by ids: %w", err)
	}
	defer rows.Close()
	out := []*models.Publisher{}
	for rows.Next() {
		p := &models.Publisher{
			Meta: models.Meta{
				RelCreatedBy: &models.Relationship{Type: "users"},
				RelUpdatedBy: &models.Relationship{Type: "users"},
				RelDeletedBy: &models.Relationship{Type: "users"},
			},
		}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Country, &p.State,
			&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
			p.RelCreatedBy, p.RelUpdatedBy, p.RelDeletedBy,
		); err != nil {
			return nil, err
		}
		clearPublisherEmptyRelationships(p)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PublisherRepository) Create(ctx context.Context, p *models.Publisher) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO publishers
			(name, country, state, createdAt, updatedAt,
			 createdBy_users_id, updatedBy_users_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Country, p.State, p.CreatedAt, p.UpdatedAt,
		p.RelCreatedBy, p.RelUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("publishers insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("publishers last id: %w", err)
	}
	p.ID = id
	return nil
}

func (r *PublisherRepository) Update(ctx context.Context, p *models.Publisher) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE publishers SET
			name = ?, country = ?, state = ?,
			updatedAt = ?, deletedAt = ?,
			updatedBy_users_id = ?, deletedBy_users_id = ?
		 WHERE id = ?`,
		p.Name, p.Country, p.State,
		p.UpdatedAt, p.DeletedAt,
		p.RelUpdatedBy, p.RelDeletedBy,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("publishers update: %w", err)
	}
	return nil
}

func (r *PublisherRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM publishers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("publishers delete: %w", err)
	}
	return nil
}

func buildPublisherWhere(args *models.PublisherArgs) (string, []any, error) {
	parts, params, hasStateFilter, err := commonClauses(args.RequestCommons, models.Publisher{}.FilterFieldMap())
	if err != nil {
		return "", nil, err
	}
	if !hasStateFilter {
		parts = append(parts, "publishers.state = ?")
		params = append(params, models.StateActive)
	}
	if args.ID != 0 {
		parts = append(parts, "publishers.id = ?")
		params = append(params, args.ID)
	}
	if len(parts) == 0 {
		return "", nil, nil
	}
	return "WHERE " + strings.Join(parts, " AND "), params, nil
}

func clearPublisherEmptyRelationships(p *models.Publisher) {
	if p.RelCreatedBy != nil && p.RelCreatedBy.ID == 0 {
		p.RelCreatedBy = nil
	}
	if p.RelUpdatedBy != nil && p.RelUpdatedBy.ID == 0 {
		p.RelUpdatedBy = nil
	}
	if p.RelDeletedBy != nil && p.RelDeletedBy.ID == 0 {
		p.RelDeletedBy = nil
	}
}
