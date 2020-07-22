package qr

import (
	"fmt"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"gocv.io/x/gocv"
)

const timeToScan = time.Second * 5

func ReadQRFromCamera() (string, error) {
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return "", fmt.Errorf("failed to OpenVideoCapture: %w", err)
	}
	window := gocv.NewWindow("Hello")

	defer webcam.Close()
	defer window.Close()

	img := gocv.NewMat()
	tk := time.NewTimer(timeToScan)

	// This loop reads an image from the webcam every millisecond
	// for 5 seconds. The last image taken will be used as the final
	//one.
loop:
	for {
		select {
		case <-tk.C:
			break loop
		default:
			webcam.Read(&img)
			window.IMShow(img)
			window.WaitKey(1)
		}
	}

	imgObject, err := img.ToImage()
	if err != nil {
		return "", fmt.Errorf("failed to get image object: %w", err)
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(imgObject)
	if err != nil {
		return "", fmt.Errorf("failed to get NewBinaryBitmapFromImage: %w", err)
	}

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decode the QR-code contents: %w", err)
	}

	return result.String(), err
}
