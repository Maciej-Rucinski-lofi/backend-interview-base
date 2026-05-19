package services_test

import (
	"context"
	"net/http"
	"testing"

	mocksvc "library-api/mock/iservices"
	mockrepo "library-api/mock/repo"
	"library-api/models"
	"library-api/services"
)

func TestPublisherService_Create_StampsMeta(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 5})

	publishers := &mockrepo.PublisherRepo{
		CreateFn: func(_ context.Context, p *models.Publisher) error {
			p.ID = 10
			return nil
		},
	}
	loc := &mocksvc.Locator{}
	svc := services.NewPublisherService(publishers, loc)

	body := &models.PublisherBody{Publisher: &models.Publisher{Name: "O'Reilly", Country: "US"}}
	if err := svc.Create(ctx, body); err != nil {
		t.Fatalf("create: %v", err)
	}
	if body.Publisher.ID != 10 {
		t.Fatalf("expected ID=10, got %d", body.Publisher.ID)
	}
	if body.Publisher.RelCreatedBy.GetID() != 5 {
		t.Fatalf("expected RelCreatedBy=5, got %v", body.Publisher.RelCreatedBy)
	}
	if body.Publisher.State != models.StateActive {
		t.Fatalf("expected state=active, got %q", body.Publisher.State)
	}
	if publishers.Calls.Create != 1 {
		t.Fatalf("expected 1 repo.Create call, got %d", publishers.Calls.Create)
	}
}

func TestPublisherService_Create_RequiresName(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})
	publishers := &mockrepo.PublisherRepo{}
	svc := services.NewPublisherService(publishers, &mocksvc.Locator{})

	err := svc.Create(ctx, &models.PublisherBody{Publisher: &models.Publisher{Name: ""}})
	he, ok := models.AsHTTPError(err)
	if !ok || he.Status != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %v", err)
	}
	if publishers.Calls.Create != 0 {
		t.Fatalf("repo.Create must not be called when validation fails")
	}
}

func TestPublisherService_Get_NotFound(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 1})
	publishers := &mockrepo.PublisherRepo{
		ListFn: func(_ context.Context, _ *models.PublisherArgs) ([]*models.Publisher, int64, error) {
			return nil, 0, nil
		},
	}
	svc := services.NewPublisherService(publishers, &mocksvc.Locator{})

	_, err := svc.Get(ctx, &models.PublisherArgs{RequestCommons: models.RequestCommons{ID: 99}})
	he, _ := models.AsHTTPError(err)
	if he == nil || he.Status != http.StatusNotFound {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestPublisherService_Delete_SoftDelete(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 3})

	existing := &models.Publisher{ID: 1, Name: "Penguin"}
	existing.State = models.StateActive
	publishers := &mockrepo.PublisherRepo{
		ListFn: func(_ context.Context, _ *models.PublisherArgs) ([]*models.Publisher, int64, error) {
			return []*models.Publisher{existing}, 1, nil
		},
	}
	svc := services.NewPublisherService(publishers, &mocksvc.Locator{})

	args := &models.PublisherArgs{RequestCommons: models.RequestCommons{ID: 1}}
	if err := svc.Delete(ctx, args); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// soft-delete: Update called, not Delete
	if publishers.Calls.Update != 1 {
		t.Fatalf("expected 1 repo.Update for soft delete, got %d", publishers.Calls.Update)
	}
	if publishers.Calls.Delete != 0 {
		t.Fatalf("repo.Delete must not be called for soft delete")
	}
}
