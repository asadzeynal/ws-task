package main

type APIClient interface {
	/*
		Implement a websocket connection function
	*/
	Connection() error

	/*
		Implement a disconnect function from websocket
	*/
	Disconnect()

	/*
		Implement a function that will subscribe to updates
		of BBO for a given symbol

		The symbol must be of the form "TOKEN_ASSET"
		As an example "USDT_BTC" where USDT is TOKEN and BTC is ASSET

		You will need to convert the symbol in such a way that
		it complies with the exchange standard
	*/
	SubscribeToChannel(symbol string) error

	/*
		Implement a function that will write the data that
		we receive from the exchange websocket to the channel
	*/
	ReadMessagesFromChannel(ch chan<- BestOrderBook)

	/*
		Implement a function that will support connecting to a websocket
	*/
	WriteMessagesToChannel()
}

// BestOrderBook struct
type BestOrderBook struct {
	Ask Order `json:"ask"` //asks.Price > any bids.Price
	Bid Order `json:"bid"`
}

// Order struct
type Order struct {
	Amount float64 `json:"amount"`
	Price  float64 `json:"price"`
}
