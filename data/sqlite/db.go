// Package sqlite holds the repository implementations backed by SQLite.
//
// In deskapi the equivalent package is data/mysql; the layering is the same:
//   - One Repository type per resource.
//   - Each repo accepts a `*sql.DB` (or transaction) so it can be tested
//     against an in-memory database.
//   - A repo's only job is to translate Args -> SQL -> records and back.
//     Business rules (validation, side-effects, audit fields) live in the
//     service layer.
//
// SQLite is used here purely so the candidate can `go run` and `go test`
// without setting up MariaDB. The pattern is identical to the production
// MySQL repos.
package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGo required
)

// Open creates a SQLite connection at `dsn` and applies migrations. Pass
// "file::memory:?cache=shared" for an in-memory test database.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// migrate applies the schema. There's only one version here — for a real
// project you'd reach for a migration library; for an interview base you want
// candidates to read the schema and immediately understand the columns.
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS authors (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			name               TEXT    NOT NULL,
			bio                TEXT    NOT NULL DEFAULT '',
			state              TEXT    NOT NULL DEFAULT 'active',
			createdAt          TIMESTAMP,
			updatedAt          TIMESTAMP,
			deletedAt          TIMESTAMP,
			createdBy_users_id INTEGER,
			updatedBy_users_id INTEGER,
			deletedBy_users_id INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_authors_state ON authors(state)`,
		`CREATE TABLE IF NOT EXISTS books (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			title              TEXT    NOT NULL,
			isbn               TEXT    NOT NULL DEFAULT '',
			pageCount          INTEGER NOT NULL DEFAULT 0,
			genre              TEXT    NOT NULL DEFAULT '',
			authors_id         INTEGER,
			state              TEXT    NOT NULL DEFAULT 'active',
			createdAt          TIMESTAMP,
			updatedAt          TIMESTAMP,
			deletedAt          TIMESTAMP,
			createdBy_users_id INTEGER,
			updatedBy_users_id INTEGER,
			deletedBy_users_id INTEGER,
			FOREIGN KEY (authors_id) REFERENCES authors(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_books_state    ON books(state)`,
		`CREATE INDEX IF NOT EXISTS idx_books_authors  ON books(authors_id)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("sqlite migrate: %w (stmt: %s)", err, s)
		}
	}
	return nil
}
