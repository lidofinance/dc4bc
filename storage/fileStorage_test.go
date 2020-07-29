package storage

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func randomBytes(n int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil
	}
	return b
}

func TestFileStorage_GetMessages(t *testing.T) {
	N := 10
	offset := 5
	fs, err := InitFileStorage("test")
	if err != nil {
		t.Error(err)
	}
	defer fs.Close()
	msgs := make([]Message, 0, N)
	for i := 0; i < N; i++ {
		msg := Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msgs = append(msgs, msg)
		if err = fs.Post(msg); err != nil {
			t.Error(err)
		}
	}
	offsetMsgs, err := fs.GetMessages(offset)
	if err != nil {
		t.Error(err)
	}
	expectedOffsetMsgs := msgs[offset:]
	if !reflect.DeepEqual(offsetMsgs, expectedOffsetMsgs) {
		t.Errorf("expected messages: %v, actual messages: %v", expectedOffsetMsgs, offsetMsgs)
	}
}
