package storage

import (
	"math/rand"
	"os"
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
	var offset uint64 = 5
	var testFile = "/tmp/dc4bc_test_file_storage"
	fs, err := NewFileStorage(testFile)
	if err != nil {
		t.Error(err)
	}
	defer fs.Close()
	defer os.Remove(testFile)

	msgs := make([]Message, 0, N)
	for i := 0; i < N; i++ {
		msg := Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msg, err = fs.Send(msg)
		if err != nil {
			t.Error(err)
		}
		msgs = append(msgs, msg)
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

func TestFileStorage_SendBatch(t *testing.T) {
	N := 10
	var offset uint64 = 5
	var testFile = "/tmp/dc4bc_test_file_storage"
	fs, err := NewFileStorage(testFile)
	if err != nil {
		t.Error(err)
	}
	defer fs.Close()
	defer os.Remove(testFile)

	msgs := make([]Message, 0, N)
	for i := 0; i < N; i++ {
		msg := Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msgs = append(msgs, msg)
	}

	sentMsgs, err := fs.SendBatch(msgs...)
	if err != nil {
		t.Error(err)
	}

	offsetMsgs, err := fs.GetMessages(offset)
	if err != nil {
		t.Error(err)
	}

	expectedOffsetMsgs := sentMsgs[offset:]
	if !reflect.DeepEqual(offsetMsgs, expectedOffsetMsgs) {
		t.Errorf("expected messages: %v, actual messages: %v", expectedOffsetMsgs, offsetMsgs)
	}
}
