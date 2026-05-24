package services_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	mocksvc "library-api/mock/iservices"
	mockrepo "library-api/mock/repo"
	"library-api/models"
	"library-api/services"
)

func TestBookService_Update_UsesOptimisticLock(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 2})

	var lockCalled bool
	books := &mockrepo.BookRepo{
		ListFn: func(_ context.Context, _ *models.BookArgs) ([]*models.Book, int64, error) {
			b := &models.Book{ID: 1, Title: "Old"}
			b.State = models.StateActive
			return []*models.Book{b}, 1, nil
		},
		UpdateWithOptimisticLockFn: func(_ context.Context, _ *models.Book, _ *time.Time) error {
			lockCalled = true
			return nil
		},
	}
	svc := services.NewBookService(books, &mockrepo.AuthorRepo{}, nil, &mocksvc.Locator{})

	body := &models.BookBody{Book: &models.Book{ID: 1, Title: "New"}}
	if err := svc.Update(ctx, body); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !lockCalled {
		t.Fatalf("UpdateWithOptimisticLock must be called on Update")
	}
	if books.Calls.Update != 0 {
		t.Fatalf("plain Update must NOT be called when optimistic locking is used")
	}
}

// TestBookService_Update_ConcurrentPatches_OneWinsOneConflicts simulates two
// concurrent PATCH requests that both read the same book snapshot (same
// updatedAt). The mock repo applies optimistic locking: the first write wins,
// the second receives 409 Conflict — matching Task 4's lost-update fix.
func TestBookService_Update_ConcurrentPatches_OneWinsOneConflicts(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})

	snapshot := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	committed := false

	books := &mockrepo.BookRepo{
		ListFn: func(_ context.Context, _ *models.BookArgs) ([]*models.Book, int64, error) {
			b := &models.Book{ID: 1, Title: "Original"}
			b.State = models.StateActive
			b.UpdatedAt = &snapshot
			return []*models.Book{b}, 1, nil
		},
		UpdateWithOptimisticLockFn: func(_ context.Context, _ *models.Book, expected *time.Time) error {
			mu.Lock()
			defer mu.Unlock()
			if expected == nil || !expected.Equal(snapshot) {
				return models.NewHTTPError(http.StatusConflict, "book was modified concurrently, please retry")
			}
			if committed {
				return models.NewHTTPError(http.StatusConflict, "book was modified concurrently, please retry")
			}
			committed = true
			return nil
		},
	}
	svc := services.NewBookService(books, &mockrepo.AuthorRepo{}, nil, &mocksvc.Locator{})

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	errs := make([]error, 2)
	bodies := []*models.BookBody{
		{Book: &models.Book{ID: 1, Title: "Writer A"}},
		{Book: &models.Book{ID: 1, Title: "Writer B"}},
	}

	for i := range bodies {
		go func(idx int) {
			defer wg.Done()
			<-start
			errs[idx] = svc.Update(ctx, bodies[idx])
		}(i)
	}
	close(start)
	wg.Wait()

	var successes, conflicts int
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		he, ok := models.AsHTTPError(err)
		if ok && he.Status == http.StatusConflict {
			conflicts++
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected 1 success and 1 conflict, got successes=%d conflicts=%d (errs=%v)", successes, conflicts, errs)
	}
	if books.Calls.UpdateWithOptimisticLock != 2 {
		t.Fatalf("expected 2 optimistic-lock updates, got %d", books.Calls.UpdateWithOptimisticLock)
	}
	if books.Calls.Update != 0 {
		t.Fatalf("plain Update must not be used")
	}
}
