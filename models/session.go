package models

import "context"

// Session represents the authenticated caller for a request. In deskapi this
// is a much richer object (installation, shard, permissions) — here we keep
// just the user id so the audit trail in Meta can be populated.
//
// The convention is exactly the same: middleware attaches a *Session to the
// request context, and any service that needs to know "who is doing this?"
// calls models.MustGetSession(ctx).
type Session struct {
	UserID int64
}

type sessionKey struct{}

// WithSession returns a context that carries the given session. Used by the
// auth middleware (and by tests).
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// GetSession returns the session attached to the context, if any.
func GetSession(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(*Session)
	return s, ok
}

// MustGetSession returns the session attached to the context or panics. Use
// this inside services and handlers where the auth middleware guarantees a
// session is present — it makes the call site short and the failure mode
// loud.
func MustGetSession(ctx context.Context) *Session {
	s, ok := GetSession(ctx)
	if !ok {
		panic("models: no session on context (missing auth middleware?)")
	}
	return s
}
