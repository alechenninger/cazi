package domain

import (
	"context"
	"errors"

	"github.com/alechenninger/cazi/pkg/cazi"
)

// ErrNotFound is returned when a widget is not found or doesn't satisfy authorization constraints.
// This maintains information hiding - callers cannot distinguish between "doesn't exist" and "not authorized".
var ErrNotFound = errors.New("widget not found")

// Note: The domain layer uses cazi.Expression directly as a standard interface,
// similar to context.Context. This treats CAZI as foundational vocabulary rather
// than an external dependency to be abstracted away.

// WidgetRepository defines operations for persisting and retrieving Widgets.
type WidgetRepository interface {
	// Save stores a widget in the repository.
	Save(ctx context.Context, widget *Widget) error

	// FindByID retrieves a widget by its ID, optionally filtered by an authorization expression.
	// If an expression is provided (Language != ""), it's evaluated against the widget data as part of the query filter.
	// Returns ErrNotFound if the widget is not found OR doesn't satisfy the expression.
	// The repository implementation decides which expression languages it supports.
	FindByID(ctx context.Context, id WidgetID, authzExpression cazi.Expression) (*Widget, error)
}
