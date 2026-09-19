package vision

import (
	"bytes"
	"errors"
	"image"

	_ "image/jpeg"
	_ "image/png"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported image format: expected PNG or JPEG")
	ErrCorrupt           = errors.New("image could not be decoded")
	ErrTooLarge          = errors.New("image dimensions exceed the supported size")
)

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
