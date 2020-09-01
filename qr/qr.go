package qr

import (
	"fmt"
	"image"
	"log"
	"time"

	encoder "github.com/skip2/go-qrcode"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"gocv.io/x/gocv"
)

const (
	timeToScan = time.Second * 5
)

type Processor interface {
	ReadQR() ([]byte, error)
	WriteQR(path string, data []byte) error
}

type CameraProcessor struct{}

func NewCameraProcessor() *CameraProcessor {
	return &CameraProcessor{}
}

func (p *CameraProcessor) ReadQR() ([]byte, error) {
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return nil, fmt.Errorf("failed to OpenVideoCapture: %w", err)
	}
	window := gocv.NewWindow("Hello")

	defer func() {
		if err := webcam.Close(); err != nil {
			log.Fatalf("failed to close camera: %v", err)
		}
	}()
	defer func() {
		if err := window.Close(); err != nil {
			log.Fatalf("failed to close camera window: %v", err)
		}
	}()

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
		return nil, fmt.Errorf("failed to get image object: %w", err)
	}

	return ReadDataFromQR(imgObject)
}

func (p *CameraProcessor) WriteQR(path string, data []byte) error {
	err := encoder.WriteFile(string(data), encoder.Medium, 512, path)
	if err != nil {
		return fmt.Errorf("failed to encode the data: %w", err)
	}

	return nil
}

func ReadDataFromQR(img image.Image) ([]byte, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, fmt.Errorf("failed to get NewBinaryBitmapFromImage: %w", err)
	}

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the QR-code contents: %w", err)
	}
	return []byte(result.String()), nil
}

func EncodeQR(data []byte) ([]byte, error) {
	return encoder.Encode(string(data), encoder.Medium, 512)
}
