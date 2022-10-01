package main

import (
	"log"
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
	log.Printf("Message in storage: %s\n", mm.String())
}

type Message struct {
	ID       string
	ClientID string
	Text     string

	Version int64
	Failed  bool
}

func (m *Message) String() string {
	return m.Text
}

type Client struct {
	ID string

	serverChannel1 chan Message
	serverChannel2 chan Message

	clientChannel chan Message

	Timestamp *LamportTimestamp
}

func NewClient(id string, clientChannel, serverChannel1, serverChannel2 chan Message) *Client {
	return &Client{
		ID:             id,
		serverChannel1: serverChannel1,
		serverChannel2: serverChannel2,
		clientChannel:  clientChannel,
		Timestamp:      NewTimestamp(),
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
		Failed:   false,
	}
	log.Printf("%s sends '%s'\n", c.ID, msg.String())
	server <- msg

	resMsg := <-c.clientChannel
	if resMsg.Failed {
		log.Printf("\t\tERROR: %s received '%s'\n", c.ID, resMsg.String())
		return
	}
	version = c.Timestamp.Tick(id, resMsg.Version)
	log.Printf("\t\t%s received '%s'\n", c.ID, resMsg.String())
}

type Server struct {
	ID             string
	storage        *Storage
	serverChannel  chan Message
	clientChannels *sync.Map

	delay time.Duration

	Timestamp *LamportTimestamp
}

func NewServer(id string, storage *Storage, serverChannel chan Message, clientChannels *sync.Map, delay time.Duration) *Server {
	return &Server{
		ID:             id,
		storage:        storage,
		serverChannel:  serverChannel,
		clientChannels: clientChannels,
		delay:          delay,

		Timestamp: NewTimestamp(),
	}
}
func (s *Server) Receive() {
	msg := <-s.serverChannel

	time.Sleep(s.delay)

	// validate before ticking
	var version int64
	if version = s.Timestamp.Get(msg.ID); msg.Version < version {
		msg.Failed = true
		log.Printf("\tERROR %s server version is higher than msg version, '%s'\n", s.ID, msg.String())
		// log.Printf("\t%s fail %s(%s), msg.Version < version %d < %d\n", s.ID, msg.ID, msg.Text, msg.Version, version)
		clientChannel, _ := s.clientChannels.Load(msg.ClientID)
		clientChannel.(chan Message) <- msg
		return
	}

	version = s.Timestamp.Tick(msg.ID, msg.Version)
	msg.Version = version

	s.storage.Begin()
	defer s.storage.End()

	// storage check is needed when server restart(resetting timestamp)
	// and when multiple servers are running
	storageMsgVersion := s.storage.ReadVersion(msg.ID)
	if storageMsgVersion >= msg.Version {
		msg.Failed = true
		log.Printf("\tERROR %s: msg version is not higher than storage's, '%s'\n", s.ID, msg.String())
		clientChannel, _ := s.clientChannels.Load(msg.ClientID)
		clientChannel.(chan Message) <- msg
		return
	}

	s.storage.Store(msg)

	log.Printf("\t%s responds '%s'\n", s.ID, msg.String())
	clientChannel, _ := s.clientChannels.Load(msg.ClientID)
	clientChannel.(chan Message) <- Message{msg.ID, msg.ClientID, msg.Text, msg.Version, false}
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
	return defaultTimestamp
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
