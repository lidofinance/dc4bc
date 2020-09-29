package qr

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
)

type chunk struct {
	Data  []byte
	Index uint
	Total uint
}

// DataToChunks divides a data on chunks with a size chunkSize
func DataToChunks(data []byte, chunkSize int) ([][]byte, error) {
	chunksCount := int(math.Ceil(float64(len(data)) / float64(chunkSize)))
	chunks := make([][]byte, 0, chunksCount)

	index := uint(0)
	for offset := 0; offset < len(data); offset += chunkSize {
		offsetEnd := offset + chunkSize
		if offsetEnd > len(data) {
			offsetEnd = len(data)
		}
		chunkBz, err := encodeChunk(chunk{
			Data:  data[offset:offsetEnd],
			Total: uint(chunksCount),
			Index: index,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode chunk: %w", err)
		}
		chunks = append(chunks, chunkBz)
		index++
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
