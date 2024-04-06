package dynamic_pp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

var (
	PPv1IdentifierLen = 6
	PPv1Identifier    = []byte{'\x50', '\x52', '\x4F', '\x58', '\x59', '\x20'}
	PPv2IdentifierLen = 12
	PPv2Identifier    = []byte{'\x0D', '\x0A', '\x0D', '\x0A', '\x00', '\x0D', '\x0A', '\x51', '\x55', '\x49', '\x54', '\x0A'}
)

func (c *dynamicPPConn) parseV1() (*net.TCPAddr, *net.TCPAddr, error) {
	inetProtocolAndFamily, err := c.readString(' ')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read inet protocol and family: %w", err)
	}

	switch inetProtocolAndFamily {
	case "TCP4", "TCP6":
		// nothing to do
	default:
		return nil, nil, fmt.Errorf("unsupported inet protocol and family: %s", inetProtocolAndFamily)
	}

	si, err := c.readString(' ')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read source ip: %w", err)
	}

	sourceIP := net.ParseIP(si)
	if sourceIP == nil {
		return nil, nil, fmt.Errorf("invalid source ip: %s", si)
	}

	di, err := c.readString(' ')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read destination ip: %w", err)
	}

	destinationIP := net.ParseIP(di)
	if destinationIP == nil {
		return nil, nil, fmt.Errorf("invalid destination ip: %s", di)
	}

	sp, err := c.readString(' ')
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read source port: %w", err)
	}

	sourcePort, err := strconv.Atoi(sp)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid source port: %d", sourcePort)
	}

	dp, err := c.readString2([]byte{'\r', '\n'})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read destination port: %w", err)
	}

	destinationPort, err := strconv.Atoi(dp)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid destination port: %d", destinationPort)
	}

	sourceAddr := &net.TCPAddr{
		IP:   sourceIP,
		Port: sourcePort,
	}
	destinationAddr := &net.TCPAddr{
		IP:   destinationIP,
		Port: destinationPort,
	}
	return sourceAddr, destinationAddr, nil
}

type _addr4 struct {
	Src     [4]byte
	Dst     [4]byte
	SrcPort uint16
	DstPort uint16
}
type _addr6 struct {
	Src     [16]byte
	Dst     [16]byte
	SrcPort uint16
	DstPort uint16
}
type _addrUnix struct {
	Src [108]byte
	Dst [108]byte
}

func (c *dynamicPPConn) parseV2() (net.Addr, net.Addr, error) {
	protocolVersionAndCommand, err := c.readByte()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read protocol version and command byte: %w", err)
	}

	version := protocolVersionAndCommand >> 4
	command := protocolVersionAndCommand & 0x0F
	if version != 2 {
		return nil, nil, fmt.Errorf("unsupported protocol version %d. only 2 is supported", version)
	}

	switch command {
	case 0x00:
		// LOCAL
		return nil, nil, fmt.Errorf("LOCAL command is not supported")
	case 0x01:
		// PROXY
		// nothing to do
	default:
		return nil, nil, fmt.Errorf("unsupported command %d. only 0 and 1 are supported", command)
	}

	addressFamilyAndTransport, err := c.readByte()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read address family and transport byte: %w", err)
	}

	addressFamily := addressFamilyAndTransport >> 4
	transport := addressFamilyAndTransport & 0x0F
	switch addressFamily {
	case 0x00:
		// UNSPEC
		return nil, nil, fmt.Errorf("UNSPEC address family is not supported")
	case 0x01, 0x02, 0x03:
		// INET, INET6, UNIX
		// nothing to do
	default:
		return nil, nil, fmt.Errorf("unsupported address family %d. only 0, 1, 2, and 3 are supported", addressFamily)
	}

	switch transport {
	case 0x00:
		// UNSPEC
		return nil, nil, fmt.Errorf("UNSPEC transport is not supported")
	case 0x01:
		// STREAM (TCP)
		// nothing to do
	case 0x02:
		// DGRAM (UDP)
		return nil, nil, fmt.Errorf("DGRAM(UDP) transport is not supported")
	default:
		return nil, nil, fmt.Errorf("unsupported transport %d. only 0, 1, and 2 are supported", transport)
	}

	followingLengthBytes, err := c.readBytes(2)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read following length: %w", err)
	}
	followingLength := binary.BigEndian.Uint16(followingLengthBytes)

	followingHeaderBytes, err := c.readBytes(int(followingLength))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read following header. expected %d bytes: %w", followingLength, err)
	}

	switch addressFamily {
	case 0x01:
		// INET
		var addr _addr4
		if err = binary.Read(bytes.NewReader(followingHeaderBytes), binary.BigEndian, &addr); err != nil {
			return nil, nil, fmt.Errorf("failed to read INET address: %w", err)
		}
		srcAddr := &net.TCPAddr{
			IP:   addr.Src[:],
			Port: int(addr.SrcPort),
		}
		dstAddr := &net.TCPAddr{
			IP:   addr.Dst[:],
			Port: int(addr.DstPort),
		}
		return srcAddr, dstAddr, nil
	case 0x02:
		// INET6
		var addr _addr6
		if err = binary.Read(bytes.NewReader(followingHeaderBytes), binary.BigEndian, &addr); err != nil {
			return nil, nil, fmt.Errorf("failed to read INET6 address: %w", err)
		}
		srcAddr := &net.TCPAddr{
			IP:   addr.Src[:],
			Port: int(addr.SrcPort),
		}
		dstAddr := &net.TCPAddr{
			IP:   addr.Dst[:],
			Port: int(addr.DstPort),
		}
		return srcAddr, dstAddr, nil
	case 0x03:
		// UNIX
		var addr _addrUnix
		if err = binary.Read(bytes.NewReader(followingHeaderBytes), binary.BigEndian, &addr); err != nil {
			return nil, nil, fmt.Errorf("failed to read UNIX address: %w", err)
		}
		srcAddr := &net.UnixAddr{
			Name: string(addr.Src[:]),
		}
		dstAddr := &net.UnixAddr{
			Name: string(addr.Dst[:]),
		}
		return srcAddr, dstAddr, nil
	default:
		return nil, nil, fmt.Errorf("unsupported address family %d. only 1, 2, and 3 are supported", addressFamily)
	}
}
