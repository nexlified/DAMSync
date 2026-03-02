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
	ID          uuid.UUID  `db:"id"`
	OrgID       uuid.UUID  `db:"org_id"`
	Domain      string     `db:"domain"`
	IsPrimary   bool       `db:"is_primary"`
	VerifiedAt  *time.Time `db:"verified_at"`
	TLSStatus   TLSStatus  `db:"tls_status"`
	ChallengeToken string  `db:"challenge_token"` // CNAME/TXT challenge
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

func (d *DomainRecord) IsVerified() bool {
	return d.VerifiedAt != nil
}

func (d *DomainRecord) IsActive() bool {
	return d.IsVerified() && d.TLSStatus == TLSStatusActive
}
