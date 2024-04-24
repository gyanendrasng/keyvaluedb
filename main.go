package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"

	"github.com/tidwall/resp"
)

const defaultListenAddress = ":5001"

type Config struct {
	ListenAddress string
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	delPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan Message
	kv        *KV
}

type Message struct {
	cmd  Command
	peer *Peer
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddress) == 0 {
		cfg.ListenAddress = defaultListenAddress
	}
	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
		msgCh:     make(chan Message),
		delPeerCh: make(chan *Peer),
		kv:        NewKV(),
	}
}

func (s *Server) handleMessage(msg Message) error {
	switch v := msg.cmd.(type) {
	case ClientCommand:
		if err := resp.NewWriter(msg.peer.conn).WriteString("OK"); err != nil {
			return err
		}
	case SetCommand:
		if err := s.kv.Set(v.key, v.val); err != nil {
			return err
		}

		if err := resp.NewWriter(msg.peer.conn).WriteString("OK"); err != nil {
			return err
		}
	case GetCommand:
		val, ok := s.kv.Get(v.key)

		if !ok {
			return fmt.Errorf("key not found")
		}

		if err := resp.NewWriter(msg.peer.conn).WriteString(string(val)); err != nil {
			return err
		}

	case HelloCommand:
		spec := map[string]string{
			"server": "redis",
		}
		_, err := msg.peer.Send((respWriteMap(spec)))

		if err != nil {
			return fmt.Errorf("peer send error:%s", err)
		}
	}
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case rawMsg := <-s.msgCh:
			if err := s.handleMessage(rawMsg); err != nil {
				slog.Error("Error while reading the message", "err", err)
			}
		case <-s.quitCh:
			return
		case peer := <-s.addPeerCh:
			s.peers[peer] = true

		case peer := <-s.delPeerCh:
			delete(s.peers, peer)
		}
	}
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()

		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh, s.delPeerCh)

	s.addPeerCh <- peer

	slog.Info("New Peer Connected", "Remote Addr", conn.RemoteAddr())

	if err := peer.readLoop(); err != nil {
		slog.Error("read error", "err", err, "remote addr", conn.RemoteAddr())
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddress)

	if err != nil {
		return err
	}

	defer ln.Close()

	s.ln = ln

	go s.loop()

	slog.Info("Server Running", "listenAddr", s.ListenAddress)

	return s.acceptLoop()
}

func main() {
	server := NewServer(Config{})
	log.Fatal(server.Start())
}
