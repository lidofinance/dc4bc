package qr

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	encoder "github.com/skip2/go-qrcode"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

func WriteQR(path string, data []byte) error {
	err := encoder.WriteFile(string(data), encoder.Medium, 512, path)
	if err != nil {
		return fmt.Errorf("failed to encode the data: %w", err)
	}

	return nil
}
