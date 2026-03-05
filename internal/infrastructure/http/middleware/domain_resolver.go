package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/ports/outbound"
	"github.com/nexlified/dam/domain"
)

const ContextKeyOrgFromDomain = "org_from_domain"

// NewDomainResolver maps the request Host header to an org and sets it in locals.
func NewDomainResolver(domainSvc inbound.DomainService, cache outbound.CachePort) fiber.Handler {
	return func(c *fiber.Ctx) error {
		host := c.Hostname()
		if host == "" {
			return c.Next()
		}

		org, err := domainSvc.ResolveOrgByDomain(c.Context(), host)
		if err == nil && org != nil {
			c.Locals(ContextKeyOrgFromDomain, org)
			if c.Locals(ContextKeyOrgID) == nil {
				c.Locals(ContextKeyOrgID, org.ID)
			}
		}
		return c.Next()
	}
}

// GetOrgFromDomain returns the org resolved from the request domain.
func GetOrgFromDomain(c *fiber.Ctx) *domain.Org {
	org, _ := c.Locals(ContextKeyOrgFromDomain).(*domain.Org)
	return org
}
