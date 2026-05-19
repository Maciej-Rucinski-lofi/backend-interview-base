package repo

import (
	"context"

	"library-api/models"
)

// PublisherRepo is a hand-rolled mock implementing services.PublisherRepository.
type PublisherRepo struct {
	ListFn      func(ctx context.Context, args *models.PublisherArgs) ([]*models.Publisher, int64, error)
	ListByIDsFn func(ctx context.Context, ids []int64) ([]*models.Publisher, error)
	CreateFn    func(ctx context.Context, p *models.Publisher) error
	UpdateFn    func(ctx context.Context, p *models.Publisher) error
	DeleteFn    func(ctx context.Context, id int64) error

	Calls struct {
		List, ListByIDs, Create, Update, Delete int
	}
}

func (m *PublisherRepo) List(ctx context.Context, args *models.PublisherArgs) ([]*models.Publisher, int64, error) {
	m.Calls.List++
	if m.ListFn == nil {
		return nil, 0, nil
	}
	return m.ListFn(ctx, args)
}

func (m *PublisherRepo) ListByIDs(ctx context.Context, ids []int64) ([]*models.Publisher, error) {
	m.Calls.ListByIDs++
	if m.ListByIDsFn == nil {
		return nil, nil
	}
	return m.ListByIDsFn(ctx, ids)
}

func (m *PublisherRepo) Create(ctx context.Context, p *models.Publisher) error {
	m.Calls.Create++
	if m.CreateFn == nil {
		return nil
	}
	return m.CreateFn(ctx, p)
}

func (m *PublisherRepo) Update(ctx context.Context, p *models.Publisher) error {
	m.Calls.Update++
	if m.UpdateFn == nil {
		return nil
	}
	return m.UpdateFn(ctx, p)
}

func (m *PublisherRepo) Delete(ctx context.Context, id int64) error {
	m.Calls.Delete++
	if m.DeleteFn == nil {
		return nil
	}
	return m.DeleteFn(ctx, id)
}
