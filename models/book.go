package models

// Book is the more interesting example: it has a foreign key (RelAuthor) so
// candidates can see how relationships work end-to-end, and a couple of
// resource-specific filter fields.
//
// As with every resource, the *DB* column for the FK is a plain int64
// (authors_id), but the *JSON* field is a Relationship so clients can use
// the standard `?include=authors` mechanism.
type Book struct {
	ID        int64  `json:"id"        db:"id"`
	Title     string `json:"title"     db:"title"`
	ISBN      string `json:"isbn"      db:"isbn"`
	PageCount int    `json:"pageCount" db:"pageCount"`
	Genre     string `json:"genre"     db:"genre"`

	// RelAuthor mirrors the deskapi convention: persisted as the FK column
	// `authors_id`, surfaced in JSON as a Relationship. See models/meta.go
	// for the same pattern on RelCreatedBy / RelUpdatedBy / RelDeletedBy.
	RelAuthor *Relationship `json:"author,omitempty" db:"authors_id"`

	// RelPublisher optionally links a book to its publisher.
	RelPublisher *Relationship `json:"publisher,omitempty" db:"publishers_id"`

	Meta
}

func (Book) TableName() string { return "books" }
func (Book) IDField() string   { return "id" }

// FilterFieldMap exposes business filters and uses qualified column names so
// joins work correctly when the repository adds them.
func (Book) FilterFieldMap() map[string]string {
	return map[string]string{
		"id":           "books.id",
		"title":        "books.title",
		"isbn":         "books.isbn",
		"genre":        "books.genre",
		"pageCount":    "books.pageCount",
		"author.id":    "books.authors_id",
		"author.name":  "authors.name",
		"publisher.id": "books.publishers_id",
		"state":        "books.state",
	}
}

// BookArgs carries a Book-specific flag (MinPageCount) on top of the
// shared filter mechanism. This shows candidates two ways to add filters:
// either let the caller use `?filter=...` directly, or — when a particular
// filter is common enough — promote it to a typed query parameter on Args.
type BookArgs struct {
	RequestCommons

	// AuthorID, if non-zero, scopes the list to one author. This is a
	// convenience filter; the same effect is achievable via
	// `?filter=[{"name":"author.id","op":"$eq","val":123}]`.
	AuthorID int64 `json:"authorId" query:"authorId"`
}

type BookBody struct {
	Book     *Book    `json:"book"`
	Included Included `json:"included"`
}

type BooksBody struct {
	Books      []*Book    `json:"books"`
	Included   Included   `json:"included"`
	Pagination Pagination `json:"pagination"`
}

// BulkDeleteBooksBody is the request body for DELETE /v1/books.
type BulkDeleteBooksBody struct {
	Filter FilterQuery `json:"filter"`
}
