package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/internal/application/services"
	"github.com/nexlified/dam/domain"
)

const (
	ContextKeyClaims = "claims"
	ContextKeyAPIKey = "api_key"
	ContextKeyOrgID  = "org_id"
)

// RequireAuth validates JWT Bearer token or API key header.
func RequireAuth(authSvc *services.AuthServiceImpl) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try Bearer JWT
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authSvc.ValidateAccessToken(c.Context(), token)
			if err != nil {
				return fiber.ErrUnauthorized
			}
			c.Locals(ContextKeyClaims, claims)
			c.Locals(ContextKeyOrgID, claims.OrgID)
			return c.Next()
		}

		// Try API key
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}
		if apiKey != "" {
			key, err := authSvc.ValidateAPIKey(c.Context(), apiKey)
			if err != nil {
				return fiber.ErrUnauthorized
			}
			c.Locals(ContextKeyAPIKey, key)
			c.Locals(ContextKeyOrgID, key.OrgID)
			return c.Next()
		}

		return fiber.ErrUnauthorized
	}
}

// RequireRole ensures the authenticated user has one of the given roles.
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims == nil {
			if c.Locals(ContextKeyAPIKey) != nil {
				return c.Next()
			}
			return fiber.ErrForbidden
		}
		for _, r := range roles {
			if string(claims.Role) == r {
				return c.Next()
			}
		}
		return fiber.ErrForbidden
	}
}

// RequireScope ensures the API key has the required scope.
// JWT users always pass.
func RequireScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals(ContextKeyClaims) != nil {
			return c.Next()
		}
		key, ok := c.Locals(ContextKeyAPIKey).(*domain.APIKey)
		if !ok {
			return fiber.ErrUnauthorized
		}
		if !key.HasScope(scope) {
			return fiber.NewError(fiber.StatusForbidden, "insufficient scope: "+scope)
		}
		return c.Next()
	}
}

// GetClaims extracts JWT claims from context.
func GetClaims(c *fiber.Ctx) *inbound.Claims {
	claims, _ := c.Locals(ContextKeyClaims).(*inbound.Claims)
	return claims
}

// GetAPIKey extracts API key from context.
func GetAPIKey(c *fiber.Ctx) *domain.APIKey {
	key, _ := c.Locals(ContextKeyAPIKey).(*domain.APIKey)
	return key
}
