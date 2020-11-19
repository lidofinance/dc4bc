package storage

import (
	"fmt"
	"testing"
)

func TestNewTendermintStorage(t *testing.T) {
	ts, err := NewTendermintStorage("http://0.0.0.0:1317", "user1", "bulletin", "test_topic", "12345678", "trip spin bench ghost ride steak fame clutch desk lake fiction emotion pen peace spare output gun genuine soccer fury affair execute bar outdoor")
	if err != nil {
		t.Error(err)
	}
	info, err := ts.keybase.Get("user1")
	if err != nil {
		t.Error(err)
	}
	account, err := ts.getAccount(info.GetAddress().String())
	if err != nil {
		t.Error(err)
	}
	fmt.Println(account.GetCoins())

	msg := Message{
		ID:            "lsflksdjf",
		DkgRoundID:    "1234",
		Offset:        0,
		Event:         "test_event",
		Data:          []byte{1, 2, 3, 4},
		Signature:     []byte{1, 2, 3, 4},
		SenderAddr:    "sender",
		RecipientAddr: "recipient",
	}

	tx, err := ts.genTx(msg)
	if err != nil {
		t.Error(err)
	}

	signedTx, err := ts.signTx(*tx)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(signedTx.GetSignatures())
}
