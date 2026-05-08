# library-api

A small REST API for a fictional library catalogue. It is intentionally
overbuilt for two resources (Author and Book) to see how you, as a candidate,
work with these **patterns**. The interview task
will ask you to add a new resource, modify an existing one, or fix a bug,
working *with* these patterns rather than around them.

## Run it

```sh
go mod tidy
go test ./...
go run ./cmd/server
```

Then in another terminal:

```sh
# Endpoints are open by default and act as user id 1; pass X-User-Id to
# act as someone else.
curl -s -H "Content-Type: application/json" \
     -d '{"author":{"name":"Ada Lovelace","bio":"first programmer"}}' \
     http://localhost:8080/v1/authors | jq

curl -s "http://localhost:8080/v1/books?include=authors" | jq

curl -s -H "X-User-Id: 42" -H "Content-Type: application/json" \
     -d '{"author":{"name":"Grace Hopper"}}' \
     http://localhost:8080/v1/authors | jq
```

The default storage is a SQLite file `library.db`. Set `LIBRARY_DSN` to
override (e.g. `file::memory:?cache=shared` to keep state in RAM).

## Layout

```
cmd/server/        — entrypoint; wires the dependency graph and starts Echo
api/v1/            — HTTP controllers; one file per resource
services/          — business logic; one file per resource
services/iservices — service interfaces and the ServiceLocator interface
data/sqlite/       — repository implementations
models/            — domain types, embedded Meta, RequestCommons, Filter
mock/              — hand-written mocks for tests
```

## The patterns you should recognise

### 1. Layered architecture

Every request flows: **Controller → Service → Repository → SQLite**.

* Controllers (`api/v1/*_controller.go`) bind the request, call the service,
  serialise the result. They never touch SQL or business rules.
* Services (`services/*_service.go`) own validation, audit-trail stamping,
  and any cross-resource rules. They depend on a `Repository` interface
  that *the service itself defines* — not the data package — so tests can
  mock the repo without importing the SQLite driver.
* Repositories (`data/sqlite/*_repo.go`) are pure CRUD plus query-shape
  translation from `Args` to SQL.

### 2. `RequestCommons` and `Args`

Every list endpoint takes a `*ResourceArgs` struct that **embeds
`models.RequestCommons`**. RequestCommons gives you, for free:

* `Page`, `PageSize`, `OrderBy`, `OrderMode`, `ID`
* `Filter` — a typed, parameterised filter list
* `Includes` — comma-separated `?include=` set
* `HardDelete` — flag for permanent vs soft delete

Resource-specific filters (e.g. `BookArgs.AuthorID`) sit alongside the
embed. The repo reads both.

To filter from inside a service, use the same builder a client uses:

```go
args := &models.BookArgs{}
args.Filter.Add("author.id", models.OpEq, authorID)
```

### 3. `FilterFieldMap`

Each model declares which API field names are allowed and how they map to
DB columns:

```go
func (Book) FilterFieldMap() map[string]string {
    return map[string]string{
        "title":     "books.title",
        "author.id": "books.authors_id",
        ...
    }
}
```

Anything not in the map is rejected by the filter parser. The same map is
used for `OrderBy` whitelisting.

### 4. `Meta` and `Relationship`

Every resource embeds `models.Meta`, which carries `CreatedAt`, `UpdatedAt`,
`DeletedAt`, the corresponding `RelCreatedBy` / `RelUpdatedBy` /
`RelDeletedBy` user pointers, and `State`.

* `Rel*` fields are persisted as foreign-key int64 columns (`createdBy_users_id`)
  but rendered in JSON as `Relationship` objects (`{"id":1,"type":"users"}`).
* Soft delete is the default. `MetaDelete` sets `State="deleted"`,
  `DeletedAt`, and `RelDeletedBy`.
* Hard delete is opt-in via `?hardDelete=true`.

### 5. `ServiceLocator`

Services don't import each other directly. They depend on
`iservices.ServiceLocator` and call peer services through it:

```go
_, err := s.svc.Author(ctx).Get(ctx, &models.AuthorArgs{...})
```

Why? Two reasons:

1. **Testability.** A unit test for `BookService` can pass a hand-rolled
   `Locator` mock that returns whatever fake `AuthorService` it likes. See
   `mock/iservices/locator_mock.go`.
2. **Construction order.** Services often have circular references in
   practice. The locator is built empty and services register themselves
   onto it after they're constructed.

In `cmd/server/main.go`:

```go
loc := services.NewLocator()
loc.SetAuthor(services.NewAuthorService(authorRepo, loc))
loc.SetBook(services.NewBookService(bookRepo, authorRepo, loc))
```

### 6. `Includes`

`?include=authors` triggers eager loading. The service collects every
referenced author id from the page, batch-fetches them, and attaches the
result under `Included.Authors` on the response body. This is the
N-+-1-avoidance pattern: one round trip for the resource, one for each
included relation type — never one round trip per row.

### 7. Sessions and audit trail

Auth middleware attaches a `*models.Session` to the request context. Every
service reads it via `models.MustGetSession(ctx)` to populate the audit
fields:

```go
session := models.MustGetSession(ctx)
b.MetaCreate(session.UserID)
```

The middleware in this project trusts an `X-User-Id` header — that's the
only thing real auth replaces.

### 8. Errors

Services return `*models.HTTPError` whenever a failure should map to a
specific status code. The single error handler in
`api/v1/router.go::ErrorHandler` translates it to a JSON response. No
controller maps errors itself.

## Testing

There are two kinds of tests:

* **Service tests** (`services/*_test.go`) — drive the service with
  hand-rolled repo + locator mocks. Fast, no DB.
* **API integration tests** (`api/v1/api_test.go`) — spin up the full real
  stack against an in-memory SQLite database and exercise every layer.

```sh
go test ./...                    # everything
go test ./services -run Create   # one service test
go test ./api/v1   -run EndToEnd # the integration test
```

## What to add (interview prompts to do)

You will find a number of tasks to complete in INTERVIEW_TASKS.md in this repo. 
