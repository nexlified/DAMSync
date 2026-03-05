package outbound

import (
	"context"

	"github.com/nexlified/dam/domain"
)

// TransformRequest specifies what transformation to apply.
type TransformRequest struct {
	Input   []byte
	Width   *int
	Height  *int
	Fit     domain.ResizeFit
	Crop    domain.CropPosition
	Quality *int
	Format  domain.OutputFormat
	FocalPoint *domain.FocalPoint
}

// TransformResult holds the output of a transformation.
type TransformResult struct {
	Data        []byte
	Format      string
	Width       int
	Height      int
	SizeBytes   int64
	ContentType string
}

// TransformerPort defines the interface for image transformation.
type TransformerPort interface {
	Transform(ctx context.Context, req *TransformRequest) (*TransformResult, error)

	// ExtractMetadata reads image dimensions, color profile etc.
	ExtractMetadata(data []byte) (*ImageMetadata, error)
}

type ImageMetadata struct {
	Width       int
	Height      int
	Format      string
	ColorSpace  string
	HasAlpha    bool
	Orientation int
}
