package common

import (
	"testing"
)

func TestDedup(t *testing.T) {
	inChannel := make(chan LogMessage, 10)
	noDupChannel := Dedup(inChannel)
	inChannel <- LogMessage{
		ID: "1",
	}
	inChannel <- LogMessage{
		ID: "2",
	}
	inChannel <- LogMessage{
		ID: "3",
	}
	if msg := <-noDupChannel; msg.ID != "1" {
		t.Fatal("Didn't get first message")
	}
	if msg := <-noDupChannel; msg.ID != "2" {
		t.Fatal("Didn't get second message")
	}
	if msg := <-noDupChannel; msg.ID != "3" {
		t.Fatal("Didn't get third message")
	}
	inChannel <- LogMessage{
		ID: "3",
	}
	inChannel <- LogMessage{
		ID: "2",
	}
	inChannel <- LogMessage{
		ID: "4",
	}
	if msg := <-noDupChannel; msg.ID != "4" {
		t.Fatal("Should have received 4")
	}
}
