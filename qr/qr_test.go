package qr

import (
	"fmt"
	encoder "github.com/skip2/go-qrcode"
	"image"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

type TestQrProcessor struct {
	qrs []string
}

func NewTestQRProcessor() *TestQrProcessor {
	return &TestQrProcessor{}
}

func (p *TestQrProcessor) ReadQR() ([]byte, error) {
	if len(p.qrs) == 0 {
		return nil, fmt.Errorf("qr not found")
	}
	qr := p.qrs[0]
	file, err := os.Open(qr)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	data, err := ReadDataFromQR(img)
	if err != nil {
		return nil, err
	}

	p.qrs = p.qrs[1:]
	if err = os.Remove(qr); err != nil {
		return nil, err
	}
	return data, nil
}

func (p *TestQrProcessor) WriteQR(path string, data []byte) error {
	err := encoder.WriteFile(string(data), encoder.Medium, 512, path)
	if err != nil {
		return fmt.Errorf("failed to encode the data: %w", err)
	}

	p.qrs = append(p.qrs, path)
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

	chunks, err := DataToChunks(data)
	if err != nil {
		t.Fatalf(err.Error())
	}

	p := NewTestQRProcessor()

	for idx, chunk := range chunks {
		path := fmt.Sprintf("/tmp/%d.png", idx)
		if err = p.WriteQR(path, chunk); err != nil {
			t.Fatalf(err.Error())
		}
	}

	recoveredDataFromQRChunks, err := ReadDataFromQRChunks(p)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !reflect.DeepEqual(data, recoveredDataFromQRChunks) {
		t.Fatal("recovered data from chunks and initial data are not equal!")
	}
}
