# Widgets Service Example

A simple microservice demonstrating usage of the CAZI interface with a clean layered architecture.

## Architecture

```
presentation/   - HTTP handlers (API layer)
application/    - Application services (uses CAZI)
domain/         - Business logic, domain models, repository interface
infrastructure/ - Implementations (in-memory repo, local CAZI)
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed flow diagrams and design decisions.

## Domain

The service manages `Widget` resources with basic create and read operations.

The domain layer uses CAZI types directly (specifically `cazi.Expression`) as part of its interface. CAZI is treated as a foundational standard interface rather than an external dependency to be abstracted away.

## Running the Service

The widgets-service is an independent Go module that can be built and run standalone:

```bash
cd examples/widgets-service
go build .
./widgets-service
```

Or run directly:
```bash
go run main.go
```

The service starts on port 8080.

## Usage Examples

Create a widget (as user "alice"):
```bash
curl -X POST http://localhost:8080/widgets \
  -H "X-User-ID: alice" \
  -H "Content-Type: application/json" \
  -d '{"id":"widget-1","name":"My Widget","description":"A test widget"}'
```

Get a widget (as owner):
```bash
curl http://localhost:8080/widgets/widget-1 \
  -H "X-User-ID: alice"
```

Get a widget (as non-owner - will be denied):
```bash
curl http://localhost:8080/widgets/widget-1 \
  -H "X-User-ID: bob"
```

## Authorization Policy

The local CAZI implementation enforces a simple policy:
- **Create**: Any user can create widgets (returns `DecisionAllow`)
- **Read**: Users can only read widgets they own (returns `DecisionConditional` with constraint expression)

### Authorization Constraints with CEL

Instead of querying data directly, the local CAZI implementation returns **CEL (Common Expression Language) expressions** that represent authorization constraints. These constraints are passed down to the repository layer and applied as part of the database query (WHERE clause).

For example, when checking read access, CAZI returns:
```json
{
  "decision": "conditional",
  "condition": {
    "language": "cel",
    "expression": "owner_id == 'alice'"
  }
}
```

The application service then:
1. Passes the `cazi.Expression` directly to the repository's `FindByID` method
2. The repository checks if it supports the expression language and evaluates accordingly
3. For CEL expressions, the repository evaluates them against widget data (in a real DB, CEL would translate to WHERE clause)
4. Returns "not found" if either the widget doesn't exist OR doesn't satisfy the expression

**Design Philosophy**: CAZI is treated as a standard interface that domain models can depend on directly, similar to how they might depend on `context.Context` or other foundational interfaces. This keeps the architecture simple while allowing different repository implementations to support different expression languages.

**Authorization Context**: CAZI responses can optionally include authorization context inspired by the [Transaction Token specification](https://www.ietf.org/archive/id/draft-ietf-oauth-transaction-tokens-06.html):
- **Requester Context**: Claims about the requester (user ID, roles, attributes, etc.)
- **Transaction Context**: Claims about the operation (operation type, resource type, environmental factors, etc.)

This context can be used by downstream services for logging, auditing, or additional policy decisions without requiring additional lookups.

### Security Benefits

**Information Hiding**: When a user tries to access a widget they're not authorized to see, they get the same "not found" error as if the widget didn't exist. This prevents leaking information about what resources exist in the system.

**Eager Filtering**: Authorization constraints are applied at the database layer, meaning:
- No redundant data fetches
- Better performance (single query instead of fetch-then-check)
- Can leverage database indexes
- Natural fit for database-backed authorization patterns

**Extensibility**: The repository interface accepts any expression language and decides which ones it supports:
- Different repositories can support different expression languages
- Easy to add support for new expression languages (Rego, custom DSL, etc.)
- Clear error messages when an unsupported language is used

