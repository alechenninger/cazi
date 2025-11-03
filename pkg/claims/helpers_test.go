package claims_test

import (
	"testing"

	"github.com/alechenninger/cazi/pkg/cazi"
	"github.com/alechenninger/cazi/pkg/claims"
)

func TestTopLevel(t *testing.T) {
	t.Run("Get from populated claims", func(t *testing.T) {
		testClaim := claims.TopLevel[string]("test_key")
		c := cazi.Claims{"test_key": "test_value"}

		value, ok := testClaim.Get(c)
		if !ok {
			t.Fatal("expected Get to return true")
		}
		if value != "test_value" {
			t.Errorf("expected 'test_value', got '%s'", value)
		}
	})

	t.Run("Get from nil claims", func(t *testing.T) {
		testClaim := claims.TopLevel[string]("test_key")
		var c cazi.Claims

		value, ok := testClaim.Get(c)
		if ok {
			t.Fatal("expected Get to return false for nil claims")
		}
		if value != "" {
			t.Errorf("expected zero value, got '%s'", value)
		}
	})

	t.Run("Get missing key", func(t *testing.T) {
		testClaim := claims.TopLevel[string]("missing")
		c := cazi.Claims{"other": "value"}

		value, ok := testClaim.Get(c)
		if ok {
			t.Fatal("expected Get to return false for missing key")
		}
		if value != "" {
			t.Errorf("expected zero value, got '%s'", value)
		}
	})

	t.Run("Get wrong type", func(t *testing.T) {
		testClaim := claims.TopLevel[string]("test_key")
		c := cazi.Claims{"test_key": 123}

		value, ok := testClaim.Get(c)
		if ok {
			t.Fatal("expected Get to return false for wrong type")
		}
		if value != "" {
			t.Errorf("expected zero value, got '%s'", value)
		}
	})

	t.Run("Set on claims", func(t *testing.T) {
		testClaim := claims.TopLevel[string]("test_key")
		c := make(cazi.Claims)

		testClaim.Set(c, "new_value")

		if c["test_key"] != "new_value" {
			t.Errorf("expected 'new_value', got '%v'", c["test_key"])
		}
	})

	t.Run("Set on nil claims does not panic", func(t *testing.T) {
		testClaim := claims.TopLevel[string]("test_key")
		var c cazi.Claims

		// Should not panic
		testClaim.Set(c, "value")
	})

	t.Run("Works with different types", func(t *testing.T) {
		intClaim := claims.TopLevel[int]("int_key")
		sliceClaim := claims.TopLevel[[]string]("slice_key")
		c := make(cazi.Claims)

		intClaim.Set(c, 42)
		sliceClaim.Set(c, []string{"a", "b"})

		intVal, ok := intClaim.Get(c)
		if !ok || intVal != 42 {
			t.Errorf("expected 42, got %d (ok=%v)", intVal, ok)
		}

		sliceVal, ok := sliceClaim.Get(c)
		if !ok || len(sliceVal) != 2 {
			t.Errorf("expected []string with 2 elements, got %v (ok=%v)", sliceVal, ok)
		}
	})
}

func TestNested(t *testing.T) {
	t.Run("Get nested value", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		c := cazi.Claims{
			"address": map[string]any{
				"city": "New York",
			},
		}

		value, ok := cityClaim.Get(c)
		if !ok {
			t.Fatal("expected Get to return true")
		}
		if value != "New York" {
			t.Errorf("expected 'New York', got '%s'", value)
		}
	})

	t.Run("Get deeply nested value", func(t *testing.T) {
		claim := claims.Nested[string]("a", "b", "c", "d")
		c := cazi.Claims{
			"a": map[string]any{
				"b": map[string]any{
					"c": map[string]any{
						"d": "deep",
					},
				},
			},
		}

		value, ok := claim.Get(c)
		if !ok {
			t.Fatal("expected Get to return true")
		}
		if value != "deep" {
			t.Errorf("expected 'deep', got '%s'", value)
		}
	})

	t.Run("Get missing path", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		c := cazi.Claims{"other": "value"}

		value, ok := cityClaim.Get(c)
		if ok {
			t.Fatal("expected Get to return false for missing path")
		}
		if value != "" {
			t.Errorf("expected zero value, got '%s'", value)
		}
	})

	t.Run("Get with non-map intermediate value", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		c := cazi.Claims{"address": "not a map"}

		value, ok := cityClaim.Get(c)
		if ok {
			t.Fatal("expected Get to return false when path contains non-map")
		}
		if value != "" {
			t.Errorf("expected zero value, got '%s'", value)
		}
	})

	t.Run("Set creates intermediate maps", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		c := make(cazi.Claims)

		cityClaim.Set(c, "Paris")

		// Verify structure was created
		address, ok := c["address"].(map[string]any)
		if !ok {
			t.Fatal("expected address to be a map")
		}
		if address["city"] != "Paris" {
			t.Errorf("expected 'Paris', got '%v'", address["city"])
		}
	})

	t.Run("Set overwrites existing value", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		c := cazi.Claims{
			"address": map[string]any{
				"city": "London",
			},
		}

		cityClaim.Set(c, "Tokyo")

		address := c["address"].(map[string]any)
		if address["city"] != "Tokyo" {
			t.Errorf("expected 'Tokyo', got '%v'", address["city"])
		}
	})

	t.Run("Set on nil claims does not panic", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		var c cazi.Claims

		// Should not panic
		cityClaim.Set(c, "Berlin")
	})

	t.Run("Set when path blocked by non-map", func(t *testing.T) {
		cityClaim := claims.Nested[string]("address", "city")
		c := cazi.Claims{"address": "not a map"}

		// Should not panic, just silently fail
		cityClaim.Set(c, "Berlin")

		// Original value unchanged
		if c["address"] != "not a map" {
			t.Errorf("expected original value to remain, got '%v'", c["address"])
		}
	})

	t.Run("Panics with empty path", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty path")
			}
		}()

		claims.Nested[string]()
	})
}

