package qr

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log"
	"os"

	"gocv.io/x/gocv"

	encoder "github.com/skip2/go-qrcode"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

var palette = color.Palette{
	image.Transparent,
	image.Black,
	image.White,
	color.RGBA{G: 255, A: 255},
	color.RGBA{G: 100, A: 255},
}

type Processor interface {
	ReadQR() ([]byte, error)
	WriteQR(path string, data []byte) error
	SetDelay(delay int)
	SetChunkSize(chunkSize int)
	CloseCameraReader()
}

type CameraProcessor struct {
	gifFramesDelay int
	chunkSize      int

	closeCameraReader chan bool
}

func NewCameraProcessor() Processor {
	return &CameraProcessor{
		closeCameraReader: make(chan bool),
	}
}

func (p *CameraProcessor) CloseCameraReader() {
	p.closeCameraReader <- true
}

func (p *CameraProcessor) SetDelay(delay int) {
	p.gifFramesDelay = delay
}

func (p *CameraProcessor) SetChunkSize(chunkSize int) {
	p.chunkSize = chunkSize
}

func (p *CameraProcessor) ReadQR() ([]byte, error) {
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return nil, fmt.Errorf("failed to OpenVideoCapture: %w", err)
	}
	window := gocv.NewWindow("Please, show a gif with QR codes")

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
	defer img.Close()

	chunks := make([]*chunk, 0)
	decodedChunksCount := uint(0)
	// detects and scans QR-cods from camera until we scan successfully
READER:
	for {
		select {
		case <-p.closeCameraReader:
			return nil, fmt.Errorf("camera reader was closed")
		default:
			webcam.Read(&img)
			window.IMShow(img)
			window.WaitKey(1)

			imgObject, err := img.ToImage()
			if err != nil {
				return nil, fmt.Errorf("failed to get image object: %w", err)
			}
			data, err := ReadDataFromQR(imgObject)
			if err != nil {
				continue
			}
			decodedChunk, err := decodeChunk(data)
			if err != nil {
				return nil, err
			}
			if cap(chunks) == 0 {
				chunks = make([]*chunk, decodedChunk.Total)
			}
			if decodedChunk.Index > decodedChunk.Total {
				return nil, fmt.Errorf("invalid QR-code chunk")
			}
			if chunks[decodedChunk.Index] != nil {
				continue
			}
			chunks[decodedChunk.Index] = decodedChunk
			decodedChunksCount++
			window.SetWindowTitle(fmt.Sprintf("Read %d/%d chunks", decodedChunksCount, decodedChunk.Total))
			if decodedChunksCount == decodedChunk.Total {
				break READER
			}
		}
	}
	window.SetWindowTitle("QR-code chunks successfully read!")
	data := make([]byte, 0)
	for _, c := range chunks {
		data = append(data, c.Data...)
	}
	return data, nil
}

func (p *CameraProcessor) WriteQR(path string, data []byte) error {
	chunks, err := DataToChunks(data, p.chunkSize)
	if err != nil {
		return fmt.Errorf("failed to divide data on chunks: %w", err)
	}
	outGif := &gif.GIF{}
	for _, c := range chunks {
		code, err := encoder.New(string(c), encoder.Medium)
		if err != nil {
			return fmt.Errorf("failed to create a QR code: %w", err)
		}
		frame := code.Image(512)
		bounds := frame.Bounds()
		palettedImage := image.NewPaletted(bounds, palette)
		draw.Draw(palettedImage, palettedImage.Rect, frame, bounds.Min, draw.Src)

		outGif.Image = append(outGif.Image, palettedImage)
		outGif.Delay = append(outGif.Delay, p.gifFramesDelay)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, outGif); err != nil {
		return fmt.Errorf("failed to encode qr gif: %w", err)
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
