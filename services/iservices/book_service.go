package iservices

import (
	"context"

	"library-api/models"
)

// BookService is the public interface for book operations.
type BookService interface {
	Get(ctx context.Context, args *models.BookArgs) (*models.BookBody, error)
	List(ctx context.Context, args *models.BookArgs) (*models.BooksBody, error)
	Create(ctx context.Context, body *models.BookBody) error
	Update(ctx context.Context, body *models.BookBody) error
	Delete(ctx context.Context, args *models.BookArgs) error

	// BulkDelete soft-deletes every book matching the filter in args and
	// returns the number of books that were deleted.
	BulkDelete(ctx context.Context, args *models.BookArgs) (int64, error)

	// TransferBooks atomically moves all active books from fromAuthorID to
	// toAuthorID. Called by AuthorService.TransferBooks via the locator.
	TransferBooks(ctx context.Context, fromAuthorID, toAuthorID int64) error
}
