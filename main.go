package main

import (
	"log"
	"os"
	"os/signal"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	client := NewClient("ascendex.com", "0/api/pro/v1/stream")
	err := client.Connection()
	if err != nil {
		log.Fatal(err)
	}
	data := make(chan BestOrderBook)
	client.ReadMessagesFromChannel(data)
	client.SubscribeToChannel("USDT_BTC")
	client.WriteMessagesToChannel()

outer:
	for {
		select {
		case book := <-data:
			log.Printf("recieved data:\n %+v", book)
		case <-client.Done():
			break outer
		case <-interrupt:
			log.Println("interrupt")
			client.Disconnect()

			break outer
		}
	}
}
