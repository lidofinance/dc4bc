package qr

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
)

const chunkSize = 256

type chunk struct {
	Data    []byte
	IsFinal bool
}

func ReadDataFromQRChunks(p Processor) ([]byte, error) {
	var (
		fullData, chunkBz []byte
		err               error
	)
	for {
		chunkBz, err = p.ReadQR()
		if err != nil {
			return nil, err
		}
		chunk, err := decodeChunk(chunkBz)
		if err != nil {
			return nil, fmt.Errorf("failed to decode chunk: %w", err)
		}
		fullData = append(fullData, chunk.Data...)
		if chunk.IsFinal {
			return fullData, nil
		}
	}
}

func DataToChunks(data []byte) ([][]byte, error) {
	chunksCount := int(math.Ceil(float64(len(data)) / chunkSize))
	chunks := make([][]byte, 0, chunksCount)

	for offset := 0; offset < len(data); offset += chunkSize {
		offsetEnd := offset + chunkSize
		if offsetEnd > len(data) {
			offsetEnd = len(data)
		}
		isFinal := offsetEnd == len(data)
		chunkBz, err := encodeChunk(chunk{
			Data:    data[offset:offsetEnd],
			IsFinal: isFinal,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode chunk: %w", err)
		}
		chunks = append(chunks, chunkBz)
	}
	return chunks, nil
}

func decodeChunk(data []byte) (*chunk, error) {
	var (
		c   chunk
		err error
	)
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	if err = dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("failed to decode chunk: %w", err)
	}
	return &c, nil
}

func encodeChunk(c chunk) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(c); err != nil {
		return nil, fmt.Errorf("failed to encode chunk: %w", err)
	}
	return buf.Bytes(), nil
}
