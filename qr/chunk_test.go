package qr

import (
	"reflect"
	"testing"
)

const (
	testChunkStr1 = "Test chunk string 1"
	testIndex1    = uint32(0)
	testTotal     = uint32(3)
	testStart1    = uint32(0)

	testChunkStr2 = "Test chunk string 2"
	testIndex2    = uint32(1)
	testStart2    = uint32(20)

	testChunkStr3 = "Test chunk string 3"
	testIndex3    = uint32(2)
	testStart3    = uint32(40)
)

var (
	testChunkBin = []byte{0, 0, 3, 0, 0, 0, 19, 0, 84, 101, 115, 116, 32, 99, 104, 117, 110, 107, 32, 115, 116, 114,
		105, 110, 103, 32, 49}
	testChunksBin = []byte{0, 0, 3, 0, 0, 0, 19, 0, 84, 101, 115, 116, 32, 99, 104, 117, 110, 107, 32, 115, 116, 114,
		105, 110, 103, 32, 49, 1, 0, 3, 0, 20, 0, 19, 0, 84, 101, 115, 116, 32, 99, 104, 117, 110, 107, 32, 115, 116,
		114, 105, 110, 103, 32, 50, 2, 0, 3, 0, 40, 0, 19, 0, 84, 101, 115, 116, 32, 99, 104, 117, 110, 107, 32, 115,
		116, 114, 105, 110, 103, 32, 51}
)

func TestChunkMarshal(t *testing.T) {
	c := Chunk{
		Header: ChunkHeader{
			Index: testIndex1,
			Total: testTotal,
			Start: testStart1,
			Len:   uint32(len(testChunkStr1)),
		},
		Data: []byte(testChunkStr1),
	}
	data, err := c.MarshalBinary()

	if err != nil {
		t.Fatalf(err.Error())
	}

	if !reflect.DeepEqual(data, testChunkBin) {
		t.Fatal("marshaled data and initial data are not equal!")
	}
}

func TestChunkUnMarshal(t *testing.T) {
	c := &Chunk{}
	err := c.UnmarshalBinary(testChunkBin)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if c.Header.Index != testIndex1 {
		t.Fatal("Header.Index is not marshaled")
	}

	if c.Header.Total != testTotal {
		t.Fatal("Header.Total is not marshaled")
	}

	if c.Header.Start != testStart1 {
		t.Fatal("Header.Start is not marshaled")
	}

	if string(c.Data) != testChunkStr1 {
		t.Fatal("unmarshalled data and initial data are not equal!")
	}

}

func TestChunksMarshal(t *testing.T) {
	cc := Chunks{
		{
			Header: ChunkHeader{
				Index: testIndex1,
				Total: testTotal,
				Start: testStart1,
				Len:   uint32(len(testChunkStr1)),
			},
			Data: []byte(testChunkStr1),
		},
		{
			Header: ChunkHeader{
				Index: testIndex2,
				Total: testTotal,
				Start: testStart2,
				Len:   uint32(len(testChunkStr2)),
			},
			Data: []byte(testChunkStr2),
		},
		{
			Header: ChunkHeader{
				Index: testIndex3,
				Total: testTotal,
				Start: testStart3,
				Len:   uint32(len(testChunkStr3)),
			},
			Data: []byte(testChunkStr3),
		},
	}
	data, err := cc.MarshalBinary()

	if err != nil {
		t.Fatalf(err.Error())
	}

	if !reflect.DeepEqual(data, testChunksBin) {
		t.Fatal("marshaled data and initial data are not equal!")
	}
}

func TestChunksUnMarshal(t *testing.T) {
	cc := &Chunks{}
	err := cc.UnmarshalBinary(testChunksBin)
	if err != nil {
		t.Fatalf(err.Error())
	}

	ccEtalon := &Chunks{
		{
			Header: ChunkHeader{
				Index: testIndex1,
				Total: testTotal,
				Start: testStart1,
				Len:   uint32(len(testChunkStr1)),
			},
			Data: []byte(testChunkStr1),
		},
		{
			Header: ChunkHeader{
				Index: testIndex2,
				Total: testTotal,
				Start: testStart2,
				Len:   uint32(len(testChunkStr2)),
			},
			Data: []byte(testChunkStr2),
		},
		{
			Header: ChunkHeader{
				Index: testIndex3,
				Total: testTotal,
				Start: testStart3,
				Len:   uint32(len(testChunkStr3)),
			},
			Data: []byte(testChunkStr3),
		},
	}

	if !reflect.DeepEqual(cc, ccEtalon) {
		t.Fatal("unmarshalled data and initial data are not equal!")
	}
}
