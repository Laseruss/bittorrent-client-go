package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"time"
)

type client struct {
	conn       net.Conn
	choked     bool
	interested bool
	bitfield   Bitfield
	peer       Peer
	infoHash   [20]byte
	peerID     [20]byte
}

func newClient(peer Peer, peerID, infoHash [20]byte) (*client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bf, err := getBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	c := &client{
		conn:       conn,
		choked:     true,
		interested: false,
		bitfield:   bf,
		peer:       peer,
		infoHash:   infoHash,
		peerID:     peerID,
	}

	return c, nil
}

func (c *client) read() (*Message, error) {
	return readMessage(c.conn)
}

func completeHandshake(conn net.Conn, infoHash, id [20]byte) (*handshake, error) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // disable the deadline

	req := newHandshake(infoHash, id)
	_, err := conn.Write(req.serialize())
	if err != nil {
		return nil, err
	}

	res, err := deserializeHandshake(conn)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(res.infoHash[:], infoHash[:]) {
		return nil, errors.New("did not get matching info hashes during the handshake")
	}
	return res, nil
}

func getBitfield(conn net.Conn) (Bitfield, error) {
	msg, err := readMessage(conn)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, errors.New("expected a bitfield message")
	}

	if msg.ID != MsgBitfield {
		return nil, errors.New("expected msg to be of type bitfield")
	}

	return msg.Payload, nil
}

func (c *client) sendHave(index int) {
	msg := make([]byte, 9)

	binary.BigEndian.PutUint32(msg[0:4], 5)
	msg[4] = 4
	binary.BigEndian.PutUint32(msg[5:], uint32(index))

	c.conn.Write(msg)
}

func (c *client) sendUnchoke() error {
	msg := Message{ID: MsgUnchoke}
	_, err := c.conn.Write(msg.serialize())
	return err
}

func (c *client) sendInterested() error {
	msg := Message{ID: MsgInterested}
	_, err := c.conn.Write(msg.serialize())
	return err
}

func (c *client) sendRequest(pieceIdx, begin, blocksize int) error {
	msg := make([]byte, 13+4)
	binary.BigEndian.PutUint32(msg[:4], 13)
	msg[4] = 6
	binary.BigEndian.PutUint32(msg[5:9], uint32(pieceIdx))
	binary.BigEndian.PutUint32(msg[9:13], uint32(begin))
	binary.BigEndian.PutUint32(msg[13:], uint32(blocksize))

	_, err := c.conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}
