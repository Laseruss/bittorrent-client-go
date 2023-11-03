package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"time"
)

const (
	MAXBLOCKSIZE = 16384
	MAXBACKLOG   = 5
)

type piece struct {
	index  int
	hash   [20]byte
	length int
}

type result struct {
	index int
	data  []byte
}

type pieceState struct {
	c          *client
	buf        []byte
	index      int
	downloaded int
	requested  int
	backlog    int
}

func Download(t *Torrent) ([]byte, error) {
	fmt.Println("starting download for", t.info.name)

	workQueue := make(chan *piece, len(t.info.pieces))
	results := make(chan *result)

	for index, hash := range t.info.pieces {
		workQueue <- &piece{index, hash, t.info.pieceLength}
	}

	peers, err := getPeers(t)
	if err != nil {
		return nil, err
	}

	for _, peer := range peers {
		go startWorker(t, peer, workQueue, results)
	}

	buf := make([]byte, t.info.length)
	donePieces := 0
	for donePieces < len(t.info.pieces) {
		res := <-results
		offset := res.index * t.info.pieceLength
		copy(buf[offset:], res.data)
		donePieces++

		percent := float64(donePieces) / float64(len(t.info.pieces)) * 100
		numWorkers := runtime.NumGoroutine() - 1
		fmt.Printf("(%0.2f%%) Downloaded piece #%d from %d peers\n", percent, res.index, numWorkers)
	}
	close(workQueue)

	return buf, nil
}

func startWorker(torrent *Torrent, peer Peer, workQueue chan *piece, results chan *result) {
	c, err := newClient(peer, torrent.peerID, torrent.info.infoHash)
	if err != nil {
		fmt.Println("could not set up the client with peer: ", peer.IP)
		return
	}
	defer c.conn.Close()
	fmt.Printf("Completed handshake with %s\n", peer.IP)

	c.sendUnchoke()
	c.sendInterested()

	for p := range workQueue {
		if !c.bitfield.HasPiece(p.index) {
			workQueue <- p // put the piece back on the queue
			continue
		}

		// download the piece
		buf, err := downloadPiece(c, p)
		if err != nil {
			fmt.Println("exiting", err)
			workQueue <- p
			return
		}

		if !checkIntegrity(p, buf) {
			fmt.Println("the received piece hash did not match expected")
			workQueue <- p
			continue
		}

		c.sendHave(p.index)
		results <- &result{p.index, buf}
	}
}

func (ps *pieceState) read() error {
	msg, err := ps.c.read()
	if err != nil {
		return err
	}

	if msg == nil { // Keep alive
		return nil
	}

	switch msg.ID {
	case MsgUnchoke:
		ps.c.choked = false
	case MsgChoke:
		ps.c.choked = true
	case MsgHave:
		pieceIdx := 0
		for _, b := range msg.Payload {
			pieceIdx = pieceIdx << 8
			pieceIdx |= int(b)
		}

		ps.c.bitfield.SetPiece(pieceIdx)
	case MsgPiece:
		if len(msg.Payload) < 8 {
			return errors.New("The piece message was to short.")
		}

		index := binary.BigEndian.Uint32(msg.Payload[:4])
		if index != uint32(ps.index) {
			return errors.New("the received piece index did not match the expected")
		}

		begin := binary.BigEndian.Uint32(msg.Payload[4:8])
		copy(ps.buf[begin:], msg.Payload[8:])

		ps.downloaded += len(msg.Payload) - 8
		ps.backlog--
	}

	return nil
}

func downloadPiece(c *client, p *piece) ([]byte, error) {
	state := pieceState{
		c:          c,
		buf:        make([]byte, p.length),
		index:      p.index,
		downloaded: 0,
		backlog:    0,
		requested:  0,
	}

	c.conn.SetDeadline(time.Now().Add(45 * time.Second))
	defer c.conn.SetDeadline(time.Time{})

	for state.downloaded < p.length {
		if !state.c.choked {
			for state.backlog < MAXBACKLOG && state.requested < p.length {
				// request a block
				blocksize := MAXBLOCKSIZE // 16 kb is the normal block size

				if p.length-state.requested < blocksize {
					blocksize = p.length - state.requested
				}

				err := c.sendRequest(p.index, state.requested, blocksize)
				if err != nil {
					return nil, err
				}

				state.backlog++
				state.requested += blocksize
			}
		}

		err := state.read()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
}

func checkIntegrity(p *piece, buf []byte) bool {
	hash := sha1.Sum(buf)

	if !bytes.Equal(hash[:], p.hash[:]) {
		return false
	}

	return true
}
