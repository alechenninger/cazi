# CAZI - Common Authorization Interface

A common authorization interface defined via gRPC/protobuf.

## Project Structure

```
cazi/
├── proto/              # Interface definitions
├── implementations/    # Different implementations of the CAZI interface
└── examples/          # Example services demonstrating usage
```

## Overview

This project defines a common authorization interface that can be:
- Implemented by various authorization providers
- Used by services needing authorization capabilities
- Enhanced with optional authorization context (requester and transaction claims)

The interface is inspired by modern authorization patterns including [Transaction Tokens (draft-ietf-oauth-transaction-tokens)](https://www.ietf.org/archive/id/draft-ietf-oauth-transaction-tokens-06.html).

## Features

### Type-Safe Standard Claims

CAZI provides a type-safe mechanism for working with authorization context claims:

```go
// Using standard claims
reqCtx := make(cazi.ContextClaims)
cazi.ClaimSub.Set(reqCtx, "user123")
cazi.ClaimRoles.Set(reqCtx, []string{"admin", "editor"})

// Reading claims with type safety
if sub, ok := cazi.ClaimSub.Get(reqCtx); ok {
    fmt.Printf("Subject: %s\n", sub)
}

// With default fallback
username := cazi.ClaimPreferredUsername.GetOrDefault(reqCtx, "anonymous")
```

### Custom Claims

Define your own typed claims using the same pattern:

```go
var (
    ClaimTenantID = cazi.Claim[string]{Key: "tenant_id"}
    ClaimClearanceLevel = cazi.Claim[int]{Key: "clearance_level"}
)

ClaimTenantID.Set(reqCtx, "tenant-456")
```

See [examples/custom-claims](examples/custom-claims/) for a complete demonstration.

## Getting Started

See the [widgets-service example](examples/widgets-service/) for a complete demonstration of:
- Using the CAZI interface in a Go service with clean layered architecture
- Treating CAZI as a foundational interface that domain models can depend on directly
- A local CAZI implementation that returns CEL (Common Expression Language) authorization constraints
- Passing authorization expressions down to the repository/database layer
- Repositories that decide which expression languages they support (CEL, Rego, etc.)
- Evaluating expressions as database filters (WHERE clauses) for eager authorization
- Security benefits: "not found" = "not authorized" (information hiding)

The widgets-service is a standalone Go module that can be built independently.

