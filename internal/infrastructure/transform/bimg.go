package transform

import (
	"context"
	"fmt"
	"strings"

	"github.com/h2non/bimg"
	"github.com/nexlified/dam/ports/outbound"
	"github.com/nexlified/dam/domain"
)

type BimgTransformer struct{}

func NewBimgTransformer() *BimgTransformer {
	return &BimgTransformer{}
}

func (t *BimgTransformer) Transform(ctx context.Context, req *outbound.TransformRequest) (*outbound.TransformResult, error) {
	if len(req.Input) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	opts := bimg.Options{
		StripMetadata: true,
		NoProfile:     true,
	}

	// Quality
	if req.Quality != nil {
		opts.Quality = *req.Quality
	} else {
		opts.Quality = 85
	}

	// Output format
	opts.Type = mapFormat(req.Format)

	// Dimensions
	if req.Width != nil {
		opts.Width = *req.Width
	}
	if req.Height != nil {
		opts.Height = *req.Height
	}

	// Resize mode
	switch req.Fit {
	case domain.FitFill:
		opts.Crop = true
		opts.Gravity = cropGravity(req.Crop, req.FocalPoint, req.Width, req.Height)
	case domain.FitFit:
		opts.Embed = false
		// bimg default keeps aspect ratio when only one dimension given
	case domain.FitWidth:
		opts.Height = 0
	case domain.FitHeight:
		opts.Width = 0
	}

	result, err := bimg.NewImage(req.Input).Process(opts)
	if err != nil {
		return nil, fmt.Errorf("bimg transform: %w", err)
	}

	size, err := bimg.NewImage(result).Size()
	if err != nil {
		size = bimg.ImageSize{}
	}

	format := string(req.Format)
	if format == "" {
		format = "jpeg"
	}

	return &outbound.TransformResult{
		Data:        result,
		Format:      format,
		Width:       size.Width,
		Height:      size.Height,
		SizeBytes:   int64(len(result)),
		ContentType: formatToMIME(format),
	}, nil
}

func (t *BimgTransformer) ExtractMetadata(data []byte) (*outbound.ImageMetadata, error) {
	img := bimg.NewImage(data)
	meta, err := img.Metadata()
	if err != nil {
		return nil, err
	}
	size, err := img.Size()
	if err != nil {
		return nil, err
	}
	return &outbound.ImageMetadata{
		Width:      size.Width,
		Height:     size.Height,
		Format:     strings.ToLower(meta.Type),
		ColorSpace: meta.Space,
		HasAlpha:   meta.Alpha,
	}, nil
}

func mapFormat(f domain.OutputFormat) bimg.ImageType {
	switch f {
	case domain.FormatJPEG:
		return bimg.JPEG
	case domain.FormatPNG:
		return bimg.PNG
	case domain.FormatWebP:
		return bimg.WEBP
	case domain.FormatAVIF:
		return bimg.AVIF
	default:
		return bimg.JPEG
	}
}

func cropGravity(crop domain.CropPosition, fp *domain.FocalPoint, w, h *int) bimg.Gravity {
	if fp != nil && crop == domain.CropFocalPoint {
		// bimg doesn't support arbitrary focal point natively — use Smart gravity as approximation
		return bimg.GravitySmart
	}
	switch crop {
	case domain.CropTop:
		return bimg.GravityNorth
	case domain.CropBottom:
		return bimg.GravitySouth
	case domain.CropLeft:
		return bimg.GravityWest
	case domain.CropRight:
		return bimg.GravityEast
	default:
		return bimg.GravityCentre
	}
}

func formatToMIME(format string) string {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	default:
		return "application/octet-stream"
	}
}
