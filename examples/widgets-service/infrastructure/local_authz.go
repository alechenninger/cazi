package infrastructure

import (
	"context"
	"fmt"

	"github.com/alechenninger/cazi/pkg/cazi"
	"github.com/alechenninger/cazi/pkg/claims"
)

// LocalAuthz is a local implementation of the CAZI interface that returns
// authorization constraints as conditional expressions rather than querying data.
type LocalAuthz struct{}

// NewLocalAuthz creates a new local authorization implementation.
func NewLocalAuthz() *LocalAuthz {
	return &LocalAuthz{}
}

// Check implements the CAZI Check operation with hardcoded policy:
// - Users can "create" widgets (unconditional allow)
// - Users can "read" widgets they own (returns conditional expression)
func (a *LocalAuthz) Check(ctx context.Context, req cazi.CheckRequest) (cazi.CheckResponse, error) {
	// Extract subject user ID
	subjectRes, ok := req.Subject.Token.(cazi.ResourceReference)
	if !ok {
		return cazi.CheckResponse{Decision: cazi.DecisionDeny}, fmt.Errorf("subject must be a ResourceReference")
	}
	if subjectRes.Type != "user" {
		return cazi.CheckResponse{Decision: cazi.DecisionDeny}, fmt.Errorf("subject must be of type 'user'")
	}
	userID := subjectRes.ID

	// Extract object widget ID
	objectRes, ok := req.Object.Token.(cazi.ResourceReference)
	if !ok {
		return cazi.CheckResponse{Decision: cazi.DecisionDeny}, fmt.Errorf("object must be a ResourceReference")
	}
	if objectRes.Type != "widget" {
		return cazi.CheckResponse{Decision: cazi.DecisionDeny}, fmt.Errorf("object must be of type 'widget'")
	}

	// Hardcoded policy
	switch req.Verb {
	case "create":
		// Anyone can create widgets
		// Include authorization context with requester info
		reqCtx := make(cazi.ContextClaims)
		claims.Sub.Set(reqCtx, userID)

		return cazi.CheckResponse{
			Decision: cazi.DecisionAllow,
			Context: cazi.AuthorizationContext{
				RequesterContext: reqCtx,
			},
		}, nil

	case "read":
		// Users can read widgets they own - return CEL expression
		// The expression represents the constraint that must be evaluated against actual data

		// Build authorization context using claims
		reqCtx := make(cazi.ContextClaims)
		claims.Sub.Set(reqCtx, userID)

		return cazi.CheckResponse{
			Decision: cazi.DecisionConditional,
			Condition: cazi.Expression{
				Language:   "cel",
				Expression: fmt.Sprintf("owner_id == '%s'", userID),
			},
			Context: cazi.AuthorizationContext{
				RequesterContext: reqCtx,
			},
		}, nil

	default:
		return cazi.CheckResponse{Decision: cazi.DecisionDeny}, fmt.Errorf("unknown verb: %s", req.Verb)
	}
}
