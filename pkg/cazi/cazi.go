package cazi

import (
	"context"
)

// Interface is the Common Authorization Interface (CAZI) in Go.
// It intentionally avoids transport details. Implementations may be local or remote.
type Interface interface {
	// Check answers whether a subject can perform a verb (relation) on an object.
	//
	// The result can be:
	//   - Allowed: true/false
	//   - Conditional: an expression the caller can evaluate in its own context
	Check(ctx context.Context, req CheckRequest) (CheckResponse, error)
}

// CheckRequest captures the inputs to an authorization check.
type CheckRequest struct {
	Subject          Subject // subject token with optional relation
	Verb             string  // verb/relation
	Object           Object  // object token
	ConsistencyToken []byte  // optional opaque token for causal consistency
}

// Subject represents the actor performing the action.
type Subject struct {
	Token    Token  // assertions about the subject
	Relation string // optional relation (e.g., "member")
}

// Object represents the target of the action.
type Object struct {
	Token Token // assertions about the object
}

// Token is a one-of representing assertions about a subject or object.
// Exactly one concrete token type should be used at a time.
// TODO: these are not always tokens; more like Assertion(s),
// some of which may be verifiable
type Token interface {
	isToken()
}

// Claims carries arbitrary JSON-like claims, optionally signed.
// TODO: can we make this the same type as ContextClaims?
type Claims struct {
	Claims map[string]any
}

func (Claims) isToken() {}

// OpaqueToken carries an opaque payload with a declared type, optionally signed (e.g., JWT).
type OpaqueToken struct {
	Type string // media/type or scheme identifier (e.g., "jwt")
	Raw  []byte // raw token bytes
}

func (OpaqueToken) isToken() {}

// ResourceReference identifies a resource by type and id.
type ResourceReference struct {
	Type string
	ID   string
}

func (ResourceReference) isToken() {}

// DecisionKind is the tri-state outcome for Check.
type DecisionKind int

const (
	DecisionUnknown     DecisionKind = iota // unspecified
	DecisionAllow                           // allowed = true
	DecisionDeny                            // allowed = false
	DecisionConditional                     // requires evaluating a returned expression
)

// CheckResponse is the outcome of a Check invocation.
type CheckResponse struct {
	Decision  DecisionKind         // allow/deny/conditional
	Condition Expression           // present when DecisionConditional (check Language != "" to detect if set)
	Context   AuthorizationContext // additional context about the authorization decision (maps may be nil if not provided)
}

// AuthorizationContext provides optional additional information about the authorization decision.
// Inspired by the Transaction Token specification (draft-ietf-oauth-transaction-tokens).
// See: https://www.ietf.org/archive/id/draft-ietf-oauth-transaction-tokens-06.html#name-jwt-body-claims
type AuthorizationContext struct {
	// RequesterContext contains claims about the requester (subject).
	// These are assertions about who is making the request, such as roles, attributes,
	// or other identity-related information that may be useful for downstream processing.
	RequesterContext ContextClaims

	// TransactionContext contains claims about the requested operation itself.
	// These are assertions about the transaction, such as environmental factors,
	// computed context, or other operation-related information.
	TransactionContext ContextClaims
}

// ContextClaims represents a set of authorization context claims as key-value pairs.
// This is JSON-compatible and can contain arbitrary structured data.
type ContextClaims map[string]any

// Claim provides type-safe access to a specific claim in ContextClaims.
type Claim[T any] struct {
	Get func(ContextClaims) (T, bool)
	Set func(ContextClaims, T)
}

// GetClaim retrieves a typed claim value from the context claims.
// This is a convenience function that provides an alternative syntax: claims.GetClaim(ClaimSub)
func GetClaim[T any](claims ContextClaims, claim Claim[T]) (T, bool) {
	return claim.Get(claims)
}

// SetClaim stores a typed claim value in the context claims.
// This is a convenience function that provides an alternative syntax: SetClaim(claims, ClaimSub, "user123")
func SetClaim[T any](claims ContextClaims, claim Claim[T], value T) {
	claim.Set(claims, value)
}

// Expression represents a condition the caller can evaluate.
// The language is intentionally unspecified; callers and implementations
// may agree on a language such as CEL, Rego, etc.
type Expression struct {
	Language   string // optional (e.g., "cel")
	Expression string // the expression to evaluate
}
