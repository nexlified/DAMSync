package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/ports/outbound"
	"github.com/nexlified/dam/domain"
	"golang.org/x/crypto/bcrypt"
)

type AuthServiceImpl struct {
	userRepo   outbound.UserRepository
	apiKeyRepo outbound.APIKeyRepository
	cache      outbound.CachePort
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewAuthService(
	userRepo outbound.UserRepository,
	apiKeyRepo outbound.APIKeyRepository,
	cache outbound.CachePort,
	accessSecret, refreshSecret string,
	accessTTL, refreshTTL time.Duration,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo:      userRepo,
		apiKeyRepo:    apiKeyRepo,
		cache:         cache,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (s *AuthServiceImpl) Login(ctx context.Context, orgSlug, email, password string) (*inbound.TokenPair, error) {
	// We need orgID to look up the user; find org by slug first
	// For now, pass orgID as a stand-in — callers will resolve org by slug externally
	// This is simplified; callers pass orgID via GetOrgBySlug first
	_ = orgSlug
	return nil, domain.ErrInvalidInput
}

// LoginWithOrgID performs login when orgID is already resolved.
func (s *AuthServiceImpl) LoginWithOrgID(ctx context.Context, orgID uuid.UUID, email, password string) (*inbound.TokenPair, error) {
	user, err := s.userRepo.GetByEmail(ctx, orgID, email)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if !user.Active {
		return nil, domain.NewError(domain.ErrUnauthorized, "account is inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	return s.generateTokenPair(user)
}

func (s *AuthServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*inbound.TokenPair, error) {
	claims, err := s.parseToken(refreshToken, s.refreshSecret)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Check if revoked in cache
	revokedKey := fmt.Sprintf("revoked_token:%s", claims.ID)
	revoked, _ := s.cache.Exists(ctx, revokedKey)
	if revoked {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return s.generateTokenPair(user)
}

func (s *AuthServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.parseToken(refreshToken, s.refreshSecret)
	if err != nil {
		return nil // already invalid, treat as success
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		key := fmt.Sprintf("revoked_token:%s", claims.ID)
		return s.cache.Set(ctx, key, "1", ttl)
	}
	return nil
}

func (s *AuthServiceImpl) ValidateAccessToken(ctx context.Context, tokenStr string) (*inbound.Claims, error) {
	claims, err := s.parseToken(tokenStr, s.accessSecret)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	return &inbound.Claims{
		UserID: claims.UserID,
		OrgID:  claims.OrgID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

func (s *AuthServiceImpl) ValidateAPIKey(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	if len(rawKey) < 8 {
		return nil, domain.ErrInvalidToken
	}
	prefix := rawKey[:8]
	key, err := s.apiKeyRepo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if key.IsRevoked() {
		return nil, domain.ErrInvalidToken
	}
	if err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(rawKey)); err != nil {
		return nil, domain.ErrInvalidToken
	}
	_ = s.apiKeyRepo.UpdateLastUsed(ctx, key.ID)
	return key, nil
}

func (s *AuthServiceImpl) CreateAPIKey(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID, req inbound.CreateAPIKeyRequest) (*domain.APIKey, string, error) {
	rawKey, err := generateSecureToken(32)
	if err != nil {
		return nil, "", err
	}
	prefix := rawKey[:8]
	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	key := &domain.APIKey{
		ID:          uuid.New(),
		OrgID:       orgID,
		UserID:      userID,
		Name:        req.Name,
		KeyPrefix:   prefix,
		KeyHash:     string(hash),
		Scopes:      req.Scopes,
		IPAllowlist: req.IPAllowlist,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.apiKeyRepo.Create(ctx, key); err != nil {
		return nil, "", err
	}
	return key, rawKey, nil
}

func (s *AuthServiceImpl) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]*domain.APIKey, error) {
	return s.apiKeyRepo.ListByOrg(ctx, orgID)
}

func (s *AuthServiceImpl) RevokeAPIKey(ctx context.Context, orgID, keyID uuid.UUID) error {
	return s.apiKeyRepo.Revoke(ctx, keyID)
}

// --- internal helpers ---

type damClaims struct {
	UserID uuid.UUID   `json:"uid"`
	OrgID  uuid.UUID   `json:"oid"`
	Email  string      `json:"email"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthServiceImpl) generateTokenPair(user *domain.User) (*inbound.TokenPair, error) {
	now := time.Now().UTC()
	accessExp := now.Add(s.accessTTL)

	accessClaims := damClaims{
		UserID: user.ID,
		OrgID:  user.OrgID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.accessSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := damClaims{
		UserID: user.ID,
		OrgID:  user.OrgID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &inbound.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
	}, nil
}

func (s *AuthServiceImpl) parseToken(tokenStr string, secret []byte) (*damClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &damClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*damClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}
	return claims, nil
}

func generateSecureToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
