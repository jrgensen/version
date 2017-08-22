package ws

import (
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"time"
	"version/types"
)

// server.
type Server struct {
	pattern   string
	messages  []*types.Message
	clients   map[int]*Client
	addCh     chan *Client
	delCh     chan *Client
	sendAllCh chan *types.Message
	doneCh    chan bool
	errCh     chan error
}

// Create new server.
func NewServer(pattern string) *Server {
	messages := []*types.Message{}
	clients := make(map[int]*Client)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	sendAllCh := make(chan *types.Message)
	doneCh := make(chan bool)
	errCh := make(chan error)

	return &Server{
		pattern,
		messages,
		clients,
		addCh,
		delCh,
		sendAllCh,
		doneCh,
		errCh,
	}
}

func (s *Server) Add(c *Client) {
	log.Println("ADD 1")
	s.addCh <- c
}

func (s *Server) Del(c *Client) {
	log.Println("DELETE 1")
	s.delCh <- c
}

func (s *Server) SendAll(msg *types.Message) {
	log.Println("SENDALL 1")
	s.sendAllCh <- msg
}

func (s *Server) Done() {
	log.Println("DONE 1")
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	t := time.Now()
	log.Println("ERROR 1: ", t.Format(time.RFC3339))
	s.errCh <- err
}

func (s *Server) sendPastMessages(c *Client) {
	for _, msg := range s.messages {
		c.Write(msg)
	}
}

func (s *Server) sendAll(msg *types.Message) {
	for _, c := range s.clients {
		c.Write(msg)
	}
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {

	log.Println("Listening server...")

	// websocket handler
	onConnected := func(ws *websocket.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				log.Println("ERROR 2")
				s.errCh <- err
			}
		}()

		log.Printf("WS CONFIG: %+v\n", ws.Config())
		log.Printf("WS REQUEST: %+v\n", ws.Request())
		client := NewClient(ws, s)
		s.Add(client)
		client.Listen()
	}
	http.Handle(s.pattern, websocket.Handler(onConnected))
	log.Println("Created handler")

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			log.Println("Added new client")
			s.clients[c.id] = c
			log.Println("Now", len(s.clients), "clients connected.")
			s.sendPastMessages(c)

		// del a client
		case c := <-s.delCh:
			log.Println("Delete client")
			delete(s.clients, c.id)

		// broadcast message for all clients
		case msg := <-s.sendAllCh:
			log.Println("Send all:", msg)
			s.messages = s.messages[:0] // as long as we send everything in one chunk, do not cache history
			s.messages = append(s.messages, msg)
			s.sendAll(msg)

		case err := <-s.errCh:
			log.Println("HANDLING ERROR")
			log.Println("DEBUG ERROR", err)

			log.Println("Error:", err.Error())

		case <-s.doneCh:
			log.Println("Done!")
			return
		}
	}
}
