package main

import (
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	client := NewClient("ascendex.com", "0/api/pro/v1/stream")
	err := client.Connection()
	if err != nil {
		t.Fatal(err)
	}

	data := make(chan BestOrderBook)
	client.ReadMessagesFromChannel(data)
	client.SubscribeToChannel("USDT_BTC")
	client.WriteMessagesToChannel()

	select {
	case <-time.After(20 * time.Second):
		t.Errorf("no data after 20 seconds")
	case res := <-data:
		t.Logf("data %+v", res)
	}
	client.Disconnect()
}
