package sqlite_test

import (
	"context"
	"net/http"
	"testing"

	"library-api/data/sqlite"
	"library-api/models"
)

// TestBookRepository_UpdateWithOptimisticLock_ConflictOnStaleSnapshot verifies
// that a second update using an outdated updatedAt is rejected with 409, so
// concurrent PATCH handlers cannot silently overwrite each other (Task 4).
func TestBookRepository_UpdateWithOptimisticLock_ConflictOnStaleSnapshot(t *testing.T) {
	db, err := sqlite.Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewBookRepository(db)
	ctx := context.Background()

	book := &models.Book{Title: "Original"}
	book.MetaCreate(1)
	if err := repo.Create(ctx, book); err != nil {
		t.Fatalf("create: %v", err)
	}

	books, _, err := repo.List(ctx, &models.BookArgs{RequestCommons: models.RequestCommons{ID: book.ID}})
	if err != nil || len(books) != 1 {
		t.Fatalf("list after create: err=%v len=%d", err, len(books))
	}
	snapshot := books[0].UpdatedAt

	book.Title = "First write"
	book.MetaUpdate(1)
	if err := repo.UpdateWithOptimisticLock(ctx, book, snapshot); err != nil {
		t.Fatalf("first update: %v", err)
	}

	book.Title = "Lost write"
	book.MetaUpdate(1)
	err = repo.UpdateWithOptimisticLock(ctx, book, snapshot)
	he, ok := models.AsHTTPError(err)
	if !ok || he.Status != http.StatusConflict {
		t.Fatalf("expected 409 on stale snapshot, got %v", err)
	}

	books, _, err = repo.List(ctx, &models.BookArgs{RequestCommons: models.RequestCommons{ID: book.ID}})
	if err != nil || len(books) != 1 {
		t.Fatalf("list after conflict: err=%v len=%d", err, len(books))
	}
	if books[0].Title != "First write" {
		t.Fatalf("expected persisted title %q, got %q", "First write", books[0].Title)
	}
}
