package file_storage

import (
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/lidofinance/dc4bc/storage"
)

func randomBytes(n int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil
	}
	return b
}

func TestFileStorage_Send(t *testing.T) {
	N := 10
	var testFile = "/tmp/dc4bc_test_file_storage"
	fs, err := NewFileStorage(testFile)
	if err != nil {
		t.Error(err)
	}
	defer fs.Close()
	defer os.Remove(testFile)

	msgs := make([]storage.Message, 0, N)
	for i := 0; i < N; i++ {
		msg := storage.Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msgs = append(msgs, msg)
	}

	if err := fs.Send(msgs...); err != nil {
		t.Error(err)
	}

	offsetMsgs, err := fs.GetMessages(0)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(offsetMsgs, msgs) {
		t.Errorf("expected messages: %v, actual messages: %v", msgs, offsetMsgs)
	}
}
