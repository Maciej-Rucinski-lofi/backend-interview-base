package services

import (
	"context"
	"net/http"
	"strings"

	"library-api/models"
	"library-api/services/iservices"
)

// BookRepository is the repo contract for books. As with AuthorRepository,
// it lives in the consumer package so tests can substitute a mock.
type BookRepository interface {
	List(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error)
	Create(ctx context.Context, b *models.Book) error
	Update(ctx context.Context, b *models.Book) error
	Delete(ctx context.Context, id int64) error
}

// BookService implements iservices.BookService.
type BookService struct {
	repo        BookRepository
	authorsRepo AuthorRepository
	svc         iservices.ServiceLocator
}

// NewBookService wires the dependencies. authorsRepo is passed directly so
// the include-resolution code can fetch authors without going through the
// locator and re-creating an AuthorService for what is effectively a
// read-only lookup.
func NewBookService(repo BookRepository, authorsRepo AuthorRepository, svc iservices.ServiceLocator) *BookService {
	return &BookService{repo: repo, authorsRepo: authorsRepo, svc: svc}
}

func (s *BookService) Get(ctx context.Context, args *models.BookArgs) (*models.BookBody, error) {
	args.PageSize = 1
	books, _, err := s.repo.List(ctx, args)
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return nil, models.ErrNotFound
	}
	included, err := s.includes(ctx, args, books)
	if err != nil {
		return nil, err
	}
	return &models.BookBody{Book: books[0], Included: included}, nil
}

func (s *BookService) List(ctx context.Context, args *models.BookArgs) (*models.BooksBody, error) {
	books, total, err := s.repo.List(ctx, args)
	if err != nil {
		return nil, err
	}
	included, err := s.includes(ctx, args, books)
	if err != nil {
		return nil, err
	}
	return &models.BooksBody{
		Books:      books,
		Included:   included,
		Pagination: models.Pagination{Page: args.Page, PageSize: args.PageSize, Total: total},
	}, nil
}

func (s *BookService) Create(ctx context.Context, body *models.BookBody) error {
	if body == nil || body.Book == nil {
		return models.NewHTTPError(http.StatusBadRequest, "book is required")
	}
	if err := s.validate(ctx, body.Book); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	body.Book.MetaCreate(session.UserID)
	return s.repo.Create(ctx, body.Book)
}

func (s *BookService) Update(ctx context.Context, body *models.BookBody) error {
	if body == nil || body.Book == nil {
		return models.NewHTTPError(http.StatusBadRequest, "book is required")
	}
	existing, err := s.Get(ctx, &models.BookArgs{RequestCommons: models.RequestCommons{ID: body.Book.ID}})
	if err != nil {
		return err
	}
	existing.Book.Title = body.Book.Title
	existing.Book.ISBN = body.Book.ISBN
	existing.Book.PageCount = body.Book.PageCount
	existing.Book.Genre = body.Book.Genre
	if body.Book.RelAuthor != nil {
		existing.Book.RelAuthor = body.Book.RelAuthor
	}
	if err := s.validate(ctx, existing.Book); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	existing.Book.MetaUpdate(session.UserID)
	return s.repo.Update(ctx, existing.Book)
}

func (s *BookService) Delete(ctx context.Context, args *models.BookArgs) error {
	body, err := s.Get(ctx, args)
	if err != nil {
		return err
	}
	if args.HardDelete {
		return s.repo.Delete(ctx, body.Book.ID)
	}
	session := models.MustGetSession(ctx)
	body.Book.MetaDelete(session.UserID)
	return s.repo.Update(ctx, body.Book)
}

// includes resolves the `?include=authors` directive by fetching the
// referenced authors in one query. This is the canonical N+1 avoidance
// pattern: collect all the IDs from the page, batch-fetch them, attach to
// the response body.
func (s *BookService) includes(ctx context.Context, args *models.BookArgs, books []*models.Book) (models.Included, error) {
	included := models.Included{}
	if !args.Includes.IsOn(models.IncludeAuthors) || len(books) == 0 {
		return included, nil
	}
	ids := make([]int64, 0, len(books))
	seen := map[int64]struct{}{}
	for _, b := range books {
		if id := b.RelAuthor.GetID(); id != 0 {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	authors, err := s.authorsRepo.ListByIDs(ctx, ids)
	if err != nil {
		return included, err
	}
	included.Authors = authors
	return included, nil
}

// validate runs the Book business rules. Note we use the locator to verify
// that the referenced author exists — services routinely talk to peer
// services through the locator rather than reaching into another repo.
func (s *BookService) validate(ctx context.Context, b *models.Book) error {
	if strings.TrimSpace(b.Title) == "" {
		return models.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if b.PageCount < 0 {
		return models.NewHTTPError(http.StatusBadRequest, "pageCount cannot be negative")
	}
	if b.RelAuthor != nil && b.RelAuthor.ID != 0 {
		_, err := s.svc.Author(ctx).Get(ctx, &models.AuthorArgs{RequestCommons: models.RequestCommons{ID: b.RelAuthor.ID}})
		if err != nil {
			if he, ok := models.AsHTTPError(err); ok && he.Status == http.StatusNotFound {
				return models.NewHTTPError(http.StatusBadRequest, "author does not exist")
			}
			return err
		}
		b.RelAuthor.Type = "authors"
	}
	return nil
}

var _ iservices.BookService = (*BookService)(nil)
