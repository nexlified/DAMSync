package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
)

// NewAuditLogger logs write operations to the audit log table.
func NewAuditLogger(auditRepo outbound.AuditLogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}
		// Log only write operations that succeeded
		method := string(c.Method())
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return nil
		}
		if c.Response().StatusCode() >= 400 {
			return nil
		}

		claims := GetClaims(c)
		var orgID uuid.UUID
		var userID *uuid.UUID

		if claims != nil {
			orgID = claims.OrgID
			userID = &claims.UserID
		} else if oid, ok := c.Locals(ContextKeyOrgID).(uuid.UUID); ok {
			orgID = oid
		}

		if orgID == uuid.Nil {
			return nil
		}

		resourceID := c.Params("id")
		log := &domain.AuditLog{
			ID:           uuid.New(),
			OrgID:        orgID,
			UserID:       userID,
			Action:       method + " " + c.Path(),
			ResourceType: resourceTypeFromPath(c.Path()),
			IP:           c.IP(),
			CreatedAt:    time.Now().UTC(),
		}
		if resourceID != "" {
			log.ResourceID = &resourceID
		}

		// Fire and forget — don't block the response
		go func() {
			_ = auditRepo.Create(c.Context(), log)
		}()

		return nil
	}
}

func resourceTypeFromPath(path string) string {
	segments := splitPath(path)
	for i, s := range segments {
		if s == "api" || s == "v1" {
			if i+1 < len(segments) {
				return segments[i+1]
			}
		}
	}
	return "unknown"
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
