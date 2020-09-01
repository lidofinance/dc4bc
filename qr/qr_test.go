package qr

import (
	"fmt"
	"testing"
)

func TestWriteChunks(t *testing.T) {
	data := []byte("hello i am a string for testing please don't delete me")
	p := NewCameraProcessor()
	chunks, err := DataToChunks(data)
	if err != nil {
		t.Fatal(err.Error())
	}
	for idx, chunk := range chunks {
		if err := p.WriteQR(fmt.Sprintf("%d.png", idx), chunk); err != nil {
			t.Fatal(err.Error())
		}
	}
}

func TestReadChunks(t *testing.T) {
	p := NewCameraProcessor()
	data, err := ReadDataFromQRChunks(p)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(string(data))
}
