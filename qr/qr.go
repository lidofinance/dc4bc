package qr

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"

	encoder "github.com/skip2/go-qrcode"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

const defaultChunkSize = 512

var palette = color.Palette{
	image.Transparent,
	image.Black,
	image.White,
	color.RGBA{G: 255, A: 255},
	color.RGBA{G: 100, A: 255},
}

type Processor interface {
	WriteQR(path string, data []byte) error
	SetDelay(delay int)
	SetChunkSize(chunkSize int)
}

type CameraProcessor struct {
	gifFramesDelay int
	chunkSize      int

	closeCameraReader chan bool
}

func NewCameraProcessor() Processor {
	return &CameraProcessor{
		closeCameraReader: make(chan bool),
		chunkSize:         defaultChunkSize,
	}
}

func (p *CameraProcessor) SetChunkSize(chunkSize int) {
	p.chunkSize = chunkSize
}

func (p *CameraProcessor) SetDelay(delay int) {
	p.gifFramesDelay = delay
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
