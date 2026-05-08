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

// TestBookService_Create_RejectsUnknownAuthor verifies the cross-service
// validation flow: a Book references an Author by id, and BookService.Create
// must consult the AuthorService (via the locator) before persisting.
//
// This is the canonical pattern candidates should be able to read and
// extend: hand-rolled mocks plus a hand-rolled locator mock, no codegen.
func TestBookService_Create_RejectsUnknownAuthor(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 7})

	authors := &mockrepo.AuthorRepo{}
	books := &mockrepo.BookRepo{}

	loc := &mocksvc.Locator{
		AuthorSvc: &mocksvc.AuthorService{
			GetFn: func(_ context.Context, _ *models.AuthorArgs) (*models.AuthorBody, error) {
				return nil, models.ErrNotFound
			},
		},
	}
	svc := services.NewBookService(books, authors, loc)

	body := &models.BookBody{Book: &models.Book{
		Title:     "Some Title",
		RelAuthor: &models.Relationship{ID: 999, Type: "authors"},
	}}
	err := svc.Create(ctx, body)
	if err == nil {
		t.Fatalf("expected error when author does not exist")
	}
	he, ok := models.AsHTTPError(err)
	if !ok || he.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 HTTPError, got: %v", err)
	}
	if books.Calls.Create != 0 {
		t.Fatalf("BookRepo.Create should not have been called when validation fails")
	}
}

// TestBookService_Create_StampsMeta verifies the audit trail is populated
// from the session and that the repository is invoked with the right Book.
func TestBookService_Create_StampsMeta(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 42})

	authors := &mockrepo.AuthorRepo{}
	books := &mockrepo.BookRepo{
		CreateFn: func(_ context.Context, b *models.Book) error {
			b.ID = 1
			return nil
		},
	}
	loc := &mocksvc.Locator{
		AuthorSvc: &mocksvc.AuthorService{
			GetFn: func(_ context.Context, args *models.AuthorArgs) (*models.AuthorBody, error) {
				return &models.AuthorBody{Author: &models.Author{ID: args.ID, Name: "Real Author"}}, nil
			},
		},
	}
	svc := services.NewBookService(books, authors, loc)

	body := &models.BookBody{Book: &models.Book{
		Title:     "Patterns",
		PageCount: 320,
		Genre:     "non-fiction",
		RelAuthor: &models.Relationship{ID: 5, Type: "authors"},
	}}
	if err := svc.Create(ctx, body); err != nil {
		t.Fatalf("create: %v", err)
	}
	if body.Book.RelCreatedBy.GetID() != 42 {
		t.Fatalf("expected RelCreatedBy=42, got %v", body.Book.RelCreatedBy)
	}
	if body.Book.State != models.StateActive {
		t.Fatalf("expected state=active, got %q", body.Book.State)
	}
	if books.Calls.Create != 1 {
		t.Fatalf("expected 1 call to repo.Create, got %d", books.Calls.Create)
	}
}

// TestBookService_Get_NotFound is the simplest possible service test: when
// the repo returns nothing, the service must return a 404 HTTPError.
func TestBookService_Get_NotFound(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})
	books := &mockrepo.BookRepo{
		ListFn: func(_ context.Context, _ *models.BookArgs) ([]*models.Book, int64, error) {
			return nil, 0, nil
		},
	}
	svc := services.NewBookService(books, &mockrepo.AuthorRepo{}, &mocksvc.Locator{})

	_, err := svc.Get(ctx, &models.BookArgs{RequestCommons: models.RequestCommons{ID: 7}})
	he, _ := models.AsHTTPError(err)
	if he == nil || he.Status != http.StatusNotFound {
		t.Fatalf("expected 404 HTTPError, got %v", err)
	}
}
