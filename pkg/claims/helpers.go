package claims

import (
	"github.com/alechenninger/cazi/pkg/cazi"
)

// TopLevel creates a claim for a top-level key in ContextClaims.
// Example: TopLevel[string]("sub") accesses claims["sub"]
func TopLevel[T any](key string) cazi.Claim[T] {
	return cazi.Claim[T]{
		Get: func(claims cazi.ContextClaims) (T, bool) {
			val, ok := claims[key] // Safe even if claims is nil
			if !ok {
				var zero T
				return zero, false
			}
			typed, ok := val.(T)
			return typed, ok
		},
		Set: func(claims cazi.ContextClaims, value T) {
			if claims != nil {
				claims[key] = value
			}
		},
	}
}

// Nested creates a claim for a nested path in ContextClaims.
// Example: Nested[string]("address", "city") accesses claims["address"]["city"]
// When setting, it creates intermediate maps as needed.
func Nested[T any](path ...string) cazi.Claim[T] {
	if len(path) == 0 {
		panic("Nested claim requires at least one path element")
	}

	return cazi.Claim[T]{
		Get: func(claims cazi.ContextClaims) (T, bool) {
			var zero T
			// Traverse the path (safe even if claims is nil)
			current := map[string]any(claims)
			for i, key := range path[:len(path)-1] {
				val, ok := current[key]
				if !ok {
					return zero, false
				}
				nested, ok := val.(map[string]any)
				if !ok {
					return zero, false
				}
				current = nested
				_ = i // avoid unused warning
			}

			// Get the final value
			finalKey := path[len(path)-1]
			val, ok := current[finalKey]
			if !ok {
				return zero, false
			}
			typed, ok := val.(T)
			return typed, ok
		},
		Set: func(claims cazi.ContextClaims, value T) {
			if claims == nil {
				return
			}

			// Traverse and create intermediate maps as needed
			current := map[string]any(claims)
			for _, key := range path[:len(path)-1] {
				val, ok := current[key]
				if !ok {
					// Create intermediate map
					nested := make(map[string]any)
					current[key] = nested
					current = nested
				} else {
					nested, ok := val.(map[string]any)
					if !ok {
						// Path exists but is not a map, can't traverse
						return
					}
					current = nested
				}
			}

			// Set the final value
			finalKey := path[len(path)-1]
			current[finalKey] = value
		},
	}
}
