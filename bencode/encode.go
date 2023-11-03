package bencode

import (
	"bytes"
	"fmt"
	"strconv"
)

func EncodeString(s string) []byte {
	var out bytes.Buffer

	l := len(s)
	out.WriteString(strconv.Itoa(l))
	out.WriteByte(':')
	out.WriteString(s)

	return out.Bytes()
}

func EncodeInt(x int) []byte {
	var out bytes.Buffer

	out.WriteByte('i')
	num := fmt.Sprint(x)
	out.WriteString(num)
	out.WriteByte('e')

	return out.Bytes()
}
