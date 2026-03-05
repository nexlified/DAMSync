package domain

import "errors"

// Sentinel errors for domain-level conditions.
var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrQuotaExceeded     = errors.New("storage quota exceeded")
	ErrInvalidMIMEType   = errors.New("invalid or disallowed MIME type")
	ErrFileTooLarge      = errors.New("file too large")
	ErrExpiredToken      = errors.New("token expired")
	ErrInvalidToken      = errors.New("invalid token")
	ErrDomainNotVerified = errors.New("domain not verified")
	ErrSignedURLExpired  = errors.New("signed URL expired")
	ErrSignedURLInvalid  = errors.New("signed URL invalid")
)

// DomainError wraps a sentinel with optional context.
type DomainError struct {
	Err     error
	Message string
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewError(sentinel error, msg string) *DomainError {
	return &DomainError{Err: sentinel, Message: msg}
}

func IsNotFound(err error) bool       { return errors.Is(err, ErrNotFound) }
func IsUnauthorized(err error) bool   { return errors.Is(err, ErrUnauthorized) }
func IsForbidden(err error) bool      { return errors.Is(err, ErrForbidden) }
func IsInvalidInput(err error) bool   { return errors.Is(err, ErrInvalidInput) }
func IsQuotaExceeded(err error) bool  { return errors.Is(err, ErrQuotaExceeded) }
func IsAlreadyExists(err error) bool  { return errors.Is(err, ErrAlreadyExists) }
