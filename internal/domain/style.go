package domain

import (
	"time"

	"github.com/google/uuid"
)

type OutputFormat string

const (
	FormatJPEG OutputFormat = "jpeg"
	FormatPNG  OutputFormat = "png"
	FormatWebP OutputFormat = "webp"
	FormatAVIF OutputFormat = "avif"
)

type ResizeFit string

const (
	FitFit    ResizeFit = "fit"    // preserve aspect ratio, fit within bounds
	FitFill   ResizeFit = "fill"   // crop to exact dimensions
	FitWidth  ResizeFit = "width"  // resize by width only
	FitHeight ResizeFit = "height" // resize by height only
)

type CropPosition string

const (
	CropCenter     CropPosition = "center"
	CropFocalPoint CropPosition = "focal"
	CropTop        CropPosition = "top"
	CropBottom     CropPosition = "bottom"
	CropLeft       CropPosition = "left"
	CropRight      CropPosition = "right"
)

type StyleOperation struct {
	Width    *int         `json:"width,omitempty"`
	Height   *int         `json:"height,omitempty"`
	Fit      ResizeFit    `json:"fit,omitempty"`
	Crop     CropPosition `json:"crop,omitempty"`
	Quality  *int         `json:"quality,omitempty"`
	Format   OutputFormat `json:"format,omitempty"`
	StripEXIF bool        `json:"strip_exif,omitempty"`
}

type ImageStyle struct {
	ID           uuid.UUID        `db:"id"`
	OrgID        uuid.UUID        `db:"org_id"`
	Name         string           `db:"name"`
	Slug         string           `db:"slug"`
	Operations   []StyleOperation `db:"operations"`
	OutputFormat OutputFormat     `db:"output_format"`
	Quality      int              `db:"quality"`
	CreatedAt    time.Time        `db:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at"`
}

type TransformCache struct {
	ID         uuid.UUID `db:"id"`
	AssetID    uuid.UUID `db:"asset_id"`
	StyleID    *uuid.UUID `db:"style_id"`
	ParamsHash string    `db:"params_hash"` // for ad-hoc transforms
	StorageKey string    `db:"storage_key"`
	SizeBytes  int64     `db:"size_bytes"`
	Format     string    `db:"format"`
	CreatedAt  time.Time `db:"created_at"`
}

// AdHocParams represents ad-hoc transform parameters (for trusted API callers).
type AdHocParams struct {
	Width   *int
	Height  *int
	Fit     ResizeFit
	Format  OutputFormat
	Quality *int
}
