package repo

import (
	"context"

	"library-api/models"
)

// BookRepo is a hand-rolled mock implementing services.BookRepository.
type BookRepo struct {
	ListFn   func(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error)
	CreateFn func(ctx context.Context, b *models.Book) error
	UpdateFn func(ctx context.Context, b *models.Book) error
	DeleteFn func(ctx context.Context, id int64) error

	Calls struct {
		List, Create, Update, Delete int
	}
}

func (m *BookRepo) List(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error) {
	m.Calls.List++
	if m.ListFn == nil {
		return nil, 0, nil
	}
	return m.ListFn(ctx, args)
}

func (m *BookRepo) Create(ctx context.Context, b *models.Book) error {
	m.Calls.Create++
	if m.CreateFn == nil {
		return nil
	}
	return m.CreateFn(ctx, b)
}

func (m *BookRepo) Update(ctx context.Context, b *models.Book) error {
	m.Calls.Update++
	if m.UpdateFn == nil {
		return nil
	}
	return m.UpdateFn(ctx, b)
}

func (m *BookRepo) Delete(ctx context.Context, id int64) error {
	m.Calls.Delete++
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, id)
}
