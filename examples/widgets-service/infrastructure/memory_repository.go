package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/alechenninger/cazi/pkg/cazi"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"widgets-service/domain"
)

// InMemoryWidgetRepository is an in-memory implementation of domain.WidgetRepository.
type InMemoryWidgetRepository struct {
	mu      sync.RWMutex
	widgets map[domain.WidgetID]domain.WidgetData
	celEnv  *cel.Env // CEL environment for evaluating expressions
}

// NewInMemoryWidgetRepository creates a new in-memory repository.
func NewInMemoryWidgetRepository() *InMemoryWidgetRepository {
	// Create CEL environment with widget object type
	env, err := cel.NewEnv(
		cel.Variable("widget", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create CEL environment: %v", err))
	}

	return &InMemoryWidgetRepository{
		widgets: make(map[domain.WidgetID]domain.WidgetData),
		celEnv:  env,
	}
}

// Save stores a widget in memory.
func (r *InMemoryWidgetRepository) Save(ctx context.Context, widget *domain.Widget) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data := widget.Serialize()
	r.widgets[widget.ID()] = data
	return nil
}

// FindByID retrieves a widget by ID from memory, optionally applying an authorization expression.
// The expression is evaluated as a filter - if it doesn't match, returns ErrNotFound (security feature).
func (r *InMemoryWidgetRepository) FindByID(ctx context.Context, id domain.WidgetID, authzExpression cazi.Expression) (*domain.Widget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.widgets[id]
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Apply authorization expression as part of the query filter
	if authzExpression.Language != "" {
		matches, err := r.evaluateExpression(data, authzExpression)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate authorization expression: %w", err)
		}
		if !matches {
			// Return same error as "not found" - security feature
			// This prevents leaking information about whether the resource exists
			return nil, domain.ErrNotFound
		}
	}

	return domain.DeserializeWidget(data), nil
}

// FindAll retrieves all widgets from memory, optionally applying an authorization expression.
// The expression is evaluated as a filter - only matching widgets are returned.
func (r *InMemoryWidgetRepository) FindAll(ctx context.Context, authzExpression cazi.Expression) ([]*domain.Widget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*domain.Widget

	for _, data := range r.widgets {
		// Apply authorization expression as part of the query filter
		if authzExpression.Language != "" {
			matches, err := r.evaluateExpression(data, authzExpression)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate authorization expression: %w", err)
			}
			if !matches {
				// Skip widgets that don't match the expression
				continue
			}
		}

		results = append(results, domain.DeserializeWidget(data))
	}

	return results, nil
}

// evaluateExpression evaluates an authorization expression against widget data.
// The repository decides which expression languages it supports.
func (r *InMemoryWidgetRepository) evaluateExpression(data domain.WidgetData, expr cazi.Expression) (bool, error) {
	switch expr.Language {
	case "cel":
		return r.evaluateCEL(data, expr.Expression)
	default:
		return false, fmt.Errorf("unsupported expression language: %s (repository only supports: cel)", expr.Language)
	}
}

// evaluateCEL evaluates a CEL expression against widget data.
// In a real database, this would be translated to a WHERE clause.
func (r *InMemoryWidgetRepository) evaluateCEL(data domain.WidgetData, expression string) (bool, error) {
	// Compile the CEL expression
	ast, issues := r.celEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}

	// Create a program from the AST
	prg, err := r.celEnv.Program(ast)
	if err != nil {
		return false, fmt.Errorf("CEL program creation error: %w", err)
	}

	// Prepare input data for CEL evaluation with widget object
	input := map[string]any{
		"widget": map[string]any{
			"owner_id":    data.OwnerID,
			"id":          data.ID,
			"name":        data.Name,
			"description": data.Description,
		},
	}

	// Evaluate the expression
	result, _, err := prg.Eval(input)
	if err != nil {
		return false, fmt.Errorf("CEL evaluation error: %w", err)
	}

	// Convert result to boolean
	boolResult, ok := result.(types.Bool)
	if !ok {
		return false, fmt.Errorf("CEL expression must return a boolean, got: %v", result.Type())
	}

	return bool(boolResult), nil
}
