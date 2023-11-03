package main

import (
	"errors"
	"io"
)

type handshake struct {
	pstr     string
	infoHash [20]byte
	peerID   [20]byte
}

const PEER_STRING = "BitTorrent protocol"

func newHandshake(infoHash, peerID [20]byte) *handshake {
	return &handshake{
		pstr:     PEER_STRING,
		infoHash: infoHash,
		peerID:   peerID,
	}
}

func (h handshake) serialize() []byte {
	buf := make([]byte, 68)
	buf[0] = 0x13 // len of pstr
	curr := 1
	curr += copy(buf[curr:], []byte(h.pstr))
	curr += copy(buf[curr:], []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}) // eight reserved bytes
	curr += copy(buf[curr:], h.infoHash[:])
	curr += copy(buf[curr:], h.peerID[:])

	return buf
}

func deserializeHandshake(r io.Reader) (*handshake, error) {
	pstrLen := make([]byte, 1)
	_, err := io.ReadFull(r, pstrLen)
	if err != nil {
		return nil, err
	}
	l := int(pstrLen[0])
	if l == 0 {
		return nil, errors.New("pstr len can not be 0")
	}

	handshakeBuf := make([]byte, 48+l)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[l+8:l+8+20]) // start reading after pstr and 8 reserved bytes and 20 bytes for the infohash
	copy(peerID[:], handshakeBuf[l+8+20:])      // start reading after infoHash and to the end to get the peerID

	h := &handshake{
		pstr:     string(handshakeBuf[0:l]),
		infoHash: infoHash,
		peerID:   peerID,
	}

	return h, nil
}
