package models

// Author is the simpler of our two example resources. It demonstrates the
// minimum a model needs in this codebase:
//
//   - business fields (Name, Bio)
//   - embedded Meta for the audit trail and soft-delete state
//   - a TableName() / IDField() pair so generic SQL can target it
//   - a FilterFieldMap() so the API can declare which fields are filterable
//   - an Args struct embedding RequestCommons
type Author struct {
	ID   int64  `json:"id"          db:"id"`
	Name string `json:"name"        db:"name"`
	Bio  string `json:"bio"         db:"bio"`

	Meta
}

// TableName returns the SQL table that holds this resource.
func (Author) TableName() string { return "authors" }

// IDField returns the primary key column name.
func (Author) IDField() string { return "id" }

// FilterFieldMap declares which API filter names are allowed and how they
// translate to actual SQL columns. Anything not in this map is rejected by
// the filter parser — this is the model's API contract.
func (Author) FilterFieldMap() map[string]string {
	return map[string]string{
		"id":    "authors.id",
		"name":  "authors.name",
		"state": "authors.state",
	}
}

// AuthorArgs carries every parameter a request can supply for Author
// endpoints. Embedding RequestCommons gives us paging/sorting/filtering for
// free; resource-specific flags go below.
type AuthorArgs struct {
	RequestCommons
}

// AuthorBody and AuthorsBody are the JSON envelopes for single and list
// responses respectively. We always wrap the resource(s) in a named field
// alongside an `included` block so the response shape is predictable across
// endpoints.
type AuthorBody struct {
	Author   *Author  `json:"author"`
	Included Included `json:"included"`
}

type AuthorsBody struct {
	Authors    []*Author  `json:"authors"`
	Included   Included   `json:"included"`
	Pagination Pagination `json:"pagination"`
}
