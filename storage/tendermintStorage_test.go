package storage

import (
	"reflect"
	"testing"
)

const (
	TestEndpoint = "http://0.0.0.0:1317"
	TestUserName = "user1"
	TestChainID  = "bulletin"
	TestTopic    = "test_topic"
	TestMnemonic = "dance vague property woman human world run real ship detect clown gossip indicate tone cry dress empower radio square guide laundry valley rose surface"
)

func TestNewTendermintStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}

	N := 100
	offset := 4

	ts, err := NewTendermintStorage(TestEndpoint, TestUserName, TestChainID, TestTopic, TestMnemonic)
	if err != nil {
		t.Error(err)
	}
	msgs := make([]Message, 0, N)
	for i := 0; i < N; i++ {
		msg := Message{
			Data:      randomBytes(10),
			Signature: randomBytes(10),
		}
		msg, err = ts.Send(msg)
		if err != nil {
			t.Error(err)
		}
		msgs = append(msgs, msg)
	}

	offsetMsgs, err := ts.GetMessages(uint64(offset))
	if err != nil {
		t.Error(err)
	}

	expectedOffsetMsgs := msgs[offset:]
	for idx, msg := range expectedOffsetMsgs {
		reflect.DeepEqual(msg.Signature, offsetMsgs[idx].Signature)
	}
}
