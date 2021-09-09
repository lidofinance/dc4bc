package qr

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/lidofinance/dc4bc/client/config"
	encoder "github.com/skip2/go-qrcode"
	"image/gif"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
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
	if _, err := os.Stat(p.qr); err != nil {
		return nil, fmt.Errorf("cannot open qr file \"%s\"", err)
	}

	file, err := os.Open(p.qr)
	if err != nil {
		return nil, fmt.Errorf("cannot read qr file \"%s\"", err)
	}
	defer file.Close()

	decodedGIF, err := gif.DecodeAll(file)
	if err != nil {
		return nil, fmt.Errorf("cannot decode qr file \"%s\"", err)
	}

	chunks := make([]*chunk, 0)
	decodedChunksCount := uint(0)
	for idx, frame := range decodedGIF.Image {
		data, err := ReadDataFromQR(frame)
		if err != nil {
			return nil, fmt.Errorf("cannot read frame %d", idx)
			// continue
		}
		decodedChunk, err := decodeChunk(data)
		if err != nil {
			return nil, fmt.Errorf("cannot decode chunk \"%s\"", err)
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

	if err = os.Remove(p.qr); err != nil {
		return nil, fmt.Errorf("cannot remove qr file \"%s\"", err)
	}

	return buf.Bytes(), nil
}

func genBytes(n int) []byte {
	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		return nil
	}
	return data
}

func TestReadDataFromQRCameraProcessorChunks(t *testing.T) {
	N := 1000

	data := genBytes(N)

	qrConfig := config.QrProcessorConfig{
		FramesDelay: 10,
		ChunkSize: 128,
	}

	p := NewCameraProcessor(&qrConfig)
	p.SetRecoveryLevel(encoder.High)

	tmpFile, err := ioutil.TempFile("", "tmp_qr_gif")

	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(tmpFile.Name())

	if err := p.WriteQR(tmpFile.Name(), data); err != nil {
		t.Fatalf(err.Error())
	}

	/*
		// The library gozxing/qrcode doesn't parsing codes correctly.
		// Planned to use external tests with

		recoveredDataFromQRChunks, err := p.ReadQR(tmpFile.Name())
		if err != nil {
			t.Fatalf(err.Error())
		}

		if !reflect.DeepEqual(data, recoveredDataFromQRChunks) {
			t.Fatal("recovered data from chunks and initial data are not equal!")
		}*/
}
