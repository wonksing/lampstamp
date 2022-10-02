package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/wonksing/lampstamp"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {

	clientChannel1 := make(chan Message)
	clientChannel2 := make(chan Message)
	var clientChannels sync.Map
	clientChannels.Store("client-1", clientChannel1)
	clientChannels.Store("client-2", clientChannel2)

	serverChannel1 := make(chan Message)
	serverChannel2 := make(chan Message)

	storage := NewStorage()

	s1 := NewServer("svr-1", storage, serverChannel1, &clientChannels, 200*time.Millisecond)
	go func() {
		for {
			s1.Receive()
		}
	}()
	s2 := NewServer("svr-2", storage, serverChannel2, &clientChannels, 0+time.Millisecond)
	go func() {
		for {
			s2.Receive()
		}
	}()

	c := NewClient("client-1", clientChannel1, serverChannel1, serverChannel2)
	c2 := NewClient("client-2", clientChannel2, serverChannel1, serverChannel2)

	go c.SendToServer1("msg-id-1", "foo")
	time.Sleep(100 * time.Millisecond)
	go c2.SendToServer2("msg-id-1", "bar")

	time.Sleep(5 * time.Second)

	mm := storage.Read("msg-id-1")
	log.Printf("message in storage: %s\n", mm.String())
}

type Message struct {
	ID       string
	ClientID string
	Text     string

	Version int64
}

func (m *Message) String() string {
	return fmt.Sprintf("%s(version: %d)", m.Text, m.Version)
}

type Client struct {
	ID string

	serverChannel1 chan Message
	serverChannel2 chan Message

	clientChannel chan Message

	Timestamp *lampstamp.MapStamp
}

func NewClient(id string, clientChannel, serverChannel1, serverChannel2 chan Message) *Client {
	return &Client{
		ID:             id,
		serverChannel1: serverChannel1,
		serverChannel2: serverChannel2,
		clientChannel:  clientChannel,
		Timestamp:      lampstamp.NewMapStamp(),
	}
}

func (c *Client) SendToServer1(id, text string) {
	c.send(c.serverChannel1, id, text)
}

func (c *Client) SendToServer2(id, text string) {
	c.send(c.serverChannel2, id, text)
}

func (c *Client) send(server chan Message, id, text string) {
	version := c.Timestamp.Inc(id)
	msg := Message{
		ID:       id,
		ClientID: c.ID,
		Text:     text,
		Version:  version,
	}
	log.Printf("%s sends '%s'\n", c.ID, msg.String())
	server <- msg

	resMsg := <-c.clientChannel

	version = c.Timestamp.Tick(id, resMsg.Version)
	log.Printf("\t\t%s received '%s'\n", c.ID, resMsg.String())
}

type Server struct {
	ID             string
	storage        *Storage
	serverChannel  chan Message
	clientChannels *sync.Map

	delay time.Duration

	Timestamp *lampstamp.MapStamp
}

func NewServer(id string, storage *Storage, serverChannel chan Message, clientChannels *sync.Map, delay time.Duration) *Server {
	return &Server{
		ID:             id,
		storage:        storage,
		serverChannel:  serverChannel,
		clientChannels: clientChannels,
		delay:          delay,

		Timestamp: lampstamp.NewMapStamp(),
	}
}
func (s *Server) Receive() {
	msg := <-s.serverChannel

	time.Sleep(s.delay)

	var version int64
	version = s.Timestamp.Tick(msg.ID, msg.Version)
	msg.Version = version

	s.storage.Begin()
	defer s.storage.End()

	s.storage.Store(msg)

	log.Printf("\t%s responds '%s'\n", s.ID, msg.String())
	clientChannel, _ := s.clientChannels.Load(msg.ClientID)
	clientChannel.(chan Message) <- Message{msg.ID, msg.ClientID, msg.Text, msg.Version}
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

func (s *Storage) ReadVersion(msgID string) int64 {
	if val, ok := s.m[msgID]; ok {
		return val.Version
	}
	return 0
}

func (s *Storage) Store(msg Message) {
	s.m[msg.ID] = msg
}

func (s *Storage) Read(id string) Message {
	if m, ok := s.m[id]; ok {
		return m
	}
	return Message{}
}
