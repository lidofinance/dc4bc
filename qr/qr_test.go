package qr

import (
	"fmt"
	encoder "github.com/skip2/go-qrcode"
	"image"
	"image/draw"
	"image/gif"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

type TestQrProcessor struct {
	qr        string
	chunkSize int
}

func NewTestQRProcessor() *TestQrProcessor {
	return &TestQrProcessor{}
}

func (p *TestQrProcessor) ReadQR() ([]byte, error) {
	file, err := os.Open(p.qr)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decodedGIF, err := gif.DecodeAll(file)
	if err != nil {
		return nil, err
	}

	chunks := make([]*chunk, 0)
	decodedChunksCount := uint(0)
	for _, frame := range decodedGIF.Image {
		data, err := ReadDataFromQR(frame)
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
		if chunks[decodedChunk.Index] != nil {
			continue
		}
		chunks[decodedChunk.Index] = decodedChunk
		decodedChunksCount++
		if decodedChunksCount == decodedChunk.Total {
			break
		}
	}
	data := make([]byte, 0)
	for _, c := range chunks {
		data = append(data, c.Data...)
	}
	if err = os.Remove(p.qr); err != nil {
		return nil, err
	}
	return data, nil
}

func (p *TestQrProcessor) WriteQR(path string, data []byte) error {
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
		outGif.Delay = append(outGif.Delay, 10)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, outGif); err != nil {
		return fmt.Errorf("failed to encode qr gif: %w", err)
	}
	p.qr = path
	return nil
}

func genBytes(n int) []byte {
	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		return nil
	}
	return data
}

func TestReadDataFromQRChunks(t *testing.T) {
	N := 5000

	data := genBytes(N)

	p := NewTestQRProcessor()
	p.chunkSize = 128

	if err := p.WriteQR("/tmp/test_gif.gif", data); err != nil {
		t.Fatalf(err.Error())
	}

	recoveredDataFromQRChunks, err := p.ReadQR()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !reflect.DeepEqual(data, recoveredDataFromQRChunks) {
		t.Fatal("recovered data from chunks and initial data are not equal!")
	}
}
