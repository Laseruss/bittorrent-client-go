package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/Laseruss/bittorrent-client/bencode"
)

type TorrentFile struct {
	name        string
	length      int
	infoHash    [20]byte
	pieceLength int
	pieces      [][20]byte
}

type Torrent struct {
	announce string
	peerID   [20]byte
	info     *TorrentFile
}

func createPeerId() ([20]byte, error) {
	var id [20]byte
	_, err := rand.Read(id[:])
	if err != nil {
		return id, err
	}

	return id, nil
}

func (t *TorrentFile) calculateInfoHash() {
	var buf bytes.Buffer

	buf.WriteByte('d')
	buf.Write(bencode.EncodeString("length"))
	buf.Write(bencode.EncodeInt(t.length))

	buf.Write(bencode.EncodeString("name"))
	buf.Write(bencode.EncodeString(t.name))

	buf.Write(bencode.EncodeString("piece length"))
	buf.Write(bencode.EncodeInt(t.pieceLength))

	buf.Write(bencode.EncodeString("pieces"))
	buf.WriteString(fmt.Sprintf("%d:", len(t.pieces)*20))
	for _, p := range t.pieces {
		buf.Write(p[:])
	}
	buf.WriteByte('e')

	t.infoHash = sha1.Sum(buf.Bytes())
}

// We can 100% make this a bit prettier but it parses the map[string]interface to typed structs instead
func newTorrent(f *os.File) (*Torrent, error) {
	dec := bencode.NewDecoder(f)

	val, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	dict, ok := val.(bencode.Dictionary)
	if !ok {
		return nil, err
	}

	return buildTorrent(dict)
}

func buildTorrent(data bencode.Dictionary) (*Torrent, error) {
	torrent := &Torrent{}

	if _, ok := data["announce"]; !ok {
		return nil, errors.New("expected announce to exist in torrent")
	}
	announce, ok := data["announce"].([]byte)
	if !ok {
		return nil, errors.New("expected announce to be string")
	}

	torrent.announce = string(announce)

	if _, ok := data["info"]; !ok {
		return nil, errors.New("expected info dict to exist in torrent")
	}
	info, ok := data["info"].(bencode.Dictionary)
	if !ok {
		return nil, errors.New("expected info to be dictionary")
	}

	file := &TorrentFile{}

	if _, ok := info["name"]; !ok {
		return nil, errors.New("expected info dict to contain name")
	}
	name, ok := info["name"].([]byte)
	if !ok {
		return nil, errors.New("expected name to be string")
	}
	file.name = string(name)

	if _, ok := info["piece length"]; !ok {
		return nil, errors.New("expected info dict to contain piece length")
	}
	pieceLength, ok := info["piece length"].(int)
	if !ok {
		return nil, errors.New("expected piece length to be int")
	}
	file.pieceLength = pieceLength

	if _, ok := info["length"]; !ok {
		return nil, errors.New("expected info dict to contain length")
	}
	length, ok := info["length"].(int)
	if !ok {
		return nil, errors.New("expected length to be int")
	}
	file.length = length

	if _, ok := info["pieces"]; !ok {
		return nil, errors.New("expected info dict to contain pieces")
	}
	pieces, ok := info["pieces"].([]byte)
	if !ok {
		return nil, errors.New("expected pieces to be string")
	}

	if len(pieces)%20 != 0 {
		return nil, errors.New("pieces not divisible by 20")
	}

	p := [][20]byte{}
	for i := 0; i < len(pieces); i += 20 {
		part := [20]byte{}

		for idx, b := range pieces[i : i+20] {
			part[idx] = byte(b)
		}

		p = append(p, part)
	}
	file.pieces = p

	torrent.info = file

	torrent.info.calculateInfoHash()

	id, err := createPeerId()
	if err != nil {
		return nil, err
	}

	torrent.peerID = id

	return torrent, nil
}

func (t *Torrent) buildTrackerURL() (string, error) {
	base, err := url.Parse(t.announce)
	if err != nil {
		return "", err
	}
	params := url.Values{
		"info_hash":  []string{string(t.info.infoHash[:])},
		"peer_id":    []string{string(t.peerID[:])},
		"port":       []string{"69420"},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.info.length)},
	}

	base.RawQuery = params.Encode()

	return base.String(), nil
}
