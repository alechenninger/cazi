package infrastructure

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"widgets-service/domain"

	"github.com/alechenninger/cazi/pkg/cazi"
	"github.com/google/cel-go/cel"
	cel2sql "github.com/spandigital/cel2sql/v3"
	"github.com/spandigital/cel2sql/v3/pg"
)

//go:embed schema.sql
var schemaSQL string

// PostgresWidgetRepository implements the WidgetRepository interface using PostgreSQL.
type PostgresWidgetRepository struct {
	db           *sql.DB
	celEnv       *cel.Env
	typeProvider pg.TypeProvider
}

// NewPostgresWidgetRepository creates a new PostgreSQL-backed widget repository.
func NewPostgresWidgetRepository(db *sql.DB) (*PostgresWidgetRepository, error) {
	// Initialize schema
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Create CEL type provider for SQL conversion
	// Define the schema matching our CEL expressions (e.g., "widget.owner_id")
	schema := pg.NewSchema([]pg.FieldSchema{
		{Name: "id", Type: "text"},
		{Name: "name", Type: "text"},
		{Name: "description", Type: "text"},
		{Name: "owner_id", Type: "text"},
	})

	// NewTypeProvider expects a map of table name to schema
	schemas := map[string]pg.Schema{
		"widget": schema,
	}
	typeProvider := pg.NewTypeProvider(schemas)

	// Create CEL environment with the widget type
	celEnv, err := cel.NewEnv(
		cel.CustomTypeProvider(typeProvider),
		cel.Variable("widget", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &PostgresWidgetRepository{
		db:           db,
		celEnv:       celEnv,
		typeProvider: typeProvider,
	}, nil
}

// Save stores a widget in the database.
func (r *PostgresWidgetRepository) Save(ctx context.Context, widget *domain.Widget) error {
	data := widget.Serialize()

	query := `
		INSERT INTO widgets (id, name, description, owner_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			owner_id = EXCLUDED.owner_id,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query, data.ID, data.Name, data.Description, data.OwnerID)
	if err != nil {
		return fmt.Errorf("failed to save widget: %w", err)
	}

	return nil
}

// FindByID retrieves a widget by ID, applying an optional authorization expression.
// If an expression is provided (Language != ""), it's evaluated as part of the query filter.
// Only returns the widget if it both exists and satisfies the authorization expression.
// Returns domain.ErrNotFound if the widget doesn't exist or doesn't satisfy the expression.
func (r *PostgresWidgetRepository) FindByID(ctx context.Context, id domain.WidgetID, authzExpression cazi.Expression) (*domain.Widget, error) {
	// Use "widget" as table alias to match CEL variable name
	query := "SELECT widget.id, widget.name, widget.description, widget.owner_id FROM widgets AS widget WHERE widget.id = $1"
	args := []interface{}{string(id)}

	// Apply authorization expression as a WHERE clause
	if authzExpression.Language != "" {
		whereClause, params, err := r.expressionToSQL(authzExpression, len(args))
		if err != nil {
			return nil, fmt.Errorf("failed to convert authorization expression to SQL: %w", err)
		}

		query += " AND " + whereClause
		args = append(args, params...)
	}

	var data domain.WidgetData
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&data.ID,
		&data.Name,
		&data.Description,
		&data.OwnerID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query widget: %w", err)
	}

	return domain.DeserializeWidget(data), nil
}

// FindAll retrieves all widgets, optionally filtered by an authorization expression.
// If an expression is provided (Language != ""), it's evaluated as part of the query filter.
// Only widgets that satisfy the expression are returned.
func (r *PostgresWidgetRepository) FindAll(ctx context.Context, authzExpression cazi.Expression) ([]*domain.Widget, error) {
	// Use "widget" as table alias to match CEL variable name
	query := "SELECT widget.id, widget.name, widget.description, widget.owner_id FROM widgets AS widget"
	var args []interface{}

	// Apply authorization expression as a WHERE clause
	if authzExpression.Language != "" {
		whereClause, params, err := r.expressionToSQL(authzExpression, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to convert authorization expression to SQL: %w", err)
		}

		query += " WHERE " + whereClause
		args = params
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query widgets: %w", err)
	}
	defer rows.Close()

	var widgets []*domain.Widget
	for rows.Next() {
		var data domain.WidgetData
		if err := rows.Scan(&data.ID, &data.Name, &data.Description, &data.OwnerID); err != nil {
			return nil, fmt.Errorf("failed to scan widget row: %w", err)
		}
		widgets = append(widgets, domain.DeserializeWidget(data))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating widget rows: %w", err)
	}

	return widgets, nil
}

// expressionToSQL converts a CAZI expression to a SQL WHERE clause using cel2sql.
// The paramOffset is the number of existing parameters in the query, used to renumber placeholders.
// Returns the SQL clause and any parameters for parameterized queries.
func (r *PostgresWidgetRepository) expressionToSQL(expr cazi.Expression, paramOffset int) (string, []interface{}, error) {
	if expr.Language != "cel" {
		return "", nil, fmt.Errorf("unsupported expression language: %s (only 'cel' is supported)", expr.Language)
	}

	// Compile the CEL expression
	ast, issues := r.celEnv.Compile(expr.Expression)
	if issues != nil && issues.Err() != nil {
		return "", nil, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	// Convert CEL AST to SQL using cel2sql
	result, err := cel2sql.ConvertParameterized(ast)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert CEL to SQL: %w", err)
	}

	// Renumber placeholders if we have existing parameters
	sql := result.SQL
	if paramOffset > 0 {
		sql = renumberPlaceholders(sql, paramOffset)
	}

	return sql, result.Parameters, nil
}

// renumberPlaceholders adjusts PostgreSQL placeholder numbers ($1, $2, etc.) by an offset.
// For example, with offset=1: "$1 AND $2" becomes "$2 AND $3"
func renumberPlaceholders(sql string, offset int) string {
	// Simple approach: replace $N with $(N+offset)
	// This works for the common case where placeholders are sequential
	var result []byte
	i := 0
	for i < len(sql) {
		if sql[i] == '$' && i+1 < len(sql) && sql[i+1] >= '0' && sql[i+1] <= '9' {
			// Found a placeholder
			j := i + 1
			for j < len(sql) && sql[j] >= '0' && sql[j] <= '9' {
				j++
			}
			// Extract the number
			num := 0
			for k := i + 1; k < j; k++ {
				num = num*10 + int(sql[k]-'0')
			}
			// Write the renumbered placeholder
			result = append(result, '$')
			result = append(result, []byte(fmt.Sprintf("%d", num+offset))...)
			i = j
		} else {
			result = append(result, sql[i])
			i++
		}
	}
	return string(result)
}

// Close closes the database connection.
func (r *PostgresWidgetRepository) Close() error {
	return r.db.Close()
}
