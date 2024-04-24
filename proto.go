package main

import (
	"bytes"
	"fmt"

	"github.com/tidwall/resp"
)

type Command interface {
}

const (
	CommandSET    = "set"
	CommandGET    = "get"
	CommandHello  = "hello"
	CommandClient = "client"
)

type SetCommand struct {
	key, val []byte
}

type GetCommand struct {
	key []byte
}

type ClientCommand struct {
	value string
}

type HelloCommand struct {
	value string
}

func respWriteMap(m map[string]string) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("%" + fmt.Sprintf("%d\r\n", len(m)))
	rw := resp.NewWriter(buf)
	for k, v := range m {
		rw.WriteString(k)
		rw.WriteString(v)
	}
	return buf.Bytes()
}
