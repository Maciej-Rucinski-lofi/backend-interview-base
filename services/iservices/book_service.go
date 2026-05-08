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
}
