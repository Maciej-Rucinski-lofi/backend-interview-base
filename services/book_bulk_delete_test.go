package services_test

import (
	"context"
	"testing"

	mocksvc "library-api/mock/iservices"
	mockrepo "library-api/mock/repo"
	"library-api/models"
	"library-api/services"
)

func TestBookService_BulkDelete_CallsRepoWithActor(t *testing.T) {
	ctx := models.WithSession(context.Background(), &models.Session{UserID: 7})

	var capturedActorID int64
	var capturedArgs *models.BookArgs

	books := &mockrepo.BookRepo{
		BulkSoftDeleteFn: func(_ context.Context, args *models.BookArgs, actorUserID int64) (int64, error) {
			capturedArgs = args
			capturedActorID = actorUserID
			return 3, nil
		},
	}
	svc := services.NewBookService(books, &mockrepo.AuthorRepo{}, nil, &mocksvc.Locator{})

	args := &models.BookArgs{}
	args.Filter.Add("genre", models.OpEq, "fiction")

	n, err := svc.BulkDelete(ctx, args)
	if err != nil {
		t.Fatalf("bulk delete: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 deleted, got %d", n)
	}
	if capturedActorID != 7 {
		t.Fatalf("expected actor=7, got %d", capturedActorID)
	}
	if len(capturedArgs.Filter.Clauses) != 1 || capturedArgs.Filter.Clauses[0].Name != "genre" {
		t.Fatalf("expected genre filter to be forwarded, got %+v", capturedArgs.Filter.Clauses)
	}
}
