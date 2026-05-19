package services_test

import (
	"context"
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
