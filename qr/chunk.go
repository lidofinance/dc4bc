package qr

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
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
	var buf bytes.Buffer

	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)

	if err != nil {
		return nil, fmt.Errorf("cannot create compression writer: %s", err)
	}

	_, err = zw.Write(data)

	if err != nil {
		return nil, fmt.Errorf("cannot write compressed data: %s", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("cannot finalize compressed data: %s", err)
	}

	chunksCount := int(math.Ceil(float64(buf.Len()) / float64(chunkSize)))
	chunks := make([][]byte, 0, chunksCount)

	index := uint(0)
	for offset := 0; offset < buf.Len(); offset += chunkSize {
		offsetEnd := offset + chunkSize
		if offsetEnd > buf.Len() {
			offsetEnd = buf.Len()
		}
		chunkBz, err := encodeChunk(chunk{
			Data:  buf.Bytes()[offset:offsetEnd],
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
	if len(data) < 5 {
		return nil, errors.New("empty chunk data")
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func encodeChunk(c chunk) ([]byte, error) {
	return json.Marshal(c)
}
