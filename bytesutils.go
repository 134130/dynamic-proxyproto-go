package dynamic_pp

import (
	"bytes"
	"io"
)

func (c *dynamicPPConn) readString(delim byte) (string, error) {
	i := bytes.IndexByte(c.buff[c.offset:], delim)
	if i == -1 {
		return "", io.EOF
	}
	c.offset += i + 1
	return string(c.buff[c.offset-i-1 : c.offset-1]), nil
}
func (c *dynamicPPConn) readString2(delim []byte) (string, error) {
	i := bytes.Index(c.buff[c.offset:], delim)
	if i == -1 {
		return "", io.EOF
	}
	c.offset += i + len(delim)
	return string(c.buff[c.offset-i-len(delim) : c.offset-len(delim)]), nil
}

func (c *dynamicPPConn) readByte() (byte, error) {
	if c.offset >= len(c.buff) {
		return 0, io.EOF
	}
	c.offset++
	return c.buff[c.offset-1], nil
}

func (c *dynamicPPConn) readBytes(n int) ([]byte, error) {
	if c.offset+n > len(c.buff) {
		return nil, io.EOF
	}
	c.offset += n
	return c.buff[c.offset-n : c.offset], nil
}
