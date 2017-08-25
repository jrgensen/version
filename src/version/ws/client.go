package ws

import (
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/net/websocket"
	//	"sync"
	"version/types"
)

const channelBufSize = 10

var maxId int = 0

// client.
type Client struct {
	id     int
	ws     *websocket.Conn
	server *Server
	ch     chan *types.Message
	doneCh chan bool
	//	sync.Mutex
}

// Create new client.
func NewClient(ws *websocket.Conn, server *Server) *Client {

	if ws == nil {
		panic("ws cannot be nil")
	}

	if server == nil {
		panic("server cannot be nil")
	}

	maxId++
	ch := make(chan *types.Message, channelBufSize)
	doneCh := make(chan bool)

	return &Client{id: maxId, ws: ws, server: server, ch: ch, doneCh: doneCh}
}

func (c *Client) Conn() *websocket.Conn {
	return c.ws
}

func (c *Client) Write(msg *types.Message) {
	//	c.Lock()
	//	defer c.Unlock()

	if len(c.ch) == cap(c.ch) {
		log.Println("Client buffer full, connection closed.")
		c.Done()
		return
	}
	select {
	case c.ch <- msg:
	default:
		c.server.Del(c)
		err := fmt.Errorf("client %d is disconnected.", c.id)
		c.server.Err(err)
	}
}

func (c *Client) Done() {
	log.Println("Client is done, closing!:", c.id, " doneCh length:", len(c.doneCh))
	c.doneCh <- true
}

// Listen Write and Read request via chanel
func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

// Listen write request via chanel
func (c *Client) listenWrite() {
	log.Println("Listening write to client")
	for {
		select {

		// send message to the client
		case msg := <-c.ch:
			log.Println("sending to client:", c.id)

			c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))

			err := websocket.JSON.Send(c.ws, msg)
			c.ws.SetWriteDeadline(time.Time{}) // disabling timeout
			if err != nil {
				c.server.Err(fmt.Errorf("client %d is disconnected (%s).", c.id, err))
                c.doneCh <- true // for listenRead method
                return
			}
			log.Println("content send to client:", c.id)

		// receive done request
		case <-c.doneCh:
			log.Println("Client closing write connection", c.id)
			c.server.Del(c)
			c.doneCh <- true // for listenRead method
			return
		}
	}
}

// Listen read request via chanel
func (c *Client) listenRead() {
	log.Println("Listening read from client")
	for {
		select {

		// receive done request
		case <-c.doneCh:
			log.Println("Client closing read connection, id:", c.id)
			c.server.Del(c)
			c.doneCh <- true // for listenWrite method
			return

		// read data from websocket connection
		default:
			log.Println("Doing client default stuff, id:", c.id)
			var msg types.Message
			err := websocket.JSON.Receive(c.ws, &msg)
			if err == io.EOF {
				c.doneCh <- true
			} else if err != nil {
				c.server.Err(err)
			} else {
				c.server.SendAll(&msg)
			}
			log.Println("Done doing client default stuff, id:", c.id)
		}
	}
}
