package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {

	clientChannel1 := make(chan Message)
	clientChannel2 := make(chan Message)
	var clientChannels sync.Map
	clientChannels.Store("wonk", clientChannel1)
	clientChannels.Store("bobi", clientChannel2)

	serverChannel1 := make(chan Message)
	serverChannel2 := make(chan Message)

	storage := NewStorage()

	s1 := NewServer("server-one", storage, serverChannel1, &clientChannels, 200*time.Millisecond)
	go func() {
		for {
			s1.Receive()
		}
	}()
	s2 := NewServer("server-two", storage, serverChannel2, &clientChannels, 0+time.Millisecond)
	go func() {
		for {
			s2.Receive()
		}
	}()

	c := NewClient("wonk", clientChannel1, serverChannel1, serverChannel2)
	c2 := NewClient("bobi", clientChannel2, serverChannel1, serverChannel2)

	go c.SendToServer1("msg-id-1", c.ID+" hello")
	time.Sleep(100 * time.Millisecond)
	go c2.SendToServer2("msg-id-1", c2.ID+" hello")

	time.Sleep(5 * time.Second)

	fmt.Printf("%s, %s, %s\n", storage.m["msg-id-1"].ID, storage.m["msg-id-1"].ClientID, storage.m["msg-id-1"].Text)
}

type Message struct {
	ID       string
	ClientID string
	Text     string
}

type Client struct {
	ID string

	serverChannel1 chan Message
	serverChannel2 chan Message

	clientChannel chan Message
}

func NewClient(id string, clientChannel, serverChannel1, serverChannel2 chan Message) *Client {
	return &Client{
		ID:             id,
		serverChannel1: serverChannel1,
		serverChannel2: serverChannel2,
		clientChannel:  clientChannel,
	}
}

func (c *Client) SendToServer1(id, text string) {
	c.send(c.serverChannel1, id, text)
}

func (c *Client) SendToServer2(id, text string) {
	c.send(c.serverChannel2, id, text)
}

func (c *Client) send(server chan Message, id, text string) {
	msg := Message{
		ID:       id,
		ClientID: c.ID,
		Text:     text,
	}
	fmt.Printf("%s send %s(%s), %v\n", msg.ClientID, msg.ID, msg.Text, time.Now())
	server <- msg

	recvedMessage := <-c.clientChannel
	fmt.Printf("\t\t%s recv %s(%s)\n", recvedMessage.ClientID, recvedMessage.ID, recvedMessage.Text)
}

type Server struct {
	ID             string
	storage        *Storage
	serverChannel  chan Message
	clientChannels *sync.Map

	delay time.Duration
}

func NewServer(id string, storage *Storage, serverChannel chan Message, clientChannels *sync.Map, delay time.Duration) *Server {
	return &Server{
		ID:             id,
		storage:        storage,
		serverChannel:  serverChannel,
		clientChannels: clientChannels,
		delay:          delay,
	}
}
func (s *Server) Receive() {
	msg := <-s.serverChannel

	time.Sleep(s.delay)

	s.storage.Begin()
	defer s.storage.End()
	s.storage.Store(msg)

	fmt.Printf("\t%s respond %s(%s)\n", s.ID, msg.ID, msg.Text)
	clientChannel, _ := s.clientChannels.Load(msg.ClientID)
	clientChannel.(chan Message) <- Message{msg.ID, msg.ClientID, msg.Text}
}

type Storage struct {
	m map[string]Message
	l sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		m: make(map[string]Message),
	}
}

func (s *Storage) Begin() {
	s.l.Lock()
}

func (s *Storage) End() {
	s.l.Unlock()
}

func (s *Storage) Store(msg Message) {
	s.m[msg.ID] = msg
}
