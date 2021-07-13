package qr

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"os"

	encoder "github.com/skip2/go-qrcode"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

const (
	defaultChunkSize       = 512
	defaultQrRecoveryLevel = encoder.Medium
	defaultFramesDelay     = 10
	endFramesDelay         = 200 // Number of frames to show after the last frame.
)

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
	ReadQR(filename string) ([]byte, error)
	SetRecoveryLevel(recoveryLevel encoder.RecoveryLevel)
}

type CameraProcessor struct {
	gifFramesDelay int
	chunkSize      int

	closeCameraReader chan bool
	qrRecoveryLevel   encoder.RecoveryLevel
}

func NewCameraProcessor() Processor {
	return &CameraProcessor{
		closeCameraReader: make(chan bool),
		chunkSize:         defaultChunkSize,
		qrRecoveryLevel:   defaultQrRecoveryLevel,
		gifFramesDelay:    defaultFramesDelay,
	}
}

func (p *CameraProcessor) SetChunkSize(chunkSize int) {
	p.chunkSize = chunkSize
}

func (p *CameraProcessor) SetDelay(delay int) {
	p.gifFramesDelay = delay
}

func (p *CameraProcessor) SetRecoveryLevel(recoveryLevel encoder.RecoveryLevel) {
	p.qrRecoveryLevel = recoveryLevel
}

func (p *CameraProcessor) WriteQR(path string, data []byte) error {
	chunks, err := DataToChunks(data, p.chunkSize)
	if err != nil {
		return fmt.Errorf("failed to divide data on chunks: %w", err)
	}
	outGif := &gif.GIF{}

	lastChunkIdx := len(chunks) - 1

	totalLen := 0
	for idx, c := range chunks {
		code, err := encoder.New(string(c), encoder.Medium)

		if err != nil {
			return fmt.Errorf("failed to create a QR code: %w", err)
		}
		frame := code.Image(512)
		bounds := frame.Bounds()
		palettedImage := image.NewPaletted(bounds, palette)
		draw.Draw(palettedImage, palettedImage.Rect, frame, bounds.Min, draw.Src)

		outGif.Image = append(outGif.Image, palettedImage)
		if idx < lastChunkIdx {
			outGif.Delay = append(outGif.Delay, p.gifFramesDelay)
		} else {
			outGif.Delay = append(outGif.Delay, endFramesDelay)
		}
		totalLen += len(c)
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
func (p *CameraProcessor) ReadQR(filename string) ([]byte, error) {
	if _, err := os.Stat(filename); err != nil {
		return nil, fmt.Errorf("cannot open qr file \"%s\"", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read qr file \"%s\"", err)
	}
	defer file.Close()

	decodedGIF, err := gif.DecodeAll(file)
	if err != nil {
		return nil, fmt.Errorf("cannot decode qr file \"%s\"", err)
	}

	chunks := Chunks{}
	decodedChunksCount := uint32(0)
	for idx, frame := range decodedGIF.Image {
		data, err := ReadDataFromQR(frame)
		if err != nil {
			return nil, fmt.Errorf("cannot read frame %d", idx)
		}
		chunk := &Chunk{}
		err = chunk.UnmarshalBinary(data)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal data from frame %d", idx)
		}
		chunks = append(chunks, chunk)
		decodedChunksCount++
		if decodedChunksCount == chunk.Header.Total {
			break
		}
	}
	data := make([]byte, 0)
	for _, c := range chunks {
		data = append(data, c.Data...)
	}

	buf := bytes.Buffer{}
	bufWriter := bufio.NewWriter(&buf)

	zr, err := gzip.NewReader(bytes.NewBuffer(data))

	if err != nil {
		return nil, fmt.Errorf("cannot create compression reader: %s", err)
	}

	defer zr.Close()

	if err != nil {
		return nil, fmt.Errorf("cannot read compression data \"%s\"", err)
	}

	_, err = io.Copy(bufWriter, zr)
	if err != nil {
		return nil, fmt.Errorf("cannot copy compression data \"%s\"", err)
	}

	if err := zr.Close(); err != nil {
		return nil, fmt.Errorf("cannot finalize readed data \"%s\"", err)
	}

	if err = os.Remove(filename); err != nil {
		return nil, fmt.Errorf("cannot remove qr file \"%s\"", err)
	}

	return buf.Bytes(), nil
}
