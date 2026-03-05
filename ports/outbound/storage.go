package outbound

import (
	"context"
	"io"
	"time"
)

// StoragePort defines the interface for object storage operations.
type StoragePort interface {
	// Upload stores an object and returns the storage key.
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Download retrieves an object by key.
	Download(ctx context.Context, key string) (io.ReadCloser, int64, error)

	// Delete removes an object.
	Delete(ctx context.Context, key string) error

	// Exists checks if an object exists.
	Exists(ctx context.Context, key string) (bool, error)

	// SignedURL generates a pre-signed URL for direct access (if supported).
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)

	// PublicURL returns the public URL for an object.
	PublicURL(key string) string

	// Copy duplicates an object within the same bucket.
	Copy(ctx context.Context, srcKey, dstKey string) error
}
