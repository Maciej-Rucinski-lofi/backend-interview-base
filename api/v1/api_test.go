package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	apiv1 "library-api/api/v1"
	"library-api/data/sqlite"
	"library-api/models"
	"library-api/services"
)

// newTestServer builds the full real stack against an in-memory SQLite
// database and returns an Echo server ready to receive requests via
// httptest. This is the deskapi-style "integration test" that exercises
// every layer end-to-end.
func newTestServer(t *testing.T) *echo.Echo {
	t.Helper()
	db, err := sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	authorRepo := sqlite.NewAuthorRepository(db)
	bookRepo := sqlite.NewBookRepository(db)

	loc := services.NewLocator()
	loc.SetAuthor(services.NewAuthorService(authorRepo, loc))
	loc.SetBook(services.NewBookService(bookRepo, authorRepo, loc))

	e := echo.New()
	e.HTTPErrorHandler = apiv1.ErrorHandler
	authed := e.Group("/v1", apiv1.AuthMiddleware())
	apiv1.Register(authed, loc)
	return e
}

// do runs an HTTP request against e and returns the recorder. Every request
// carries an X-User-Id so the auth middleware lets it through.
func do(t *testing.T, e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	req.Header.Set("X-User-Id", "1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestEndToEnd walks the canonical happy path:
//   - create an author
//   - create a book that references the author
//   - list books with `?include=authors` and assert the author is inlined
//   - delete the book, then the author
//
// If a candidate breaks any layer this test will catch it.
func TestEndToEnd(t *testing.T) {
	e := newTestServer(t)

	rec := do(t, e, http.MethodPost, "/v1/authors",
		`{"author":{"name":"Ada Lovelace","bio":"first programmer"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create author: status %d body=%s", rec.Code, rec.Body.String())
	}
	var ab models.AuthorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &ab); err != nil {
		t.Fatalf("unmarshal author: %v", err)
	}
	if ab.Author.ID == 0 {
		t.Fatalf("expected author id, got 0")
	}
	if ab.Author.RelCreatedBy.GetID() != 1 {
		t.Fatalf("expected RelCreatedBy=1, got %v", ab.Author.RelCreatedBy)
	}

	bookJSON := `{"book":{"title":"Notes","pageCount":42,"genre":"essay","author":` +
		jsonInt(ab.Author.ID) + `}}`
	rec = do(t, e, http.MethodPost, "/v1/books", bookJSON)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create book: status %d body=%s", rec.Code, rec.Body.String())
	}
	var bb models.BookBody
	if err := json.Unmarshal(rec.Body.Bytes(), &bb); err != nil {
		t.Fatalf("unmarshal book: %v", err)
	}
	if bb.Book.ID == 0 {
		t.Fatalf("expected book id, got 0")
	}

	rec = do(t, e, http.MethodGet, "/v1/books?include=authors", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list books: status %d body=%s", rec.Code, rec.Body.String())
	}
	var bs models.BooksBody
	if err := json.Unmarshal(rec.Body.Bytes(), &bs); err != nil {
		t.Fatalf("unmarshal books: %v", err)
	}
	if len(bs.Books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(bs.Books))
	}
	if len(bs.Included.Authors) != 1 || bs.Included.Authors[0].ID != ab.Author.ID {
		t.Fatalf("expected included author %d, got %+v", ab.Author.ID, bs.Included.Authors)
	}

	// Trying to delete the author while it still has a book must fail.
	rec = do(t, e, http.MethodDelete, "/v1/authors/"+jsonInt(ab.Author.ID), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 deleting author with books, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Delete the book first.
	rec = do(t, e, http.MethodDelete, "/v1/books/"+jsonInt(bb.Book.ID), "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete book: %d body=%s", rec.Code, rec.Body.String())
	}

	// Now the author can go.
	rec = do(t, e, http.MethodDelete, "/v1/authors/"+jsonInt(ab.Author.ID), "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete author: %d body=%s", rec.Code, rec.Body.String())
	}

	// And the soft-deleted records should be hidden from default lists.
	rec = do(t, e, http.MethodGet, "/v1/authors", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list authors: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Fatalf("expected total:0 after delete, got %s", rec.Body.String())
	}
}

// jsonInt formats an int64 as a JSON number string. (httptest doesn't have a
// helper for this and inlining a fmt.Sprintf would obscure the test bodies.)
func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// TestFilter exercises the ?filter=...&include=... binding path end-to-end.
// We seed two books in different genres and verify the filter clause
// returns only the matching one. This is the canonical "API field name ->
// DB column via FilterFieldMap" round-trip.
func TestFilter(t *testing.T) {
	e := newTestServer(t)

	// Set up an author so the books have somewhere to point.
	rec := do(t, e, http.MethodPost, "/v1/authors", `{"author":{"name":"A"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed author: %d body=%s", rec.Code, rec.Body.String())
	}
	var ab models.AuthorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &ab)

	for _, body := range []string{
		`{"book":{"title":"Hello","genre":"fiction","author":` + jsonInt(ab.Author.ID) + `}}`,
		`{"book":{"title":"World","genre":"non-fiction","author":` + jsonInt(ab.Author.ID) + `}}`,
	} {
		rec = do(t, e, http.MethodPost, "/v1/books", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed book: %d body=%s", rec.Code, rec.Body.String())
		}
	}

	// `?filter=[{"name":"genre","op":"$eq","val":"fiction"}]`, URL-encoded.
	q := `/v1/books?filter=` + url.QueryEscape(`[{"name":"genre","op":"$eq","val":"fiction"}]`)
	rec = do(t, e, http.MethodGet, q, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("filter list: %d body=%s", rec.Code, rec.Body.String())
	}
	var bs models.BooksBody
	if err := json.Unmarshal(rec.Body.Bytes(), &bs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(bs.Books) != 1 || bs.Books[0].Title != "Hello" {
		t.Fatalf("expected only the fiction book, got %+v", bs.Books)
	}
}
