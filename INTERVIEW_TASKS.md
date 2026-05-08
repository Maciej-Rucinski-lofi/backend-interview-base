# Interview tasks

## Task 1 — Add a `Publisher` resource end-to-end

Add a Publisher (id, name, country) with full CRUD at
`/v1/publishers`. Books should optionally reference a Publisher.

---

## Task 2 — Filter and sort books by `author.name`

Make `?filter=[{"name":"author.name","op":"$like","val":"Ada%"}]`
work on `GET /v1/books`. Same for `?orderBy=author.name`.

---

## Task 3 — Implement `POST /v1/authors/:id/transfer-books`

Move every book from author `:id` to a target author
specified in the request body, atomically. If anything fails, no books
should have moved.

---

## Task 4 — Find and fix the race in `BookService.Update`

Two concurrent `PATCH /v1/books/:id` requests can lose one
of the writes. Fix this.

---

## Task 5 — Bulk `DELETE /v1/books` with a filter body

Implement an endpoint that accepts a filter and
soft-deletes every matching book. The client expects to know how many
were deleted.
