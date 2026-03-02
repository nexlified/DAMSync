package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
)

type DomainServiceImpl struct {
	domainRepo outbound.DomainRepository
	orgRepo    outbound.OrgRepository
	cache      outbound.CachePort
}

func NewDomainService(
	domainRepo outbound.DomainRepository,
	orgRepo outbound.OrgRepository,
	cache outbound.CachePort,
) *DomainServiceImpl {
	return &DomainServiceImpl{
		domainRepo: domainRepo,
		orgRepo:    orgRepo,
		cache:      cache,
	}
}

func (s *DomainServiceImpl) AddDomain(ctx context.Context, orgID uuid.UUID, domainName string) (*domain.DomainRecord, error) {
	// Check for duplicate
	existing, _ := s.domainRepo.GetByDomain(ctx, domainName)
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	token, err := generateChallengeToken()
	if err != nil {
		return nil, err
	}

	dr := &domain.DomainRecord{
		ID:             uuid.New(),
		OrgID:          orgID,
		Domain:         domainName,
		IsPrimary:      false,
		TLSStatus:      domain.TLSStatusPending,
		ChallengeToken: token,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := s.domainRepo.Create(ctx, dr); err != nil {
		return nil, err
	}
	return dr, nil
}

func (s *DomainServiceImpl) ListDomains(ctx context.Context, orgID uuid.UUID) ([]*domain.DomainRecord, error) {
	return s.domainRepo.ListByOrg(ctx, orgID)
}

func (s *DomainServiceImpl) InitiateVerification(ctx context.Context, orgID, domainID uuid.UUID) (*domain.DomainRecord, error) {
	dr, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if dr.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	// Return domain record with challenge token for CNAME verification
	return dr, nil
}

func (s *DomainServiceImpl) VerifyDomain(ctx context.Context, orgID, domainID uuid.UUID) (*domain.DomainRecord, error) {
	dr, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if dr.OrgID != orgID {
		return nil, domain.ErrNotFound
	}

	// Attempt DNS verification: look for TXT record "dam-verify=<token>"
	if err := verifyDNSChallenge(dr.Domain, dr.ChallengeToken); err != nil {
		return nil, domain.NewError(domain.ErrDomainNotVerified, fmt.Sprintf("DNS verification failed: %s", err.Error()))
	}

	now := time.Now().UTC()
	dr.VerifiedAt = &now
	dr.TLSStatus = domain.TLSStatusActive
	dr.UpdatedAt = now

	if err := s.domainRepo.Update(ctx, dr); err != nil {
		return nil, err
	}

	// Cache domain → org mapping
	cacheKey := fmt.Sprintf("domain_org:%s", dr.Domain)
	_ = s.cache.Set(ctx, cacheKey, dr.OrgID.String(), 24*time.Hour)

	return dr, nil
}

func (s *DomainServiceImpl) RemoveDomain(ctx context.Context, orgID, domainID uuid.UUID) error {
	dr, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return err
	}
	if dr.OrgID != orgID {
		return domain.ErrNotFound
	}
	// Remove from cache
	cacheKey := fmt.Sprintf("domain_org:%s", dr.Domain)
	_ = s.cache.Del(ctx, cacheKey)
	return s.domainRepo.Delete(ctx, domainID)
}

func (s *DomainServiceImpl) ResolveOrgByDomain(ctx context.Context, domainName string) (*domain.Org, error) {
	// Fast path: Redis cache
	cacheKey := fmt.Sprintf("domain_org:%s", domainName)
	if orgIDStr, err := s.cache.Get(ctx, cacheKey); err == nil && orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err == nil {
			return s.orgRepo.GetByID(ctx, orgID)
		}
	}

	// Slow path: DB lookup
	dr, err := s.domainRepo.GetByDomain(ctx, domainName)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if !dr.IsVerified() {
		return nil, domain.ErrDomainNotVerified
	}

	org, err := s.orgRepo.GetByID(ctx, dr.OrgID)
	if err != nil {
		return nil, err
	}

	// Warm cache
	_ = s.cache.Set(ctx, cacheKey, dr.OrgID.String(), 24*time.Hour)
	return org, nil
}

// --- internal ---

func generateChallengeToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func verifyDNSChallenge(domainName, token string) error {
	txtRecords, err := net.LookupTXT(domainName)
	if err != nil {
		return err
	}
	expected := fmt.Sprintf("dam-verify=%s", token)
	for _, record := range txtRecords {
		if record == expected {
			return nil
		}
	}
	return fmt.Errorf("TXT record '%s' not found for domain %s", expected, domainName)
}
