package main

import "testing"

func TestGenSignID(t *testing.T) {
	origName := "test"
	ID1, err := getSignID(origName)
	if err != nil {
		t.Fatal(err)
	}

	ID2, err := getSignID(origName)
	if err != nil {
		t.Fatal(err)
	}

	if ID1 == ID2 {
		t.Fatal("ID1 and ID2 should not be equal:", ID1)
	}
}
