package v1

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"library-api/models"
	"library-api/services/iservices"
)

// AuthorController is the HTTP controller for /v1/authors. It is stateless
// (only holds the locator); routes are registered by calling Routes() with
// an echo.Group.
type AuthorController struct {
	svc iservices.ServiceLocator
}

// NewAuthorController is the canonical constructor.
func NewAuthorController(svc iservices.ServiceLocator) *AuthorController {
	return &AuthorController{svc: svc}
}

// Routes attaches every author route onto g. We register on a Group rather
// than on the Echo root so the caller decides the prefix and the auth chain.
func (h *AuthorController) Routes(g *echo.Group) {
	g.GET("/authors", h.list)
	g.GET("/authors/:id", h.get)
	g.POST("/authors", h.create)
	g.PATCH("/authors/:id", h.update)
	g.DELETE("/authors/:id", h.delete)
}

func (h *AuthorController) list(c echo.Context) error {
	args := &models.AuthorArgs{}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	body, err := h.svc.Author(c.Request().Context()).List(c.Request().Context(), args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, body)
}

func (h *AuthorController) get(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	args := &models.AuthorArgs{RequestCommons: models.RequestCommons{ID: id}}
	body, err := h.svc.Author(c.Request().Context()).Get(c.Request().Context(), args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, body)
}

func (h *AuthorController) create(c echo.Context) error {
	body := &models.AuthorBody{}
	if err := c.Bind(body); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	if err := h.svc.Author(c.Request().Context()).Create(c.Request().Context(), body); err != nil {
		return err
	}
	// Return the freshly-saved record so the caller sees the audit trail.
	resp, err := h.svc.Author(c.Request().Context()).Get(
		c.Request().Context(),
		&models.AuthorArgs{RequestCommons: models.RequestCommons{ID: body.Author.ID}},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *AuthorController) update(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	body := &models.AuthorBody{}
	if err := c.Bind(body); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	if body.Author == nil {
		body.Author = &models.Author{}
	}
	body.Author.ID = id
	if err := h.svc.Author(c.Request().Context()).Update(c.Request().Context(), body); err != nil {
		return err
	}
	resp, err := h.svc.Author(c.Request().Context()).Get(
		c.Request().Context(),
		&models.AuthorArgs{RequestCommons: models.RequestCommons{ID: id}},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *AuthorController) delete(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	args := &models.AuthorArgs{RequestCommons: models.RequestCommons{ID: id}}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	if err := h.svc.Author(c.Request().Context()).Delete(c.Request().Context(), args); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// pathID extracts a positive int64 :id parameter. Used by both controllers.
func pathID(c echo.Context) (int64, error) {
	raw := c.Param("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, models.NewHTTPError(http.StatusBadRequest, "id must be a positive integer")
	}
	return id, nil
}
