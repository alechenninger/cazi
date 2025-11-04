package infrastructure

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"testing"

	"widgets-service/domain"

	"github.com/alechenninger/cazi/pkg/cazi"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupContainerRuntime configures environment for Docker or Podman.
// Returns true if a container runtime is available, false otherwise.
func setupContainerRuntime(t *testing.T) bool {
	t.Helper()

	// Check if DOCKER_HOST is already set (e.g., by CI or manual config)
	if os.Getenv("DOCKER_HOST") != "" {
		return true
	}

	// Try Docker first
	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.Command("docker", "info").Run(); err == nil {
			t.Log("Using Docker")
			return true
		}
	}

	// Try Podman
	if podmanPath, err := exec.LookPath("podman"); err == nil {
		// Check if podman is accessible
		if err := exec.Command(podmanPath, "info").Run(); err == nil {
			// Get Podman socket path
			cmd := exec.Command(podmanPath, "info", "--format", "{{.Host.RemoteSocket.Path}}")
			output, err := cmd.Output()
			if err == nil && len(output) > 0 {
				socketPath := string(output[:len(output)-1]) // trim newline
				t.Logf("Using Podman with socket: %s", socketPath)

				// Configure testcontainers for Podman
				os.Setenv("DOCKER_HOST", socketPath)
				os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
				return true
			}
		}
	}

	return false
}

// setupPostgresContainer starts a PostgreSQL container for testing.
func setupPostgresContainer(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, *sql.DB) {
	t.Helper()

	// Check if container runtime is available
	if !setupContainerRuntime(t) {
		t.Skip("No container runtime (Docker/Podman) available - skipping PostgreSQL tests")
		return nil, nil
	}

	// Create PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Skipf("Failed to start postgres container (this is OK if container runtime isn't configured): %v", err)
		return nil, nil
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	t.Logf("PostgreSQL container started: %s", connStr)
	return pgContainer, db
}

func TestPostgresRepository_SaveAndFindByID(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Create and save a widget
	widget := domain.NewWidget("widget-1", "Test Widget", "A test widget", "user-alice")
	err = repo.Save(ctx, widget)
	if err != nil {
		t.Fatalf("failed to save widget: %v", err)
	}

	// Retrieve the widget without authorization constraint
	retrieved, err := repo.FindByID(ctx, "widget-1", cazi.Expression{})
	if err != nil {
		t.Fatalf("failed to retrieve widget: %v", err)
	}

	// Verify the retrieved widget
	if retrieved.ID() != widget.ID() {
		t.Errorf("expected ID %s, got %s", widget.ID(), retrieved.ID())
	}
	if retrieved.Name() != widget.Name() {
		t.Errorf("expected name %s, got %s", widget.Name(), retrieved.Name())
	}
	if retrieved.Description() != widget.Description() {
		t.Errorf("expected description %s, got %s", widget.Description(), retrieved.Description())
	}
	if retrieved.OwnerID() != widget.OwnerID() {
		t.Errorf("expected owner %s, got %s", widget.OwnerID(), retrieved.OwnerID())
	}
}

func TestPostgresRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Try to retrieve a non-existent widget
	_, err = repo.FindByID(ctx, "non-existent", cazi.Expression{})
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPostgresRepository_FindByID_WithAuthzExpression(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Create and save widgets for different users
	aliceWidget := domain.NewWidget("widget-alice", "Alice's Widget", "Owned by Alice", "user-alice")
	bobWidget := domain.NewWidget("widget-bob", "Bob's Widget", "Owned by Bob", "user-bob")

	if err := repo.Save(ctx, aliceWidget); err != nil {
		t.Fatalf("failed to save alice's widget: %v", err)
	}
	if err := repo.Save(ctx, bobWidget); err != nil {
		t.Fatalf("failed to save bob's widget: %v", err)
	}

	// Alice tries to retrieve her own widget with authorization expression
	aliceExpr := cazi.Expression{
		Language:   "cel",
		Expression: "widget.owner_id == 'user-alice'",
	}
	retrieved, err := repo.FindByID(ctx, "widget-alice", aliceExpr)
	if err != nil {
		t.Fatalf("alice should be able to retrieve her widget: %v", err)
	}
	if retrieved.ID() != aliceWidget.ID() {
		t.Errorf("expected widget %s, got %s", aliceWidget.ID(), retrieved.ID())
	}

	// Alice tries to retrieve Bob's widget with her authorization expression
	_, err = repo.FindByID(ctx, "widget-bob", aliceExpr)
	if err != domain.ErrNotFound {
		t.Errorf("alice should not be able to retrieve bob's widget, expected ErrNotFound, got %v", err)
	}

	// Bob tries to retrieve his own widget with authorization expression
	bobExpr := cazi.Expression{
		Language:   "cel",
		Expression: "widget.owner_id == 'user-bob'",
	}
	retrieved, err = repo.FindByID(ctx, "widget-bob", bobExpr)
	if err != nil {
		t.Fatalf("bob should be able to retrieve his widget: %v", err)
	}
	if retrieved.ID() != bobWidget.ID() {
		t.Errorf("expected widget %s, got %s", bobWidget.ID(), retrieved.ID())
	}
}

func TestPostgresRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Create and save multiple widgets
	widgets := []*domain.Widget{
		domain.NewWidget("widget-1", "Widget 1", "First widget", "user-alice"),
		domain.NewWidget("widget-2", "Widget 2", "Second widget", "user-alice"),
		domain.NewWidget("widget-3", "Widget 3", "Third widget", "user-bob"),
	}

	for _, w := range widgets {
		if err := repo.Save(ctx, w); err != nil {
			t.Fatalf("failed to save widget %s: %v", w.ID(), err)
		}
	}

	// Retrieve all widgets without filter
	allWidgets, err := repo.FindAll(ctx, cazi.Expression{})
	if err != nil {
		t.Fatalf("failed to find all widgets: %v", err)
	}
	if len(allWidgets) != 3 {
		t.Errorf("expected 3 widgets, got %d", len(allWidgets))
	}
}

func TestPostgresRepository_FindAll_WithAuthzExpression(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Create and save multiple widgets for different users
	widgets := []*domain.Widget{
		domain.NewWidget("widget-alice-1", "Alice Widget 1", "Owned by Alice", "user-alice"),
		domain.NewWidget("widget-alice-2", "Alice Widget 2", "Also owned by Alice", "user-alice"),
		domain.NewWidget("widget-bob-1", "Bob Widget 1", "Owned by Bob", "user-bob"),
		domain.NewWidget("widget-bob-2", "Bob Widget 2", "Also owned by Bob", "user-bob"),
		domain.NewWidget("widget-bob-3", "Bob Widget 3", "Another Bob widget", "user-bob"),
	}

	for _, w := range widgets {
		if err := repo.Save(ctx, w); err != nil {
			t.Fatalf("failed to save widget %s: %v", w.ID(), err)
		}
	}

	// Alice retrieves her widgets
	aliceExpr := cazi.Expression{
		Language:   "cel",
		Expression: "widget.owner_id == 'user-alice'",
	}
	aliceWidgets, err := repo.FindAll(ctx, aliceExpr)
	if err != nil {
		t.Fatalf("failed to find alice's widgets: %v", err)
	}
	if len(aliceWidgets) != 2 {
		t.Errorf("expected 2 widgets for alice, got %d", len(aliceWidgets))
	}
	for _, w := range aliceWidgets {
		if w.OwnerID() != "user-alice" {
			t.Errorf("alice's query returned widget owned by %s", w.OwnerID())
		}
	}

	// Bob retrieves his widgets
	bobExpr := cazi.Expression{
		Language:   "cel",
		Expression: "widget.owner_id == 'user-bob'",
	}
	bobWidgets, err := repo.FindAll(ctx, bobExpr)
	if err != nil {
		t.Fatalf("failed to find bob's widgets: %v", err)
	}
	if len(bobWidgets) != 3 {
		t.Errorf("expected 3 widgets for bob, got %d", len(bobWidgets))
	}
	for _, w := range bobWidgets {
		if w.OwnerID() != "user-bob" {
			t.Errorf("bob's query returned widget owned by %s", w.OwnerID())
		}
	}
}

func TestPostgresRepository_FindAll_NoResults(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Try to find widgets with a filter that matches nothing
	expr := cazi.Expression{
		Language:   "cel",
		Expression: "widget.owner_id == 'user-nobody'",
	}
	widgets, err := repo.FindAll(ctx, expr)
	if err != nil {
		t.Fatalf("failed to find widgets: %v", err)
	}
	if len(widgets) != 0 {
		t.Errorf("expected 0 widgets, got %d", len(widgets))
	}
}

func TestPostgresRepository_Update(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Create and save a widget
	widget := domain.NewWidget("widget-1", "Original Name", "Original Description", "user-alice")
	if err := repo.Save(ctx, widget); err != nil {
		t.Fatalf("failed to save widget: %v", err)
	}

	// Update the widget (Save handles upsert)
	updatedWidget := domain.NewWidget("widget-1", "Updated Name", "Updated Description", "user-alice")
	if err := repo.Save(ctx, updatedWidget); err != nil {
		t.Fatalf("failed to update widget: %v", err)
	}

	// Retrieve and verify the update
	retrieved, err := repo.FindByID(ctx, "widget-1", cazi.Expression{})
	if err != nil {
		t.Fatalf("failed to retrieve updated widget: %v", err)
	}
	if retrieved.Name() != "Updated Name" {
		t.Errorf("expected updated name 'Updated Name', got %s", retrieved.Name())
	}
	if retrieved.Description() != "Updated Description" {
		t.Errorf("expected updated description 'Updated Description', got %s", retrieved.Description())
	}
}

func TestPostgresRepository_ComplexAuthzExpression(t *testing.T) {
	ctx := context.Background()
	pgContainer, db := setupPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)
	defer db.Close()

	repo, err := NewPostgresWidgetRepository(db)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Create widgets with different names
	widgets := []*domain.Widget{
		domain.NewWidget("widget-1", "Test Widget", "Description", "user-alice"),
		domain.NewWidget("widget-2", "Production Widget", "Description", "user-alice"),
		domain.NewWidget("widget-3", "Test Widget", "Description", "user-bob"),
	}

	for _, w := range widgets {
		if err := repo.Save(ctx, w); err != nil {
			t.Fatalf("failed to save widget %s: %v", w.ID(), err)
		}
	}

	// Complex expression: Alice's widgets that start with "Test"
	expr := cazi.Expression{
		Language:   "cel",
		Expression: "widget.owner_id == 'user-alice' && widget.name.startsWith('Test')",
	}

	results, err := repo.FindAll(ctx, expr)
	if err != nil {
		t.Fatalf("failed to find widgets with complex expression: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 widget, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID() != "widget-1" {
		t.Errorf("expected widget-1, got %s", results[0].ID())
	}
}
