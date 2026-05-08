package services

import (
	"context"
	"net/http"
	"strings"

	"library-api/models"
	"library-api/services/iservices"
)

// AuthorRepository is the repository contract this service depends on. Note
// it lives next to the service, not in the data package — the *consumer*
// owns the interface so the service can be mocked without pulling in the
// real driver.
type AuthorRepository interface {
	List(ctx context.Context, args *models.AuthorArgs) ([]*models.Author, int64, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*models.Author, error)
	Create(ctx context.Context, a *models.Author) error
	Update(ctx context.Context, a *models.Author) error
	Delete(ctx context.Context, id int64) error
}

// AuthorService is the concrete implementation. It is constructed at
// startup with a repo and the locator, then registered onto the locator.
type AuthorService struct {
	repo AuthorRepository
	svc  iservices.ServiceLocator
}

// NewAuthorService builds the service. svc is the locator; the service can
// reach peer services via svc.Book(ctx) etc.
func NewAuthorService(repo AuthorRepository, svc iservices.ServiceLocator) *AuthorService {
	return &AuthorService{repo: repo, svc: svc}
}

// Get returns one author. We forward to List with a PageSize of 1 so all the
// filter / state logic lives in exactly one place. This is the same approach
// the production codebase takes (Get is a thin wrapper over List).
func (s *AuthorService) Get(ctx context.Context, args *models.AuthorArgs) (*models.AuthorBody, error) {
	args.PageSize = 1
	authors, _, err := s.repo.List(ctx, args)
	if err != nil {
		return nil, err
	}
	if len(authors) == 0 {
		return nil, models.ErrNotFound
	}
	return &models.AuthorBody{Author: authors[0], Included: models.Included{}}, nil
}

// List returns a page of authors and the pagination metadata.
func (s *AuthorService) List(ctx context.Context, args *models.AuthorArgs) (*models.AuthorsBody, error) {
	authors, total, err := s.repo.List(ctx, args)
	if err != nil {
		return nil, err
	}
	return &models.AuthorsBody{
		Authors:    authors,
		Included:   models.Included{},
		Pagination: models.Pagination{Page: args.Page, PageSize: args.PageSize, Total: total},
	}, nil
}

// Create validates the body, stamps the audit trail, and inserts.
func (s *AuthorService) Create(ctx context.Context, body *models.AuthorBody) error {
	if body == nil || body.Author == nil {
		return models.NewHTTPError(http.StatusBadRequest, "author is required")
	}
	if err := s.validate(body.Author); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	body.Author.MetaCreate(session.UserID)
	return s.repo.Create(ctx, body.Author)
}

// Update merges the request body onto the existing record and re-validates.
// We deliberately re-fetch first so users can't touch fields the API doesn't
// expose for editing (state, audit trail).
func (s *AuthorService) Update(ctx context.Context, body *models.AuthorBody) error {
	if body == nil || body.Author == nil {
		return models.NewHTTPError(http.StatusBadRequest, "author is required")
	}
	existing, err := s.Get(ctx, &models.AuthorArgs{RequestCommons: models.RequestCommons{ID: body.Author.ID}})
	if err != nil {
		return err
	}
	existing.Author.Name = body.Author.Name
	existing.Author.Bio = body.Author.Bio
	if err := s.validate(existing.Author); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	existing.Author.MetaUpdate(session.UserID)
	return s.repo.Update(ctx, existing.Author)
}

// Delete soft-deletes by default. HardDelete=true permanently removes.
//
// Before deleting we use the locator to check that no books still point at
// this author — a real-world example of cross-service business rules
// flowing through the locator.
func (s *AuthorService) Delete(ctx context.Context, args *models.AuthorArgs) error {
	body, err := s.Get(ctx, args)
	if err != nil {
		return err
	}

	bookArgs := &models.BookArgs{AuthorID: body.Author.ID}
	books, err := s.svc.Book(ctx).List(ctx, bookArgs)
	if err != nil {
		return err
	}
	if books.Pagination.Total > 0 {
		return models.NewHTTPError(http.StatusConflict, "author has books; reassign or delete them first")
	}

	if args.HardDelete {
		return s.repo.Delete(ctx, body.Author.ID)
	}
	session := models.MustGetSession(ctx)
	body.Author.MetaDelete(session.UserID)
	return s.repo.Update(ctx, body.Author)
}

// validate is intentionally tiny. The point isn't validation rigour; it is
// to show candidates *where* validation lives — at the start of Create /
// Update, never in the controller and never in the repo.
func (s *AuthorService) validate(a *models.Author) error {
	if strings.TrimSpace(a.Name) == "" {
		return models.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if len(a.Name) > 200 {
		return models.NewHTTPError(http.StatusBadRequest, "name is too long")
	}
	return nil
}

// Compile-time interface check.
var _ iservices.AuthorService = (*AuthorService)(nil)
