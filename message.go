package main

import (
	"encoding/binary"
	"io"
)

type messageID uint8

const (
	MsgChoke messageID = iota
	MsgUnchoke
	MsgInterested
	MsgNotInterested
	MsgHave
	MsgBitfield
	MsgRequest
	MsgPiece
	MsgCancel
)

type Message struct {
	ID      messageID
	Payload []byte
}

func (m *Message) serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1)
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)

	return buf
}

func readMessage(r io.Reader) (*Message, error) {
	var length uint32
	err := binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	// Keep alive message
	if length == 0 {
		return nil, nil
	}

	msg := make([]byte, length)
	_, err = io.ReadFull(r, msg)
	if err != nil {
		return nil, err
	}

	message := &Message{
		ID:      messageID(msg[0]),
		Payload: msg[1:],
	}

	return message, nil
}
