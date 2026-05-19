package repo

import (
	"context"
	"time"

	"library-api/models"
)

// BookRepo is a hand-rolled mock implementing services.BookRepository.
type BookRepo struct {
	ListFn                  func(ctx context.Context, args *models.BookArgs) ([]*models.Book, int64, error)
	CreateFn                func(ctx context.Context, b *models.Book) error
	UpdateFn                func(ctx context.Context, b *models.Book) error
	UpdateWithOptimisticLockFn func(ctx context.Context, b *models.Book, expectedUpdatedAt *time.Time) error
	DeleteFn                func(ctx context.Context, id int64) error
	TransferBooksFn         func(ctx context.Context, fromAuthorID, toAuthorID, actorUserID int64) error
	BulkSoftDeleteFn        func(ctx context.Context, args *models.BookArgs, actorUserID int64) (int64, error)

	Calls struct {
		List, Create, Update, UpdateWithOptimisticLock, Delete, TransferBooks, BulkSoftDelete int
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

func (m *BookRepo) UpdateWithOptimisticLock(ctx context.Context, b *models.Book, expectedUpdatedAt *time.Time) error {
	m.Calls.UpdateWithOptimisticLock++
	if m.UpdateWithOptimisticLockFn == nil {
		return nil
	}
	return m.UpdateWithOptimisticLockFn(ctx, b, expectedUpdatedAt)
}

func (m *BookRepo) Delete(ctx context.Context, id int64) error {
	m.Calls.Delete++
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, id)
}

func (m *BookRepo) TransferBooks(ctx context.Context, fromAuthorID, toAuthorID, actorUserID int64) error {
	m.Calls.TransferBooks++
	if m.TransferBooksFn == nil {
		return nil
	}
	return m.TransferBooksFn(ctx, fromAuthorID, toAuthorID, actorUserID)
}

func (m *BookRepo) BulkSoftDelete(ctx context.Context, args *models.BookArgs, actorUserID int64) (int64, error) {
	m.Calls.BulkSoftDelete++
	if m.BulkSoftDeleteFn == nil {
		return 0, nil
	}
	return m.BulkSoftDeleteFn(ctx, args, actorUserID)
}
