package services_test

import (
	"context"
	"net/http"
	"testing"

	mocksvc "library-api/mock/iservices"
	mockrepo "library-api/mock/repo"
	"library-api/models"
	"library-api/services"
)

func TestAuthorService_TransferBooks_RequiresTarget(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})
	authors := &mockrepo.AuthorRepo{}
	svc := services.NewAuthorService(authors, &mocksvc.Locator{})

	err := svc.TransferBooks(ctx, 1, 0)
	he, ok := models.AsHTTPError(err)
	if !ok || he.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 when targetAuthorId=0, got %v", err)
	}
}

func TestAuthorService_TransferBooks_SourceNotFound(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})
	authors := &mockrepo.AuthorRepo{
		ListFn: func(_ context.Context, _ *models.AuthorArgs) ([]*models.Author, int64, error) {
			return nil, 0, nil // source author not found
		},
	}
	svc := services.NewAuthorService(authors, &mocksvc.Locator{})

	err := svc.TransferBooks(ctx, 99, 1)
	he, ok := models.AsHTTPError(err)
	if !ok || he.Status != http.StatusNotFound {
		t.Fatalf("expected 404 when source author not found, got %v", err)
	}
}

func TestAuthorService_TransferBooks_TargetNotFound(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})

	callCount := 0
	authors := &mockrepo.AuthorRepo{
		ListFn: func(_ context.Context, args *models.AuthorArgs) ([]*models.Author, int64, error) {
			callCount++
			if callCount == 1 {
				// first call: source author found
				return []*models.Author{{ID: args.ID, Name: "Source"}}, 1, nil
			}
			// second call: target author not found
			return nil, 0, nil
		},
	}
	svc := services.NewAuthorService(authors, &mocksvc.Locator{})

	err := svc.TransferBooks(ctx, 1, 2)
	he, ok := models.AsHTTPError(err)
	if !ok || he.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 when target author not found, got %v", err)
	}
}

func TestAuthorService_TransferBooks_CallsBookService(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})

	authors := &mockrepo.AuthorRepo{
		ListFn: func(_ context.Context, args *models.AuthorArgs) ([]*models.Author, int64, error) {
			return []*models.Author{{ID: args.ID, Name: "Author"}}, 1, nil
		},
	}

	var capturedFrom, capturedTo int64
	bookSvc := &mocksvc.BookService{
		TransferBooksFn: func(_ context.Context, from, to int64) error {
			capturedFrom, capturedTo = from, to
			return nil
		},
	}
	loc := &mocksvc.Locator{BookSvc: bookSvc}
	svc := services.NewAuthorService(authors, loc)

	if err := svc.TransferBooks(ctx, 10, 20); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFrom != 10 || capturedTo != 20 {
		t.Fatalf("expected TransferBooks(10, 20), got (%d, %d)", capturedFrom, capturedTo)
	}
}
