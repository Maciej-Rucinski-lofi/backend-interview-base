package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"library-api/models"
)

// BookRepository is the SQLite-backed implementation of services.BookRepository.
type BookRepository struct {
	db *sql.DB
}

func NewBookRepository(db *sql.DB) *BookRepository {
	return &BookRepository{db: db}
}

// bookFrom is the common FROM clause. We always LEFT JOIN authors so that
// ?filter=[{"name":"author.name",...}] and ?orderBy=author.name work without
// any extra plumbing — the qualified column authors.name is already in scope.
const bookFrom = `FROM books LEFT JOIN authors ON books.authors_id = authors.id`

// List returns books matching args plus the total row count.
func (r *BookRepository) List(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error) {
	args.ApplyDefaults(ctx)

	where, params, err := buildBookWhere(args)
	if err != nil {
		return nil, 0, err
	}

	order := buildOrderBy(&args.RequestCommons, models.Book{}.FilterFieldMap(), "books.id")

	q := `SELECT books.id, books.title, books.isbn, books.pageCount, books.genre,
		books.authors_id, books.publishers_id, books.state,
		books.createdAt, books.updatedAt, books.deletedAt,
		books.createdBy_users_id, books.updatedBy_users_id, books.deletedBy_users_id
		` + bookFrom + ` ` + where + ` ` + order + ` LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, append(params, args.PageSize, args.Offset())...)
	if err != nil {
		return nil, 0, fmt.Errorf("books list: %w", err)
	}
	defer rows.Close()

	out := []*models.Book{}
	for rows.Next() {
		b := &models.Book{
			RelAuthor:    &models.Relationship{Type: "authors"},
			RelPublisher: &models.Relationship{Type: "publishers"},
			Meta: models.Meta{
				RelCreatedBy: &models.Relationship{Type: "users"},
				RelUpdatedBy: &models.Relationship{Type: "users"},
				RelDeletedBy: &models.Relationship{Type: "users"},
			},
		}
		if err := rows.Scan(
			&b.ID, &b.Title, &b.ISBN, &b.PageCount, &b.Genre,
			b.RelAuthor, b.RelPublisher, &b.State,
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
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) `+bookFrom+` `+where, params...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("books count: %w", err)
	}
	return out, total, nil
}

func (r *BookRepository) Create(ctx context.Context, b *models.Book) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO books
			(title, isbn, pageCount, genre, authors_id, publishers_id, state,
			 createdAt, updatedAt,
			 createdBy_users_id, updatedBy_users_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.ISBN, b.PageCount, b.Genre, b.RelAuthor, b.RelPublisher, b.State,
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
			authors_id = ?, publishers_id = ?, state = ?,
			updatedAt = ?, deletedAt = ?,
			updatedBy_users_id = ?, deletedBy_users_id = ?
		 WHERE id = ?`,
		b.Title, b.ISBN, b.PageCount, b.Genre,
		b.RelAuthor, b.RelPublisher, b.State,
		b.UpdatedAt, b.DeletedAt,
		b.RelUpdatedBy, b.RelDeletedBy,
		b.ID,
	)
	if err != nil {
		return fmt.Errorf("books update: %w", err)
	}
	return nil
}

// UpdateWithOptimisticLock updates the book only when its current updatedAt
// in the database matches expectedUpdatedAt. If another writer already
// committed a change between our read and this write, RowsAffected is 0 and
// we return a 409 Conflict so the client can retry.
func (r *BookRepository) UpdateWithOptimisticLock(ctx context.Context, b *models.Book, expectedUpdatedAt *time.Time) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE books SET
			title = ?, isbn = ?, pageCount = ?, genre = ?,
			authors_id = ?, publishers_id = ?, state = ?,
			updatedAt = ?, deletedAt = ?,
			updatedBy_users_id = ?, deletedBy_users_id = ?
		 WHERE id = ? AND updatedAt = ?`,
		b.Title, b.ISBN, b.PageCount, b.Genre,
		b.RelAuthor, b.RelPublisher, b.State,
		b.UpdatedAt, b.DeletedAt,
		b.RelUpdatedBy, b.RelDeletedBy,
		b.ID, expectedUpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("books update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.NewHTTPError(http.StatusConflict, "book was modified concurrently, please retry")
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

// TransferBooks atomically reassigns all active books from fromAuthorID to
// toAuthorID in a single UPDATE statement, stamping the audit trail.
func (r *BookRepository) TransferBooks(ctx context.Context, fromAuthorID, toAuthorID, actorUserID int64) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE books SET authors_id = ?, updatedAt = ?, updatedBy_users_id = ?
		 WHERE authors_id = ? AND state = ?`,
		toAuthorID, now, actorUserID, fromAuthorID, models.StateActive,
	)
	if err != nil {
		return fmt.Errorf("books transfer: %w", err)
	}
	return nil
}

// BulkSoftDelete soft-deletes every active book matching args and returns the
// number of rows updated. The WHERE clause is built the same way as List so
// callers can reuse the same filter syntax.
func (r *BookRepository) BulkSoftDelete(ctx context.Context, args *models.BookArgs, actorUserID int64) (int64, error) {
	where, params, err := buildBookWhere(args)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	// Build the UPDATE with the same WHERE fragment used by List.
	// We replace the default "books.state = 'active'" that buildBookWhere
	// already adds, so we only soft-delete active records.
	q := `UPDATE books SET state = ?, deletedAt = ?, updatedAt = ?,
		deletedBy_users_id = ?, updatedBy_users_id = ?
		` + strings.Replace(where, "WHERE ", "WHERE ", 1) // passthrough
	allParams := append([]any{models.StateDeleted, now, now, actorUserID, actorUserID}, params...)
	res, err := r.db.ExecContext(ctx, q, allParams...)
	if err != nil {
		return 0, fmt.Errorf("books bulk delete: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
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
	if b.RelPublisher != nil && b.RelPublisher.ID == 0 {
		b.RelPublisher = nil
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
