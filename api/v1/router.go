// Package v1 holds the HTTP layer. Each resource has its own controller
// file; controllers are stateless and depend only on the iservices.ServiceLocator.
//
// The pattern matches deskapi: NewXxxController(svc).Routes(group) attaches
// the routes onto an echo.Group.
package v1

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"library-api/models"
	"library-api/services/iservices"
)

// Register wires every v1 controller into the given Echo group. The middleware
// stack assumed here is: requestLogger -> auth (which puts a *Session on the
// context) -> the per-controller handlers.
func Register(g *echo.Group, svc iservices.ServiceLocator) {
	NewAuthorController(svc).Routes(g)
	NewBookController(svc).Routes(g)
	NewPublisherController(svc).Routes(g)
}

// ErrorHandler is the central translator from service errors to HTTP
// responses. It is wired in main.go via e.HTTPErrorHandler. Keeping the
// translation in one place means the controllers can just `return err`.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status := http.StatusInternalServerError
	message := "internal server error"

	var he *models.HTTPError
	switch {
	case errors.As(err, &he):
		status = he.Status
		message = he.Message
	default:
		// Echo's own *echo.HTTPError still flows through here for
		// 404/405/400-binding errors. Honour them when present.
		var ehe *echo.HTTPError
		if errors.As(err, &ehe) {
			status = ehe.Code
			if msg, ok := ehe.Message.(string); ok && msg != "" {
				message = msg
			}
		}
	}
	_ = c.JSON(status, map[string]any{"error": message})
}

// defaultUserID is the user id we attach to the session when no X-User-Id
// header is provided. Setting it (rather than rejecting the request) keeps
// the API friendly to curl and tests while still demonstrating the
// session-on-context pattern that real auth uses.
const defaultUserID int64 = 1

// AuthMiddleware is a stand-in for the real authentication middleware. It
// reads X-User-Id (defaulting to 1) and attaches a *models.Session to the
// request context so services can fetch the actor with models.MustGetSession.
//
// Real deskapi auth is much more involved (token verification, installation
// scoping, permissions). For an interview base, X-User-Id is enough — the
// candidate sees how a session reaches the service layer.
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := defaultUserID
			if id := c.Request().Header.Get("X-User-Id"); id != "" {
				var parsed int64
				for _, r := range id {
					if r < '0' || r > '9' {
						return models.NewHTTPError(http.StatusUnauthorized, "X-User-Id must be numeric")
					}
					parsed = parsed*10 + int64(r-'0')
				}
				userID = parsed
			}
			ctx := models.WithSession(c.Request().Context(), &models.Session{UserID: userID})
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
