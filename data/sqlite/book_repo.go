package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"library-api/models"
)

// BookRepository is the SQLite-backed implementation of services.BookRepository.
type BookRepository struct {
	db *sql.DB
}

func NewBookRepository(db *sql.DB) *BookRepository {
	return &BookRepository{db: db}
}

// List returns books matching args plus the total row count.
func (r *BookRepository) List(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error) {
	args.ApplyDefaults(ctx)

	where, params, err := buildBookWhere(args)
	if err != nil {
		return nil, 0, err
	}

	order := buildOrderBy(&args.RequestCommons, models.Book{}.FilterFieldMap(), "books.id")

	q := `SELECT id, title, isbn, pageCount, genre, authors_id, state,
		createdAt, updatedAt, deletedAt,
		createdBy_users_id, updatedBy_users_id, deletedBy_users_id
		FROM books ` + where + ` ` + order + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, append(params, args.PageSize, args.Offset())...)
	if err != nil {
		return nil, 0, fmt.Errorf("books list: %w", err)
	}
	defer rows.Close()

	out := []*models.Book{}
	for rows.Next() {
		b := &models.Book{
			RelAuthor: &models.Relationship{Type: "authors"},
			Meta: models.Meta{
				RelCreatedBy: &models.Relationship{Type: "users"},
				RelUpdatedBy: &models.Relationship{Type: "users"},
				RelDeletedBy: &models.Relationship{Type: "users"},
			},
		}
		if err := rows.Scan(
			&b.ID, &b.Title, &b.ISBN, &b.PageCount, &b.Genre,
			b.RelAuthor, &b.State,
			&b.CreatedAt, &b.UpdatedAt, &b.DeletedAt,
			b.RelCreatedBy, b.RelUpdatedBy, b.RelDeletedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("books scan: %w", err)
		}
		clearBookEmptyRelationships(b)
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("books rows: %w", err)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books `+where, params...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("books count: %w", err)
	}
	return out, total, nil
}

func (r *BookRepository) Create(ctx context.Context, b *models.Book) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO books
			(title, isbn, pageCount, genre, authors_id, state,
			 createdAt, updatedAt,
			 createdBy_users_id, updatedBy_users_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.ISBN, b.PageCount, b.Genre, b.RelAuthor, b.State,
		b.CreatedAt, b.UpdatedAt,
		b.RelCreatedBy, b.RelUpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("books insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("books last id: %w", err)
	}
	b.ID = id
	return nil
}

func (r *BookRepository) Update(ctx context.Context, b *models.Book) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE books SET
			title = ?, isbn = ?, pageCount = ?, genre = ?,
			authors_id = ?, state = ?,
			updatedAt = ?, deletedAt = ?,
			updatedBy_users_id = ?, deletedBy_users_id = ?
		 WHERE id = ?`,
		b.Title, b.ISBN, b.PageCount, b.Genre,
		b.RelAuthor, b.State,
		b.UpdatedAt, b.DeletedAt,
		b.RelUpdatedBy, b.RelDeletedBy,
		b.ID,
	)
	if err != nil {
		return fmt.Errorf("books update: %w", err)
	}
	return nil
}

func (r *BookRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("books delete: %w", err)
	}
	return nil
}

func buildBookWhere(args *models.BookArgs) (string, []any, error) {
	parts, params, hasStateFilter, err := commonClauses(args.RequestCommons, models.Book{}.FilterFieldMap())
	if err != nil {
		return "", nil, err
	}
	if !hasStateFilter {
		parts = append(parts, "books.state = ?")
		params = append(params, models.StateActive)
	}
	if args.ID != 0 {
		parts = append(parts, "books.id = ?")
		params = append(params, args.ID)
	}
	if args.AuthorID != 0 {
		parts = append(parts, "books.authors_id = ?")
		params = append(params, args.AuthorID)
	}
	if len(parts) == 0 {
		return "", nil, nil
	}
	return "WHERE " + strings.Join(parts, " AND "), params, nil
}

func clearBookEmptyRelationships(b *models.Book) {
	if b.RelAuthor != nil && b.RelAuthor.ID == 0 {
		b.RelAuthor = nil
	}
	if b.RelCreatedBy != nil && b.RelCreatedBy.ID == 0 {
		b.RelCreatedBy = nil
	}
	if b.RelUpdatedBy != nil && b.RelUpdatedBy.ID == 0 {
		b.RelUpdatedBy = nil
	}
	if b.RelDeletedBy != nil && b.RelDeletedBy.ID == 0 {
		b.RelDeletedBy = nil
	}
}

