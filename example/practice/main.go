package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func main() {

	clientChannel1 := make(chan Message)
	clientChannel2 := make(chan Message)
	var clientChannels sync.Map
	clientChannels.Store("wonk", clientChannel1)
	clientChannels.Store("bobi", clientChannel2)

	serverChannel1 := make(chan Message)
	serverChannel2 := make(chan Message)

	storage := NewStorage()

	s1 := NewServer("server-one", NewTimestamp(), storage, serverChannel1, &clientChannels)
	go func() {
		for {
			s1.Receive()
		}
	}()
	s2 := NewServer("server-two", NewTimestamp(), storage, serverChannel2, &clientChannels)
	go func() {
		for {
			s2.Receive()
		}
	}()

	c := NewClient("wonk", NewTimestamp(), clientChannel1, serverChannel1, serverChannel2)
	c2 := NewClient("bobi", NewTimestamp(), clientChannel2, serverChannel1, serverChannel2)
	// c.Send("msg-id-1", "hello")

	texts := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	for i := 0; i < len(texts); i++ {
		go c.SendToServer1("msg-id-1", "hello "+texts[i])
		// go c2.SendToServer2("msg-id-1", "hello "+texts[i])
		// go c.SendToServer2("msg-id-1", "hello "+texts[i])
	}
	time.Sleep(15 * time.Second)

	fmt.Println(c.ts.Get("msg-id-1"), c2.ts.Get("msg-id-1"), storage.m["msg-id-1"].Version)
}

type Message struct {
	ID       string
	ClientID string
	Text     string
	Version  int64
	Failed   bool
}

type Client struct {
	ID             string
	ts             *Timestamp
	serverChannel1 chan Message
	serverChannel2 chan Message

	clientChannel chan Message
}

func NewClient(id string, ts *Timestamp, clientChannel, serverChannel1, serverChannel2 chan Message) *Client {
	return &Client{
		ID:             id,
		ts:             ts,
		serverChannel1: serverChannel1,
		serverChannel2: serverChannel2,
		clientChannel:  clientChannel,
	}
}

func (c *Client) Send(id, text string) {
	texts := []string{"a", "b", "c"}
	version := c.ts.Inc(id)

	for i := 0; i < len(texts); i++ {

		go func(i int) {
			msg := Message{id, c.ID, text + texts[i], version, false}
			fmt.Printf("%s send %s(%s), version %d, %v\n", msg.ClientID, msg.ID, msg.Text, msg.Version, time.Now())
			c.serverChannel1 <- msg

			recvedMessage := <-c.clientChannel
			if recvedMessage.Failed {
				log.Printf("%s fail %s(%s), version %d\n", recvedMessage.ClientID, recvedMessage.ID, recvedMessage.Text, recvedMessage.Version)
				return
			}
			version = c.ts.Tick(id, recvedMessage.Version)
			fmt.Printf("%s recv %s(%s), version %d\n", recvedMessage.ClientID, recvedMessage.ID, recvedMessage.Text, version)
		}(i)
	}
}

func (c *Client) SendToServer1(id, text string) {
	c.send(c.serverChannel1, id, text)
}

func (c *Client) SendToServer2(id, text string) {
	c.send(c.serverChannel2, id, text)
}

func (c *Client) send(server chan Message, id, text string) {
	version := c.ts.Inc(id)
	msg := Message{id, c.ID, text, version, false}
	fmt.Printf("%s send %s(%s), version %d, %v\n", msg.ClientID, msg.ID, msg.Text, msg.Version, time.Now())
	server <- msg

	recvedMessage := <-c.clientChannel
	if recvedMessage.Failed {
		log.Printf("%s fail %s(%s), version %d\n", recvedMessage.ClientID, recvedMessage.ID, recvedMessage.Text, recvedMessage.Version)
		return
	}
	version = c.ts.Tick(id, recvedMessage.Version)
	fmt.Printf("%s recv %s(%s), version %d\n", recvedMessage.ClientID, recvedMessage.ID, recvedMessage.Text, version)
}

type Server struct {
	ID             string
	ts             *Timestamp
	storage        *Storage
	serverChannel  chan Message
	clientChannels *sync.Map
}

func NewServer(id string, ts *Timestamp, storage *Storage, serverChannel chan Message, clientChannels *sync.Map) *Server {
	return &Server{
		ID:             id,
		ts:             ts,
		storage:        storage,
		serverChannel:  serverChannel,
		clientChannels: clientChannels,
	}
}
func (s *Server) Receive() {
	msg := <-s.serverChannel
	// check before ticking
	var version int64
	if version = s.ts.Get(msg.ID); msg.Version < version {
		log.Printf("\t%s fail %s(%s), msg.Version < version %d < %d\n", s.ID, msg.ID, msg.Text, msg.Version, version)
		clientChannel, _ := s.clientChannels.Load(msg.ClientID)
		clientChannel.(chan Message) <- Message{msg.ID, msg.ClientID, msg.Text, msg.Version, true}
		return
	}

	version = s.ts.Tick(msg.ID, msg.Version)
	s.storage.Begin()

	// storage check is needed when server restart(resetting timestamp)
	storageMsgVersion := s.storage.ReadVersion(msg.ID)
	if storageMsgVersion >= version {
		log.Printf("\t%s fail %s(%s), storageMsgVersion >= version %d >= %d\n", s.ID, msg.ID, msg.Text, storageMsgVersion, version)
		clientChannel, _ := s.clientChannels.Load(msg.ClientID)
		clientChannel.(chan Message) <- Message{msg.ID, msg.ClientID, msg.Text, msg.Version, true}
		return
	}

	msg.Version = version
	s.storage.Store(msg)
	s.storage.End()

	fmt.Printf("\t%s respond %s(%s), version %d\n", s.ID, msg.ID, msg.Text, msg.Version)
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
