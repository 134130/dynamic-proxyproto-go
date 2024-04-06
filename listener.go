package dynamic_pp

import (
	"bytes"
	"net"
	"sync"
)

type DynamicPPListener struct {
	net.Listener
}

func (l *DynamicPPListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return newDynamicPPConn(conn), nil
}

type dynamicPPConn struct {
	net.Conn
	once     sync.Once
	buff     []byte
	offset   int
	readFunc func([]byte) (int, error)
	readErr  error

	srcAddr net.Addr
	dstAddr net.Addr
}

func newDynamicPPConn(conn net.Conn) *dynamicPPConn {
	c := &dynamicPPConn{
		Conn: conn,
		buff: make([]byte, 512),
	}
	c.readFunc = c.deferredReadFunc
	return c
}

func (c *dynamicPPConn) Read(b []byte) (int, error) {
	c.once.Do(func() {
		c.readErr = c.readProxyProtocol()
	})
	if c.readErr != nil {
		return 0, c.readErr
	}
	return c.readFunc(b)
}

func (c *dynamicPPConn) LocalAddr() net.Addr {
	c.once.Do(func() {
		c.readErr = c.readProxyProtocol()
	})

	if c.dstAddr != nil {
		return c.dstAddr
	}
	return c.Conn.LocalAddr()
}

func (c *dynamicPPConn) RemoteAddr() net.Addr {
	c.once.Do(func() {
		c.readErr = c.readProxyProtocol()
	})

	if c.srcAddr != nil {
		return c.srcAddr
	}
	return c.Conn.RemoteAddr()
}

func (c *dynamicPPConn) readProxyProtocol() error {
	n, err := c.Conn.Read(c.buff)
	if err != nil {
		return err
	}
	c.buff = c.buff[:n]
	c.readFunc = c.deferredReadFunc

	switch {
	case PPv1IdentifierLen <= n && bytes.Equal(c.buff[:PPv1IdentifierLen], PPv1Identifier):
		// skip the identifier
		c.offset = PPv1IdentifierLen
		// parse v1
		srcAddr, dstAddr, err := c.parseV1()
		if err != nil {
			// failed to parse proxy protocol v1
			// reset the buffer to the original state
			c.offset = 0
		}
		c.srcAddr, c.dstAddr = srcAddr, dstAddr
	case PPv2IdentifierLen <= n && bytes.Equal(c.buff[:PPv2IdentifierLen], PPv2Identifier):
		// skip the signature
		c.offset = PPv2IdentifierLen
		// parse v2
		srcAddr, dstAddr, err := c.parseV2()
		if err != nil {
			// failed to parse proxy protocol v2
			// reset the buffer to the original state
			c.offset = 0
		}
		c.srcAddr, c.dstAddr = srcAddr, dstAddr
	default:
	}
	return nil
}

func (c *dynamicPPConn) deferredReadFunc(b []byte) (int, error) {
	if len(c.buff)-c.offset > 0 {
		n := copy(b, c.buff[c.offset:])
		c.offset += n
		return n, nil
	}

	c.readFunc = c.Conn.Read
	return c.readFunc(b)
}

var _ net.Conn = &dynamicPPConn{}
