package main

import (
	"fmt"
	"time"
)

func main() {
	ts := NewTimestamp()
	count := ts.Get("wonk")
	fmt.Println(count)

	count = ts.Tick("wonk", 0)
	fmt.Println(count)

	count = ts.Inc("wonk")
	fmt.Println(count)

	count = ts.Inc("wonk")
	fmt.Println(count)

	serverChannel1 := make(chan Message)
	serverChannel2 := make(chan Message)

	s1 := NewServer("S1", NewTimestamp(), serverChannel1)
	go s1.Receive()

	c := NewClient("wonk", NewTimestamp(), serverChannel1, serverChannel2)
	c.SendToServer1("hello")
	time.Sleep(5 * time.Second)
}

type Message struct {
	ClientID string
	Text     string
	Count    int64
}

type Client struct {
	ID             string
	ts             *Timestamp
	serverChannel1 chan Message
	serverChannel2 chan Message
}

func NewClient(id string, ts *Timestamp, serverChannel1, serverChannel2 chan Message) *Client {
	return &Client{
		ID:             id,
		ts:             ts,
		serverChannel1: serverChannel1,
		serverChannel2: serverChannel2,
	}
}

func (c *Client) SendToServer1(msg string) {
	count := c.ts.Inc(c.ID)
	fmt.Printf("client: %s, count: %d\n", c.ID, count)
	c.serverChannel1 <- Message{c.ID, msg, count}
}

type Server struct {
	ID string
	ts *Timestamp
	ch chan Message
}

func NewServer(id string, ts *Timestamp, ch chan Message) *Server {
	return &Server{
		ID: id,
		ts: ts,
		ch: ch,
	}
}
func (s *Server) Receive() {
	for m := range s.ch {
		count := s.ts.Tick(m.ClientID, m.Count)
		fmt.Printf("%s, %d\n", s.ID, count)
	}
}
