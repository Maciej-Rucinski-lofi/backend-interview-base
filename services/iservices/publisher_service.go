package iservices

import (
	"context"

	"library-api/models"
)

// PublisherService is the public surface for publisher operations.
type PublisherService interface {
	Get(ctx context.Context, args *models.PublisherArgs) (*models.PublisherBody, error)
	List(ctx context.Context, args *models.PublisherArgs) (*models.PublishersBody, error)
	Create(ctx context.Context, body *models.PublisherBody) error
	Update(ctx context.Context, body *models.PublisherBody) error
	Delete(ctx context.Context, args *models.PublisherArgs) error
}
