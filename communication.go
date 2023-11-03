package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/Laseruss/bittorrent-client/bencode"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p Peer) String() string {
	return fmt.Sprintf("%s:%d", p.IP.String(), p.Port)
}

type Peers []Peer

func getPeers(t *Torrent) (Peers, error) {
	url, err := t.buildTrackerURL()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	dec := bencode.NewDecoder(resp.Body)
	defer resp.Body.Close()

	val, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	dict, ok := val.(bencode.Dictionary)
	if !ok {
		return nil, errors.New("expected to get a dictionary")
	}

	peersData, ok := dict["peers"].([]byte)
	if !ok {
		return nil, errors.New("expected peers to be []byte")
	}

	peers, err := deserializePeers(peersData)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func deserializePeers(data []byte) (Peers, error) {
	const peerSize = 6
	if len(data)%peerSize != 0 {
		return nil, errors.New("got malformed peers info")
	}
	numPeers := len(data) / peerSize

	peers := make(Peers, 0, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peer := Peer{}
		peer.IP = net.IP(data[offset : offset+4])
		peer.Port = binary.BigEndian.Uint16(data[offset+4 : offset+6])

		peers = append(peers, peer)
	}

	return peers, nil
}
