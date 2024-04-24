package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/tidwall/resp"
)

type Peer struct {
	conn  net.Conn
	msgCh chan Message
	delCH chan *Peer
}

func (p *Peer) Send(msg []byte) (int, error) {
	return p.conn.Write(msg)
}

func NewPeer(conn net.Conn, msgCh chan Message, delCh chan *Peer) *Peer {
	return &Peer{
		conn:  conn,
		msgCh: msgCh,
		delCH: delCh,
	}
}

func (p *Peer) readLoop() error {
	rd := resp.NewReader(p.conn)

	for {
		v, _, err := rd.ReadValue()

		if err == io.EOF {
			p.delCH <- p
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		var cmd Command

		if v.Type() == resp.Array {
			rawCmd := v.Array()[0]

			switch rawCmd.String() {
			case CommandClient:
				cmd = ClientCommand{
					value: v.Array()[1].String(),
				}
			case CommandGET:
				cmd = GetCommand{
					key: v.Array()[1].Bytes(),
				}
			case CommandSET:
				cmd = SetCommand{
					key: v.Array()[1].Bytes(),
					val: v.Array()[2].Bytes(),
				}
			case CommandHello:
				cmd = HelloCommand{
					// value: v.Array()[1].String(),
				}
			default:
				fmt.Println("got this unhandled error", rawCmd)
			}

			p.msgCh <- Message{
				cmd:  cmd,
				peer: p,
			}
		}
	}
	return nil
}
