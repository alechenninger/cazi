package claims_test

import (
	"testing"

	"github.com/alechenninger/cazi/pkg/cazi"
	"github.com/alechenninger/cazi/pkg/claims"
)

func TestStandardClaims(t *testing.T) {
	t.Run("Sub claim", func(t *testing.T) {
		c := make(cazi.Claims)
		claims.Sub.Set(c, "user-123")

		value, ok := claims.Sub.Get(c)
		if !ok {
			t.Fatal("expected Sub.Get to return true")
		}
		if value != "user-123" {
			t.Errorf("expected 'user-123', got '%s'", value)
		}
	})

	t.Run("PreferredUsername claim", func(t *testing.T) {
		c := make(cazi.Claims)
		claims.PreferredUsername.Set(c, "alice")

		value, ok := claims.PreferredUsername.Get(c)
		if !ok {
			t.Fatal("expected PreferredUsername.Get to return true")
		}
		if value != "alice" {
			t.Errorf("expected 'alice', got '%s'", value)
		}
	})

	t.Run("Email claim", func(t *testing.T) {
		c := make(cazi.Claims)
		claims.Email.Set(c, "user@example.com")

		value, ok := claims.Email.Get(c)
		if !ok {
			t.Fatal("expected Email.Get to return true")
		}
		if value != "user@example.com" {
			t.Errorf("expected 'user@example.com', got '%s'", value)
		}
	})

	t.Run("Roles claim", func(t *testing.T) {
		c := make(cazi.Claims)
		roles := []string{"admin", "editor"}
		claims.Roles.Set(c, roles)

		value, ok := claims.Roles.Get(c)
		if !ok {
			t.Fatal("expected Roles.Get to return true")
		}
		if len(value) != 2 || value[0] != "admin" || value[1] != "editor" {
			t.Errorf("expected ['admin', 'editor'], got %v", value)
		}
	})

	t.Run("Groups claim", func(t *testing.T) {
		c := make(cazi.Claims)
		groups := []string{"developers", "ops"}
		claims.Groups.Set(c, groups)

		value, ok := claims.Groups.Get(c)
		if !ok {
			t.Fatal("expected Groups.Get to return true")
		}
		if len(value) != 2 || value[0] != "developers" || value[1] != "ops" {
			t.Errorf("expected ['developers', 'ops'], got %v", value)
		}
	})

	t.Run("Multiple claims together", func(t *testing.T) {
		c := make(cazi.Claims)

		claims.Sub.Set(c, "user-456")
		claims.Email.Set(c, "test@example.com")
		claims.Roles.Set(c, []string{"viewer"})

		sub, ok := claims.Sub.Get(c)
		if !ok || sub != "user-456" {
			t.Errorf("expected 'user-456', got '%s' (ok=%v)", sub, ok)
		}

		email, ok := claims.Email.Get(c)
		if !ok || email != "test@example.com" {
			t.Errorf("expected 'test@example.com', got '%s' (ok=%v)", email, ok)
		}

		roles, ok := claims.Roles.Get(c)
		if !ok || len(roles) != 1 || roles[0] != "viewer" {
			t.Errorf("expected ['viewer'], got %v (ok=%v)", roles, ok)
		}
	})

	t.Run("Claims are independent", func(t *testing.T) {
		c1 := make(cazi.Claims)
		c2 := make(cazi.Claims)

		claims.Sub.Set(c1, "user-1")
		claims.Sub.Set(c2, "user-2")

		val1, _ := claims.Sub.Get(c1)
		val2, _ := claims.Sub.Get(c2)

		if val1 == val2 {
			t.Error("expected claims to be independent")
		}
	})
}

