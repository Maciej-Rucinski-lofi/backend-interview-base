// Package mockiservices holds hand-written mocks for the service
// interfaces. (The directory is `mock/iservices` to mirror the real
// `services/iservices` location, but the package is named differently to
// avoid a name collision with the import below.)
//
// The Locator mock here is enough for unit tests that need to plug a fake
// peer service into the system under test (e.g. when testing AuthorService
// you might pass a mock BookService so the cross-service "books still
// reference this author" check is observable).
package mockiservices

import (
	"context"

	"library-api/models"
	"library-api/services/iservices"
)

// AuthorService is a function-field mock implementing iservices.AuthorService.
type AuthorService struct {
	GetFn    func(ctx context.Context, args *models.AuthorArgs) (*models.AuthorBody, error)
	ListFn   func(ctx context.Context, args *models.AuthorArgs) (*models.AuthorsBody, error)
	CreateFn func(ctx context.Context, body *models.AuthorBody) error
	UpdateFn func(ctx context.Context, body *models.AuthorBody) error
	DeleteFn func(ctx context.Context, args *models.AuthorArgs) error
}

func (m *AuthorService) Get(ctx context.Context, args *models.AuthorArgs) (*models.AuthorBody, error) {
	if m.GetFn == nil {
		return &models.AuthorBody{Author: &models.Author{ID: args.ID, Name: "stub"}}, nil
	}
	return m.GetFn(ctx, args)
}
func (m *AuthorService) List(ctx context.Context, args *models.AuthorArgs) (*models.AuthorsBody, error) {
	if m.ListFn == nil {
		return &models.AuthorsBody{}, nil
	}
	return m.ListFn(ctx, args)
}
func (m *AuthorService) Create(ctx context.Context, body *models.AuthorBody) error {
	if m.CreateFn == nil {
		return nil
	}
	return m.CreateFn(ctx, body)
}
func (m *AuthorService) Update(ctx context.Context, body *models.AuthorBody) error {
	if m.UpdateFn == nil {
		return nil
	}
	return m.UpdateFn(ctx, body)
}
func (m *AuthorService) Delete(ctx context.Context, args *models.AuthorArgs) error {
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, args)
}

// BookService mirror of the above for books.
type BookService struct {
	GetFn    func(ctx context.Context, args *models.BookArgs) (*models.BookBody, error)
	ListFn   func(ctx context.Context, args *models.BookArgs) (*models.BooksBody, error)
	CreateFn func(ctx context.Context, body *models.BookBody) error
	UpdateFn func(ctx context.Context, body *models.BookBody) error
	DeleteFn func(ctx context.Context, args *models.BookArgs) error
}

func (m *BookService) Get(ctx context.Context, args *models.BookArgs) (*models.BookBody, error) {
	if m.GetFn == nil {
		return &models.BookBody{Book: &models.Book{ID: args.ID, Title: "stub"}}, nil
	}
	return m.GetFn(ctx, args)
}
func (m *BookService) List(ctx context.Context, args *models.BookArgs) (*models.BooksBody, error) {
	if m.ListFn == nil {
		return &models.BooksBody{}, nil
	}
	return m.ListFn(ctx, args)
}
func (m *BookService) Create(ctx context.Context, body *models.BookBody) error {
	if m.CreateFn == nil {
		return nil
	}
	return m.CreateFn(ctx, body)
}
func (m *BookService) Update(ctx context.Context, body *models.BookBody) error {
	if m.UpdateFn == nil {
		return nil
	}
	return m.UpdateFn(ctx, body)
}
func (m *BookService) Delete(ctx context.Context, args *models.BookArgs) error {
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, args)
}

// Locator is a hand-written ServiceLocator mock. Tests instantiate it,
// assign whichever AuthorSvc/BookSvc behaviour they need, and pass it to
// the system under test.
type Locator struct {
	AuthorSvc iservices.AuthorService
	BookSvc   iservices.BookService
}

func (m *Locator) Author(_ context.Context) iservices.AuthorService {
	if m.AuthorSvc == nil {
		m.AuthorSvc = &AuthorService{}
	}
	return m.AuthorSvc
}
func (m *Locator) Book(_ context.Context) iservices.BookService {
	if m.BookSvc == nil {
		m.BookSvc = &BookService{}
	}
	return m.BookSvc
}

var _ iservices.ServiceLocator = (*Locator)(nil)
