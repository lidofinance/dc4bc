package qr

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

const chunkHeaderSize = 8

type ChunkHeader struct {
	Index uint32
	Total uint32
	Start uint32
	Len   uint32
}

type Chunk struct {
	Header ChunkHeader
	Data   []byte
}

func (c Chunk) IsEmpty() bool {
	return len(c.Data) == 0
}

func (c Chunk) MarshalBinary() ([]byte, error) {
	if len(c.Data) == 0 {
		return nil, errors.New("empty chunk data")
	}

	if c.Header.Total == 0 {
		return nil, errors.New("incorrect total value")
	}

	if c.Header.Index > c.Header.Total-1 {
		return nil, errors.New("index larger then total")
	}

	data := make([]byte, len(c.Data)+chunkHeaderSize)
	data[0], data[1] = byte(c.Header.Index), byte(c.Header.Index>>8)
	data[2], data[3] = byte(c.Header.Total), byte(c.Header.Total>>8)
	data[4], data[5] = byte(c.Header.Start), byte(c.Header.Start>>8)
	data[6], data[7] = byte(c.Header.Len), byte(c.Header.Len>>8)

	copy(data[chunkHeaderSize:], c.Data)
	return data, nil
}

func (c *Chunk) UnmarshalBinary(data []byte) error {
	if len(data) <= chunkHeaderSize {
		return errors.New("data length to short")
	}
	cLen := uint32(data[7])<<8 | uint32(data[6])
	*c = Chunk{
		Header: ChunkHeader{
			Index: uint32(data[1])<<8 | uint32(data[0]),
			Total: uint32(data[3])<<8 | uint32(data[2]),
			Start: uint32(data[5])<<8 | uint32(data[4]),
			Len:   cLen,
		},
		Data: make([]byte, cLen),
	}
	if uint32(len(data)) < chunkHeaderSize+cLen {
		return errors.New("data payload length to short")
	}

	copy(c.Data, data[chunkHeaderSize:chunkHeaderSize+cLen])

	return nil
}

type Chunks []*Chunk

func (cc Chunks) MarshalBinary() ([]byte, error) {
	if len(cc) == 0 {
		return nil, errors.New("empty chunks data")
	}

	data := make([]byte, 0)
	for idx, chunk := range cc {
		binChunk, err := chunk.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("cannot marshal chunk with index %d: %s", idx, err)
		}
		data = append(data, binChunk...)
	}

	return data, nil
}

func (cc *Chunks) UnmarshalBinary(data []byte) error {
	chunks := make([]*Chunk, 0)

	for idx := 0; idx < len(data); {
		chunk := &Chunk{}
		err := chunk.UnmarshalBinary(data[idx:])
		if err != nil {
			return fmt.Errorf("cannot marshal chunk with index %d: %s", idx, err)
		}
		chunks = append(chunks, chunk)
		idx += chunkHeaderSize + int(chunk.Header.Len)
	}
	*cc = chunks
	return nil
}

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
		chunk := &Chunk{
			Header: ChunkHeader{
				Index: uint32(index),
				Total: uint32(chunksCount),
				Start: uint32(offset),
				Len:   uint32(len(buf.Bytes()[offset:offsetEnd])),
			},
			Data: buf.Bytes()[offset:offsetEnd],
		}
		chunkBin, err := chunk.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to encode chunk: %w", err)
		}

		chunks = append(chunks, chunkBin)
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
