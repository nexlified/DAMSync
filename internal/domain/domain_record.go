package domain

import (
	"time"

	"github.com/google/uuid"
)

type TLSStatus string

const (
	TLSStatusPending  TLSStatus = "pending"
	TLSStatusActive   TLSStatus = "active"
	TLSStatusFailed   TLSStatus = "failed"
	TLSStatusDisabled TLSStatus = "disabled"
)

type DomainRecord struct {
	ID             uuid.UUID  `db:"id"              json:"id"`
	OrgID          uuid.UUID  `db:"org_id"          json:"org_id"`
	Domain         string     `db:"domain"          json:"domain"`
	IsPrimary      bool       `db:"is_primary"      json:"is_primary"`
	VerifiedAt     *time.Time `db:"verified_at"     json:"verified_at"`
	TLSStatus      TLSStatus  `db:"tls_status"      json:"tls_status"`
	ChallengeToken string     `db:"challenge_token" json:"challenge_token"` // CNAME/TXT challenge
	CreatedAt      time.Time  `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"      json:"updated_at"`
}

func (d *DomainRecord) IsVerified() bool {
	return d.VerifiedAt != nil
}

func (d *DomainRecord) IsActive() bool {
	return d.IsVerified() && d.TLSStatus == TLSStatusActive
}
