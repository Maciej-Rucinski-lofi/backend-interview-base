package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"library-api/models"
	"library-api/services/iservices"
)

// BookController serves /v1/books.
type BookController struct {
	svc iservices.ServiceLocator
}

func NewBookController(svc iservices.ServiceLocator) *BookController {
	return &BookController{svc: svc}
}

func (h *BookController) Routes(g *echo.Group) {
	g.GET("/books", h.list)
	g.GET("/books/:id", h.get)
	g.POST("/books", h.create)
	g.PATCH("/books/:id", h.update)
	g.DELETE("/books", h.bulkDelete)
	g.DELETE("/books/:id", h.delete)
}

func (h *BookController) list(c echo.Context) error {
	args := &models.BookArgs{}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	body, err := h.svc.Book(c.Request().Context()).List(c.Request().Context(), args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, body)
}

func (h *BookController) get(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	args := &models.BookArgs{RequestCommons: models.RequestCommons{ID: id}}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	args.ID = id
	body, err := h.svc.Book(c.Request().Context()).Get(c.Request().Context(), args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, body)
}

func (h *BookController) create(c echo.Context) error {
	body := &models.BookBody{}
	if err := c.Bind(body); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	if err := h.svc.Book(c.Request().Context()).Create(c.Request().Context(), body); err != nil {
		return err
	}
	resp, err := h.svc.Book(c.Request().Context()).Get(
		c.Request().Context(),
		&models.BookArgs{RequestCommons: models.RequestCommons{ID: body.Book.ID, Includes: models.NewIncluder(models.IncludeAuthors)}},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *BookController) update(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	body := &models.BookBody{}
	if err := c.Bind(body); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	if body.Book == nil {
		body.Book = &models.Book{}
	}
	body.Book.ID = id
	if err := h.svc.Book(c.Request().Context()).Update(c.Request().Context(), body); err != nil {
		return err
	}
	resp, err := h.svc.Book(c.Request().Context()).Get(
		c.Request().Context(),
		&models.BookArgs{RequestCommons: models.RequestCommons{ID: id, Includes: models.NewIncluder(models.IncludeAuthors)}},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *BookController) bulkDelete(c echo.Context) error {
	req := &models.BulkDeleteBooksBody{}
	if err := c.Bind(req); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	args := &models.BookArgs{}
	args.Filter = req.Filter
	ctx := c.Request().Context()
	n, err := h.svc.Book(ctx).BulkDelete(ctx, args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]int64{"deleted": n})
}

func (h *BookController) delete(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	args := &models.BookArgs{RequestCommons: models.RequestCommons{ID: id}}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	args.ID = id
	if err := h.svc.Book(c.Request().Context()).Delete(c.Request().Context(), args); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
