package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type OrgServiceImpl struct {
	orgRepo  outbound.OrgRepository
	userRepo outbound.UserRepository
}

func NewOrgService(orgRepo outbound.OrgRepository, userRepo outbound.UserRepository) *OrgServiceImpl {
	return &OrgServiceImpl{orgRepo: orgRepo, userRepo: userRepo}
}

func (s *OrgServiceImpl) CreateOrg(ctx context.Context, req inbound.CreateOrgRequest) (*domain.Org, error) {
	org := &domain.Org{
		ID:                uuid.New(),
		Name:              req.Name,
		Slug:              req.Slug,
		Plan:              req.Plan,
		StorageQuotaBytes: 10 * 1024 * 1024 * 1024, // 10 GiB default
		Settings: domain.OrgSettings{
			StripEXIF:            true,
			WebhookRetentionDays: 30,
			SoftDeleteDays:       30,
			MaxFileSizeBytes:     100 * 1024 * 1024, // 100 MiB default
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrgServiceImpl) GetOrg(ctx context.Context, id uuid.UUID) (*domain.Org, error) {
	return s.orgRepo.GetByID(ctx, id)
}

func (s *OrgServiceImpl) GetOrgBySlug(ctx context.Context, slug string) (*domain.Org, error) {
	return s.orgRepo.GetBySlug(ctx, slug)
}

func (s *OrgServiceImpl) UpdateOrg(ctx context.Context, id uuid.UUID, req inbound.UpdateOrgRequest) (*domain.Org, error) {
	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Settings != nil {
		org.Settings = *req.Settings
	}
	org.UpdatedAt = time.Now().UTC()
	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrgServiceImpl) DeleteOrg(ctx context.Context, id uuid.UUID) error {
	return s.orgRepo.Delete(ctx, id)
}

func (s *OrgServiceImpl) CreateUser(ctx context.Context, orgID uuid.UUID, req inbound.CreateUserRequest) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		ID:           uuid.New(),
		OrgID:        orgID,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		Active:       true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *OrgServiceImpl) ListUsers(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error) {
	return s.userRepo.ListByOrg(ctx, orgID)
}

func (s *OrgServiceImpl) GetUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (s *OrgServiceImpl) UpdateUser(ctx context.Context, orgID, userID uuid.UUID, req inbound.UpdateUserRequest) (*domain.User, error) {
	user, err := s.GetUser(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Active != nil {
		user.Active = *req.Active
	}
	user.UpdatedAt = time.Now().UTC()
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *OrgServiceImpl) DeleteUser(ctx context.Context, orgID, userID uuid.UUID) error {
	user, err := s.GetUser(ctx, orgID, userID)
	if err != nil {
		return err
	}
	return s.userRepo.Delete(ctx, user.ID)
}

func (s *OrgServiceImpl) GetStorageUsage(ctx context.Context, orgID uuid.UUID) (*inbound.StorageUsage, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	pct := 0.0
	if org.StorageQuotaBytes > 0 {
		pct = float64(org.StorageUsedBytes) / float64(org.StorageQuotaBytes) * 100
	}
	return &inbound.StorageUsage{
		UsedBytes:  org.StorageUsedBytes,
		QuotaBytes: org.StorageQuotaBytes,
		Percent:    pct,
	}, nil
}
