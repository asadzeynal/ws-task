package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type UntypedJson map[string]any

type Client struct {
	addr string
	path string
	conn *websocket.Conn
	done chan struct{}
}

func NewClient(addr, path string) *Client {
	return &Client{
		addr: addr,
		path: path,
		done: make(chan struct{}),
	}
}

func (c *Client) Done() chan struct{} {
	return c.done
}

func (c *Client) Connection() error {
	url := url.URL{Scheme: "wss", Host: c.addr, Path: c.path}
	log.Printf("connecting to %s", url.String())

	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}

	c.conn = conn
	return nil
}

func (c *Client) Disconnect() {
	if c.conn == nil {
		log.Println("There is no connection")
		return
	}
	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
	}

	close(c.done)
	<-time.After(time.Second)
	c.conn.Close()
}

func (c *Client) SubscribeToChannel(symbol string) error {
	if c.conn == nil {
		return fmt.Errorf("there is no connection")
	}

	splitSymbol := strings.Split(symbol, "_")
	if len(splitSymbol) != 2 {
		return fmt.Errorf("incorrect symbol format")
	}

	message := struct {
		Op string `json:"op"`
		Ch string `json:"ch"`
	}{
		Op: "sub",
		Ch: fmt.Sprintf("bbo:%s/%s", splitSymbol[1], splitSymbol[0]),
	}

	err := c.conn.WriteJSON(message)
	if err != nil {
		c.Disconnect()
		return fmt.Errorf("error while subscribing, terminating connection: %w", err)
	}

	return nil
}

func (c *Client) ReadMessagesFromChannel(ch chan<- BestOrderBook) {
	go func() {
		for {
			select {
			case <-c.done:
				return
			case res := <-c.WaitForData():
				if len(res) == 0 {
					continue
				}

				var untyped UntypedJson
				err := json.Unmarshal(res, &untyped)
				if err != nil {
					log.Println("could not parse json :%w", err)
					continue
				}

				switch untyped["m"] {
				case "connected":
					log.Println("connected successfully")
				case "pong":
					log.Println("pong recieved from server")
				case "sub":
					log.Println("successfully subscribed to a channel")
				case "bbo":
					book, err := makeOrderBook(res)
					if err != nil {
						fmt.Println(err)
						continue
					}
					ch <- book
				default:
					log.Printf("unrecognized message type, message: %s", string(res))
				}
			}
		}
	}()
}

func (c *Client) WaitForData() chan []byte {
	ch := make(chan []byte)
	go func() {
		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("error while reading server response: %w", err)
			ch <- []byte{}
			return
		}

		ch <- bytes
	}()
	return ch
}

func (c *Client) WriteMessagesToChannel() {
	ticker := time.NewTicker(14 * time.Second)
	go func() {
		c.ping()
		for {
			select {
			case <-c.done:
				return
			case <-ticker.C:
				c.ping()
			}
		}
	}()
}

func (c *Client) ping() {
	msg := struct {
		Op string `json:"op"`
	}{"ping"}

	log.Println("sending ping to the server")
	err := c.conn.WriteJSON(msg)
	if err != nil {
		log.Println("error while sending ping: %w", err)
	}
}

func makeOrderBook(msg []byte) (BestOrderBook, error) {
	res := struct {
		Symbol string `json:"symbol"`
		Data   struct {
			Bid []string `json:"bid"`
			Ask []string `json:"ask"`
		} `json:"data"`
	}{}

	err := json.Unmarshal(msg, &res)
	if err != nil {
		return BestOrderBook{}, err
	}

	if len(res.Data.Ask) != 2 || len(res.Data.Bid) != 2 {
		return BestOrderBook{}, fmt.Errorf("error: invalid response from server")
	}
	askAmountS := res.Data.Ask[0]
	askPriceS := res.Data.Ask[1]
	bidAmountS := res.Data.Bid[0]
	bidPriceS := res.Data.Bid[1]

	askAmount, err := strconv.ParseFloat(askAmountS, 64)
	if err != nil {
		return BestOrderBook{}, fmt.Errorf("invalid float value: %s", askAmountS)
	}
	askPrice, err := strconv.ParseFloat(askPriceS, 64)
	if err != nil {
		return BestOrderBook{}, fmt.Errorf("invalid float value: %s", askPriceS)
	}
	bidAmount, err := strconv.ParseFloat(bidAmountS, 64)
	if err != nil {
		return BestOrderBook{}, fmt.Errorf("invalid float value: %s", bidAmountS)
	}
	bidPrice, err := strconv.ParseFloat(bidPriceS, 64)
	if err != nil {
		return BestOrderBook{}, fmt.Errorf("invalid float value: %s", bidPriceS)
	}

	askOrder := Order{
		Amount: askAmount,
		Price:  askPrice,
	}
	bidOrder := Order{
		Amount: bidAmount,
		Price:  bidPrice,
	}

	return BestOrderBook{Ask: askOrder, Bid: bidOrder}, nil
}
