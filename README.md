# CAZI - Common Authorization Interface

A Go interface specification for authorization systems.

## Purpose

Define a common interface that decouples authorization decisions from their enforcement. Services check authorization through a standard interface; implementations determine policy.

## Core Interface

```go
type Interface interface {
    Check(ctx context.Context, req CheckRequest) (CheckResponse, error)
}
```

A `Check` operation accepts a subject, verb, and object, returning:
- **Allow/Deny**: Boolean decision
- **Conditional**: Expression for caller to evaluate

## Key Concepts

**Assertions**: Subjects and objects are represented as assertions—resource references, claims, or opaque payloads. This allows flexible identity representation without prescribing verification mechanisms.

**Conditional Responses**: Authorization systems can return expressions (CEL, Rego, etc.) instead of making decisions directly. Callers evaluate these expressions against their own data, enabling authorization at query time.

**Authorization Context**: Responses may include claims about the requester for downstream use without additional lookups.

## Structure

- `pkg/cazi/` - Core interface and types
- `pkg/claims/` - Helpers for type-safe claim access
- `examples/widgets-service/` - Reference implementation

## Example Use

An authorization system returns a CEL expression like `owner_id == 'user-123'`. The caller passes this to its repository as a database filter. Resources that don't match are not found—indistinguishable from non-existent resources.
