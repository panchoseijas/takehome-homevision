package vision

import (
	"bytes"
	"errors"
	"image"

	// Register the decoders that ValidateImage accepts.
	_ "image/jpeg"
	_ "image/png"
)

var (
	// ErrUnsupportedFormat is returned when the bytes are not PNG or JPEG.
	ErrUnsupportedFormat = errors.New("unsupported image format: expected PNG or JPEG")
	// ErrCorrupt is returned when the bytes claim a supported format but cannot be decoded.
	ErrCorrupt = errors.New("image could not be decoded")
	// ErrTooLarge is returned when the decoded image would exceed the pixel budget.
	ErrTooLarge = errors.New("image dimensions exceed the supported size")
)

// ValidateImage inspects only the image header. It confirms the format is PNG
// or JPEG and that width*height stays within maxPixels, without decoding
// pixels, so callers can reject bad uploads before paying for a full decode.
func ValidateImage(data []byte, maxPixels int) (image.Config, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		if errors.Is(err, image.ErrFormat) {
			return image.Config{}, ErrUnsupportedFormat
		}
		return image.Config{}, ErrCorrupt
	}
	if format != "png" && format != "jpeg" {
		return image.Config{}, ErrUnsupportedFormat
	}
	if config.Width <= 0 || config.Height <= 0 {
		return image.Config{}, ErrCorrupt
	}
	if config.Width*config.Height > maxPixels {
		return image.Config{}, ErrTooLarge
	}
	return config, nil
}
