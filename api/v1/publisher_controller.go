package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"library-api/models"
	"library-api/services/iservices"
)

// PublisherController serves /v1/publishers.
type PublisherController struct {
	svc iservices.ServiceLocator
}

func NewPublisherController(svc iservices.ServiceLocator) *PublisherController {
	return &PublisherController{svc: svc}
}

func (h *PublisherController) Routes(g *echo.Group) {
	g.GET("/publishers", h.list)
	g.GET("/publishers/:id", h.get)
	g.POST("/publishers", h.create)
	g.PATCH("/publishers/:id", h.update)
	g.DELETE("/publishers/:id", h.delete)
}

func (h *PublisherController) list(c echo.Context) error {
	args := &models.PublisherArgs{}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	body, err := h.svc.Publisher(c.Request().Context()).List(c.Request().Context(), args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, body)
}

func (h *PublisherController) get(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	args := &models.PublisherArgs{RequestCommons: models.RequestCommons{ID: id}}
	body, err := h.svc.Publisher(c.Request().Context()).Get(c.Request().Context(), args)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, body)
}

func (h *PublisherController) create(c echo.Context) error {
	body := &models.PublisherBody{}
	if err := c.Bind(body); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	if err := h.svc.Publisher(c.Request().Context()).Create(c.Request().Context(), body); err != nil {
		return err
	}
	resp, err := h.svc.Publisher(c.Request().Context()).Get(
		c.Request().Context(),
		&models.PublisherArgs{RequestCommons: models.RequestCommons{ID: body.Publisher.ID}},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *PublisherController) update(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	body := &models.PublisherBody{}
	if err := c.Bind(body); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	if body.Publisher == nil {
		body.Publisher = &models.Publisher{}
	}
	body.Publisher.ID = id
	if err := h.svc.Publisher(c.Request().Context()).Update(c.Request().Context(), body); err != nil {
		return err
	}
	resp, err := h.svc.Publisher(c.Request().Context()).Get(
		c.Request().Context(),
		&models.PublisherArgs{RequestCommons: models.RequestCommons{ID: id}},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *PublisherController) delete(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return err
	}
	args := &models.PublisherArgs{RequestCommons: models.RequestCommons{ID: id}}
	if err := c.Bind(args); err != nil {
		return models.NewHTTPError(http.StatusBadRequest, "invalid query: "+err.Error())
	}
	if err := h.svc.Publisher(c.Request().Context()).Delete(c.Request().Context(), args); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
