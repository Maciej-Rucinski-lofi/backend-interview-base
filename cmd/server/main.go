// Command server runs the library-api HTTP server.
//
// This is the wiring file: it constructs the dependency graph and starts
// Echo. The order matters and mirrors deskapi/main.go:
//
//  1. Open the database.
//  2. Build the repositories.
//  3. Build the empty service locator.
//  4. Build the services with their repos + the locator and register them.
//  5. Build the Echo server, attach middleware and routes.
//
// The "register-after-construct" dance is what lets services reference each
// other without an initialisation cycle.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	apiv1 "library-api/api/v1"
	"library-api/data/sqlite"
	"library-api/services"
)

func main() {
	dsn := os.Getenv("LIBRARY_DSN")
	if dsn == "" {
		dsn = "file:library.db?cache=shared"
	}

	db, err := sqlite.Open(dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	authorRepo := sqlite.NewAuthorRepository(db)
	bookRepo := sqlite.NewBookRepository(db)
	publisherRepo := sqlite.NewPublisherRepository(db)

	locator := services.NewLocator()
	locator.SetAuthor(services.NewAuthorService(authorRepo, locator))
	locator.SetBook(services.NewBookService(bookRepo, authorRepo, publisherRepo, locator))
	locator.SetPublisher(services.NewPublisherService(publisherRepo, locator))

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = apiv1.ErrorHandler
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	// Health probe sits outside the auth chain.
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	authed := e.Group("/v1", apiv1.AuthMiddleware())
	apiv1.Register(authed, locator)

	addr := os.Getenv("LIBRARY_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
