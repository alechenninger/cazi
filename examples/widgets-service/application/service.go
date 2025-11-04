package application

import (
	"context"
	"fmt"

	"widgets-service/domain"

	"github.com/alechenninger/cazi/pkg/cazi"
	"github.com/alechenninger/cazi/pkg/claims"
)

// CreateWidgetRequest is a value object for creating widgets.
type CreateWidgetRequest struct {
	Subject     cazi.Subject
	WidgetID    string
	Name        string
	Description string
}

// GetWidgetRequest is a value object for retrieving widgets.
type GetWidgetRequest struct {
	Subject  cazi.Subject
	WidgetID string
}

// ListWidgetsRequest is a value object for listing widgets.
type ListWidgetsRequest struct {
	Subject cazi.Subject
}

// WidgetResponse is a value object representing a widget in the application layer.
type WidgetResponse struct {
	ID          string
	Name        string
	Description string
	OwnerID     string
}

// WidgetService provides operations for managing widgets.
// It works with value objects only.
type WidgetService struct {
	repo  domain.WidgetRepository
	authz cazi.Interface
}

// NewWidgetService creates a new widget service.
func NewWidgetService(repo domain.WidgetRepository, authz cazi.Interface) *WidgetService {
	return &WidgetService{
		repo:  repo,
		authz: authz,
	}
}

// CreateWidget creates a new widget for the given subject.
func (s *WidgetService) CreateWidget(ctx context.Context, req CreateWidgetRequest) (*WidgetResponse, error) {
	// Check authorization
	authzResp, err := s.authz.Check(ctx, cazi.CheckRequest{
		Subject: req.Subject,
		Verb:    "create",
		Object: cazi.Object{
			Assertion: cazi.ResourceReference{Type: "widget", ID: req.WidgetID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	if authzResp.Decision != cazi.DecisionAllow {
		return nil, domain.ErrUnauthorized
	}

	// Extract user ID from authorization context
	// This decouples us from the specific token type
	ownerID, ok := claims.Sub.Get(authzResp.Context.RequesterContext)
	if !ok {
		return nil, fmt.Errorf("authorization context did not provide subject identifier")
	}

	// Create domain object
	widget := domain.NewWidget(
		domain.WidgetID(req.WidgetID),
		req.Name,
		req.Description,
		ownerID,
	)

	// Persist
	if err := s.repo.Save(ctx, widget); err != nil {
		return nil, fmt.Errorf("failed to save widget: %w", err)
	}

	// Return response
	return &WidgetResponse{
		ID:          string(widget.ID()),
		Name:        widget.Name(),
		Description: widget.Description(),
		OwnerID:     widget.OwnerID(),
	}, nil
}

// GetWidget retrieves a widget by ID for the given subject.
func (s *WidgetService) GetWidget(ctx context.Context, req GetWidgetRequest) (*WidgetResponse, error) {
	// Check authorization
	authzReq := cazi.CheckRequest{
		Subject: req.Subject,
		Verb:    "read",
		Object: cazi.Object{
			Assertion: cazi.ResourceReference{Type: "widget", ID: req.WidgetID},
		},
	}

	authzResp, err := s.authz.Check(ctx, authzReq)
	if err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	if authzResp.Decision == cazi.DecisionDeny {
		return nil, domain.ErrUnauthorized
	}

	// Pass authorization expression directly to repository if conditional
	// The repository decides which expression languages it supports
	widget, err := s.repo.FindByID(ctx, domain.WidgetID(req.WidgetID), authzResp.Condition)
	if err != nil {
		// Return the error as-is so caller can distinguish ErrNotFound from other errors
		return nil, err
	}

	return &WidgetResponse{
		ID:          string(widget.ID()),
		Name:        widget.Name(),
		Description: widget.Description(),
		OwnerID:     widget.OwnerID(),
	}, nil
}

// ListWidgets retrieves all widgets the subject is authorized to see.
func (s *WidgetService) ListWidgets(ctx context.Context, req ListWidgetsRequest) ([]*WidgetResponse, error) {
	// Get authorization filter for listing widgets
	authzResp, err := s.authz.ListObjects(ctx, cazi.ListObjectsRequest{
		Subject:    req.Subject,
		Verb:       "read",
		ObjectType: "widget",
	})
	if err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	if authzResp.Decision == cazi.DecisionDeny {
		return nil, domain.ErrUnauthorized
	}

	// Pass authorization expression directly to repository
	// The repository applies it as a filter (e.g., WHERE clause)
	widgets, err := s.repo.FindAll(ctx, authzResp.Condition)
	if err != nil {
		return nil, fmt.Errorf("failed to list widgets: %w", err)
	}

	// Convert to response objects
	responses := make([]*WidgetResponse, len(widgets))
	for i, widget := range widgets {
		responses[i] = &WidgetResponse{
			ID:          string(widget.ID()),
			Name:        widget.Name(),
			Description: widget.Description(),
			OwnerID:     widget.OwnerID(),
		}
	}

	return responses, nil
}
