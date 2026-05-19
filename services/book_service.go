package services

import (
	"context"
	"net/http"
	"strings"
	"time"

	"library-api/models"
	"library-api/services/iservices"
)

// BookRepository is the repo contract for books. As with AuthorRepository,
// it lives in the consumer package so tests can substitute a mock.
type BookRepository interface {
	List(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error)
	Create(ctx context.Context, b *models.Book) error
	Update(ctx context.Context, b *models.Book) error
	// UpdateWithOptimisticLock updates the book only if its updatedAt matches
	// expectedUpdatedAt, returning 409 if another writer already modified it.
	UpdateWithOptimisticLock(ctx context.Context, b *models.Book, expectedUpdatedAt *time.Time) error
	Delete(ctx context.Context, id int64) error
	// TransferBooks atomically reassigns all active books from one author to another.
	TransferBooks(ctx context.Context, fromAuthorID, toAuthorID, actorUserID int64) error
	// BulkSoftDelete soft-deletes every book matching args and returns the count.
	BulkSoftDelete(ctx context.Context, args *models.BookArgs, actorUserID int64) (int64, error)
}

// bookPublisherRepository is the minimal publisher repo surface BookService needs
// to resolve ?include=publishers without depending on the full PublisherRepository.
type bookPublisherRepository interface {
	ListByIDs(ctx context.Context, ids []int64) ([]*models.Publisher, error)
}

// BookService implements iservices.BookService.
type BookService struct {
	repo           BookRepository
	authorsRepo    AuthorRepository
	publishersRepo bookPublisherRepository
	svc            iservices.ServiceLocator
}

// NewBookService wires the dependencies. authorsRepo is passed directly so
// the include-resolution code can fetch authors without going through the
// locator and re-creating an AuthorService for what is effectively a
// read-only lookup. publishersRepo may be nil; publisher includes are skipped.
func NewBookService(repo BookRepository, authorsRepo AuthorRepository, publishersRepo bookPublisherRepository, svc iservices.ServiceLocator) *BookService {
	return &BookService{repo: repo, authorsRepo: authorsRepo, publishersRepo: publishersRepo, svc: svc}
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
	// Capture the pre-update timestamp for optimistic locking. Two concurrent
	// PATCH requests that both read the same snapshot will race at the UPDATE:
	// only the first one wins; the second sees 0 rows affected and gets a 409.
	prevUpdatedAt := existing.Book.UpdatedAt

	existing.Book.Title = body.Book.Title
	existing.Book.ISBN = body.Book.ISBN
	existing.Book.PageCount = body.Book.PageCount
	existing.Book.Genre = body.Book.Genre
	if body.Book.RelAuthor != nil {
		existing.Book.RelAuthor = body.Book.RelAuthor
	}
	if body.Book.RelPublisher != nil {
		existing.Book.RelPublisher = body.Book.RelPublisher
	}
	if err := s.validate(ctx, existing.Book); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	existing.Book.MetaUpdate(session.UserID)
	return s.repo.UpdateWithOptimisticLock(ctx, existing.Book, prevUpdatedAt)
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

// TransferBooks delegates to the repo for an atomic single-UPDATE transfer.
// Both authors are validated before this is called (by AuthorService).
func (s *BookService) TransferBooks(ctx context.Context, fromAuthorID, toAuthorID int64) error {
	session := models.MustGetSession(ctx)
	return s.repo.TransferBooks(ctx, fromAuthorID, toAuthorID, session.UserID)
}

// BulkDelete soft-deletes every active book matching args.Filter and returns
// the number deleted. The caller sets args.Filter programmatically or via the
// request body.
func (s *BookService) BulkDelete(ctx context.Context, args *models.BookArgs) (int64, error) {
	session := models.MustGetSession(ctx)
	return s.repo.BulkSoftDelete(ctx, args, session.UserID)
}

// includes resolves ?include=authors and ?include=publishers directives by
// batch-fetching the referenced records — one round trip per relation type.
func (s *BookService) includes(ctx context.Context, args *models.BookArgs, books []*models.Book) (models.Included, error) {
	included := models.Included{}
	if len(books) == 0 {
		return included, nil
	}

	if args.Includes.IsOn(models.IncludeAuthors) {
		ids := uniqueIDs(books, func(b *models.Book) int64 { return b.RelAuthor.GetID() })
		if len(ids) > 0 {
			authors, err := s.authorsRepo.ListByIDs(ctx, ids)
			if err != nil {
				return included, err
			}
			included.Authors = authors
		}
	}

	if args.Includes.IsOn(models.IncludePublishers) && s.publishersRepo != nil {
		ids := uniqueIDs(books, func(b *models.Book) int64 { return b.RelPublisher.GetID() })
		if len(ids) > 0 {
			publishers, err := s.publishersRepo.ListByIDs(ctx, ids)
			if err != nil {
				return included, err
			}
			included.Publishers = publishers
		}
	}

	return included, nil
}

func uniqueIDs[T any](items []T, id func(T) int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(items))
	for _, item := range items {
		if v := id(item); v != 0 {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				out = append(out, v)
			}
		}
	}
	return out
}

// validate runs the Book business rules. Note we use the locator to verify
// that the referenced author and publisher exist.
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
	if b.RelPublisher != nil && b.RelPublisher.ID != 0 {
		_, err := s.svc.Publisher(ctx).Get(ctx, &models.PublisherArgs{RequestCommons: models.RequestCommons{ID: b.RelPublisher.ID}})
		if err != nil {
			if he, ok := models.AsHTTPError(err); ok && he.Status == http.StatusNotFound {
				return models.NewHTTPError(http.StatusBadRequest, "publisher does not exist")
			}
			return err
		}
		b.RelPublisher.Type = "publishers"
	}
	return nil
}

var _ iservices.BookService = (*BookService)(nil)
