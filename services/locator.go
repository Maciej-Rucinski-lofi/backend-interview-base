// Package services contains the concrete service implementations. Each
// service holds:
//
//   - a Repository interface defined by the service itself (so tests can
//     plug in a mock without dragging in the SQLite driver),
//   - a reference to the ServiceLocator so it can call peers.
package services

import (
	"context"

	"library-api/services/iservices"
)

// Locator is the concrete ServiceLocator. It wires the live services
// together at startup and is what controllers receive.
//
// Why a struct full of pointers? Two reasons:
//  1. Services are stateless; one instance per process is plenty.
//  2. Construction order is hairy when services reference each other; the
//     setter pattern (NewLocator -> SetAuthor / SetBook) avoids the cycle.
type Locator struct {
	author iservices.AuthorService
	book   iservices.BookService
}

// NewLocator returns an empty locator. Callers register concrete services
// onto it via the Set* methods. This mirrors the dependency-graph
// construction in deskapi/main.
func NewLocator() *Locator { return &Locator{} }

// SetAuthor registers an AuthorService with the locator.
func (l *Locator) SetAuthor(s iservices.AuthorService) { l.author = s }

// SetBook registers a BookService with the locator.
func (l *Locator) SetBook(s iservices.BookService) { l.book = s }

// Author satisfies iservices.ServiceLocator.
func (l *Locator) Author(_ context.Context) iservices.AuthorService { return l.author }

// Book satisfies iservices.ServiceLocator.
func (l *Locator) Book(_ context.Context) iservices.BookService { return l.book }

// Compile-time check that *Locator implements the umbrella interface.
var _ iservices.ServiceLocator = (*Locator)(nil)
