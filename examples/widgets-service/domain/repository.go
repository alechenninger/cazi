package domain

import (
	"context"

	"github.com/alechenninger/cazi/pkg/cazi"
)

// Note: The domain layer uses cazi.Expression directly as a standard interface,
// similar to context.Context. This treats CAZI as foundational vocabulary rather
// than an external dependency to be abstracted away.

// WidgetRepository defines operations for persisting and retrieving Widgets.
type WidgetRepository interface {
	// Save stores a widget in the repository.
	Save(ctx context.Context, widget *Widget) error

	// FindByID retrieves a widget by its ID, optionally filtered by an authorization expression.
	// If an expression is provided (Language != ""), it's evaluated against the widget data as part of the query filter.
	// Returns an error if the widget is not found OR doesn't satisfy the expression.
	// The repository implementation decides which expression languages it supports.
	FindByID(ctx context.Context, id WidgetID, authzExpression cazi.Expression) (*Widget, error)
}
