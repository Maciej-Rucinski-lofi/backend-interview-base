package services

import (
	"context"
	"net/http"
	"strings"

	"library-api/models"
	"library-api/services/iservices"
)

// PublisherRepository is the repo contract this service depends on.
type PublisherRepository interface {
	List(ctx context.Context, args *models.PublisherArgs) ([]*models.Publisher, int64, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*models.Publisher, error)
	Create(ctx context.Context, p *models.Publisher) error
	Update(ctx context.Context, p *models.Publisher) error
	Delete(ctx context.Context, id int64) error
}

// PublisherService is the concrete implementation of iservices.PublisherService.
type PublisherService struct {
	repo PublisherRepository
	svc  iservices.ServiceLocator
}

// NewPublisherService builds the service.
func NewPublisherService(repo PublisherRepository, svc iservices.ServiceLocator) *PublisherService {
	return &PublisherService{repo: repo, svc: svc}
}

func (s *PublisherService) Get(ctx context.Context, args *models.PublisherArgs) (*models.PublisherBody, error) {
	args.PageSize = 1
	publishers, _, err := s.repo.List(ctx, args)
	if err != nil {
		return nil, err
	}
	if len(publishers) == 0 {
		return nil, models.ErrNotFound
	}
	return &models.PublisherBody{Publisher: publishers[0], Included: models.Included{}}, nil
}

func (s *PublisherService) List(ctx context.Context, args *models.PublisherArgs) (*models.PublishersBody, error) {
	publishers, total, err := s.repo.List(ctx, args)
	if err != nil {
		return nil, err
	}
	return &models.PublishersBody{
		Publishers: publishers,
		Included:   models.Included{},
		Pagination: models.Pagination{Page: args.Page, PageSize: args.PageSize, Total: total},
	}, nil
}

func (s *PublisherService) Create(ctx context.Context, body *models.PublisherBody) error {
	if body == nil || body.Publisher == nil {
		return models.NewHTTPError(http.StatusBadRequest, "publisher is required")
	}
	if err := s.validate(body.Publisher); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	body.Publisher.MetaCreate(session.UserID)
	return s.repo.Create(ctx, body.Publisher)
}

func (s *PublisherService) Update(ctx context.Context, body *models.PublisherBody) error {
	if body == nil || body.Publisher == nil {
		return models.NewHTTPError(http.StatusBadRequest, "publisher is required")
	}
	existing, err := s.Get(ctx, &models.PublisherArgs{RequestCommons: models.RequestCommons{ID: body.Publisher.ID}})
	if err != nil {
		return err
	}
	existing.Publisher.Name = body.Publisher.Name
	existing.Publisher.Country = body.Publisher.Country
	if err := s.validate(existing.Publisher); err != nil {
		return err
	}
	session := models.MustGetSession(ctx)
	existing.Publisher.MetaUpdate(session.UserID)
	return s.repo.Update(ctx, existing.Publisher)
}

func (s *PublisherService) Delete(ctx context.Context, args *models.PublisherArgs) error {
	body, err := s.Get(ctx, args)
	if err != nil {
		return err
	}
	if args.HardDelete {
		return s.repo.Delete(ctx, body.Publisher.ID)
	}
	session := models.MustGetSession(ctx)
	body.Publisher.MetaDelete(session.UserID)
	return s.repo.Update(ctx, body.Publisher)
}

func (s *PublisherService) validate(p *models.Publisher) error {
	if strings.TrimSpace(p.Name) == "" {
		return models.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if len(p.Name) > 200 {
		return models.NewHTTPError(http.StatusBadRequest, "name is too long")
	}
	return nil
}

var _ iservices.PublisherService = (*PublisherService)(nil)
