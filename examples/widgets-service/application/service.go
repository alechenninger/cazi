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
	authzReq := cazi.CheckRequest{
		Subject: req.Subject,
		Verb:    "create",
		Object: cazi.Object{
			Token: cazi.ResourceReference{Type: "widget", ID: req.WidgetID},
		},
	}

	authzResp, err := s.authz.Check(ctx, authzReq)
	if err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	if authzResp.Decision != cazi.DecisionAllow {
		return nil, fmt.Errorf("access denied: cannot create widget")
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
			Token: cazi.ResourceReference{Type: "widget", ID: req.WidgetID},
		},
	}

	authzResp, err := s.authz.Check(ctx, authzReq)
	if err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	if authzResp.Decision == cazi.DecisionDeny {
		return nil, fmt.Errorf("access denied: cannot read widget")
	}

	// Pass authorization expression directly to repository if conditional
	// The repository decides which expression languages it supports
	widget, err := s.repo.FindByID(ctx, domain.WidgetID(req.WidgetID), authzResp.Condition)
	if err != nil {
		// "Not found" could mean either doesn't exist or not authorized (security feature)
		return nil, fmt.Errorf("widget not found or access denied")
	}

	return &WidgetResponse{
		ID:          string(widget.ID()),
		Name:        widget.Name(),
		Description: widget.Description(),
		OwnerID:     widget.OwnerID(),
	}, nil
}
