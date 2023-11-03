package bencode

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type (
	Dictionary map[string]interface{}
	List       []interface{}
)

type Decoder struct {
	rd *bufio.Reader
}

func NewDecoder(rd io.Reader) *Decoder {
	return &Decoder{
		rd: bufio.NewReader(rd),
	}
}

// Wrapper read functions for Decoder
func (d *Decoder) readByte() (byte, error) {
	b, err := d.rd.ReadByte()
	if err != nil {
		return 0, err
	}

	return b, nil
}

func (d *Decoder) readBytes(delim byte) ([]byte, error) {
	bytes, err := d.rd.ReadBytes(delim)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (d *Decoder) read(buf []byte) (int, error) {
	n, err := d.rd.Read(buf)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (d *Decoder) peek() (byte, error) {
	bytes, err := d.rd.Peek(1)
	if err != nil {
		return 0, err
	}

	return bytes[0], nil
}

// Functions to decode the different types of bencode values

func (d *Decoder) Decode() (interface{}, error) {
	return d.decodeVal()
}

func (d *Decoder) decodeVal() (interface{}, error) {
	b, err := d.peek()
	if err != nil {
		return nil, err
	}

	switch b {
	case 'i':
		return d.decodeNumber()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return d.decodeString()
	case 'l':
		return d.decodeList()
	case 'd':
		return d.decodeDict()
	default:
		return nil, errors.New(fmt.Sprintf("I didn't recognize that character: %x", b))
	}
}

func (d *Decoder) decodeNumber() (int, error) {
	b, err := d.readByte()
	if err != nil {
		return 0, err
	}

	if b != 'i' {
		return 0, errors.New("expected to get 'i' to start a number")
	}

	bytes, err := d.readBytes('e')
	if err != nil {
		return 0, err
	}

	bytes = bytes[:len(bytes)-1]

	res, err := strconv.Atoi(string(bytes))
	if err != nil {
		return 0, err
	}

	return res, nil
}

func (d *Decoder) decodeString() ([]byte, error) {
	bytes, err := d.readBytes(':')
	if err != nil {
		return nil, err
	}
	bytes = bytes[:len(bytes)-1]

	length, err := strconv.Atoi(string(bytes))
	if err != nil {
		return nil, err
	}

	buf := make([]byte, length)
	currIdx := 0
	for {
		n, err := d.read(buf[currIdx:])
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		currIdx += n

		if currIdx+n == length {
			break
		}
	}

	return buf, nil
}

func (d *Decoder) decodeList() (List, error) {
	b, err := d.readByte()
	if err != nil {
		return nil, err
	} else if b != 'l' {
		return nil, errors.New("expected first character in list to be 'l'")
	}

	l := List{}

	for {
		b, err = d.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			break
		}

		val, err := d.decodeVal()
		if err != nil {
			return nil, err
		}

		l = append(l, val)
	}

	return l, nil
}

func (d *Decoder) decodeDict() (Dictionary, error) {
	b, err := d.readByte()
	if err != nil {
		return nil, err
	} else if b != 'd' {
		return nil, errors.New("expected first character in dict to b 'd'")
	}

	dict := Dictionary{}
	for {
		b, err = d.peek()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			break
		}

		key, err := d.decodeVal()
		if err != nil {
			return nil, err
		}

		k, ok := key.([]byte)
		if !ok {
			return nil, errors.New("key in dict need to be a string")
		}

		val, err := d.decodeVal()
		if err != nil {
			return nil, err
		}

		dict[string(k)] = val
	}

	return dict, nil
}
