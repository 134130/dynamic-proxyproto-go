package dynamic_pp

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"testing"

	"github.com/pires/go-proxyproto"
)

func randBytes(length int) []byte {
	res, err := rand.Read(make([]byte, length))
	if err != nil {
		panic(err)
	}
	return make([]byte, res)
}

func TestParseRemoteAddr(t *testing.T) {
	var tests = []struct {
		name          string
		input         *proxyproto.Header
		clientContent []byte
		serverContent []byte
	}{{
		"ppv1/ipv4/small",
		&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		},
		[]byte("ping"),
		[]byte("pong"),
	}, {
		"ppv1/ipv6/small",
		&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv6,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 2000,
			},
		},
		[]byte("ping"),
		[]byte("pong"),
	}, {
		"ppv2/ipv4/small",
		&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		},
		[]byte("ping"),
		[]byte("pong"),
	}, {
		"ppv2/ipv6/small",
		&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv6,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 2000,
			},
		},
		[]byte("ping"),
		[]byte("pong"),
	}, {
		"bypass/ipv4/small",
		nil,
		[]byte("ping"),
		[]byte("pong"),
	}, {
		"bypass/ipv6/small",
		nil,
		[]byte("ping"),
		[]byte("pong"),
	}, {
		"ppv1/ipv4/large",
		&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		},
		randBytes(4096),
		randBytes(4096),
	}, {
		"ppv1/ipv6/large",
		&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv6,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 2000,
			},
		},
		randBytes(4096),
		randBytes(4096),
	}, {
		"ppv2/ipv4/large",
		&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		},
		randBytes(4096),
		randBytes(4096),
	}, {
		"ppv2/ipv6/large",
		&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv6,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 2000,
			},
		},
		randBytes(4096),
		randBytes(4096),
	}, {
		"bypass/ipv4/large",
		nil,
		randBytes(4096),
		randBytes(4096),
	}, {
		"bypass/ipv6/large",
		nil,
		randBytes(4096),
		randBytes(4096),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listening: %v", err)
			}

			ppln := &DynamicPPListener{Listener: ln}
			defer ppln.Close()

			clientResult := make(chan error)
			go func() {
				// Connect to the server
				clientConn, err := net.Dial("tcp", ppln.Addr().String())
				if err != nil {
					clientResult <- err
					return
				}
				defer clientConn.Close()

				// Write out the header
				if tt.input != nil {
					if _, err := tt.input.WriteTo(clientConn); err != nil {
						clientResult <- fmt.Errorf("client write header: %v", err)
						return
					}
				}

				// Send a content
				if _, err := clientConn.Write(tt.clientContent); err != nil {
					clientResult <- fmt.Errorf("client write: %v", err)
					return
				}

				// Read the response
				recv := make([]byte, len(tt.serverContent))
				if _, err = clientConn.Read(recv); err != nil {
					clientResult <- fmt.Errorf("client read: %v", err)
					return
				}
				// Check the response
				if !bytes.Equal(recv, tt.serverContent) {
					clientResult <- fmt.Errorf("client got: %v, want: %s", recv, tt.serverContent)
					return
				}
				close(clientResult)
			}()

			// Accept the connection
			serverConn, err := ppln.Accept()
			if err != nil {
				t.Fatalf("server accept: %v", err)
			}
			defer serverConn.Close()

			// Read the content
			recv := make([]byte, len(tt.clientContent))
			if _, err = serverConn.Read(recv); err != nil {
				t.Fatalf("server read: %v", err)
			}
			// Check the content
			if !bytes.Equal(recv, tt.clientContent) {
				t.Fatalf("server got: %v, want: %s", recv, tt.clientContent)
			}

			// Write the response
			if _, err := serverConn.Write(tt.serverContent); err != nil {
				t.Fatalf("server write: %v", err)
			}

			// Check the remote addr
			addr := serverConn.RemoteAddr().String()
			if tt.input != nil {
				if addr != tt.input.SourceAddr.String() {
					t.Fatalf("server remote addr: %v, want: %v", addr, tt.input.SourceAddr.String())
				}
			}

			// Check the client result
			if err := <-clientResult; err != nil {
				t.Fatalf("client result: %v", err)
			}
		})
	}
}

func TestMalformedHeader(t *testing.T) {
	var tests = []struct {
		name  string
		input *proxyproto.Header
	}{{
		"ppv1/ipv4",
		&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		},
	}, {
		"ppv1/ipv6",
		&proxyproto.Header{
			Version:           1,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv6,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 2000,
			},
		},
	}, {
		"ppv2/ipv4",
		&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		},
	}, {
		"ppv2/ipv6",
		&proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv6,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("ffff::ffff"),
				Port: 2000,
			},
		},
	}}

	for _, tt := range tests {
		format, _ := tt.input.Format()
		for i := 13; i < len(format); i++ {
			t.Run(fmt.Sprintf("%s/%d", tt.name, i), func(t *testing.T) {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("listening: %v", err)
				}

				ppln := &DynamicPPListener{Listener: ln}
				defer ppln.Close()

				clientResult := make(chan error)
				go func() {
					// Connect to the server
					clientConn, err := net.Dial("tcp", ppln.Addr().String())
					if err != nil {
						clientResult <- err
						return
					}
					defer clientConn.Close()

					// Write out the header
					if tt.input != nil {
						format, err = tt.input.Format()
						if err != nil {
							clientResult <- fmt.Errorf("client format header: %v", err)
							return
						}

						if _, err = clientConn.Write(format[:i]); err != nil {
							clientResult <- fmt.Errorf("client write header: %v", err)
							return
						}
					}

					close(clientResult)
				}()

				// Accept the connection
				serverConn, err := ppln.Accept()
				if err != nil {
					t.Fatalf("server accept: %v", err)
				}
				defer serverConn.Close()

				// Read the content
				recv := make([]byte, i)
				if _, err = serverConn.Read(recv); err != nil {
					t.Fatalf("server read: %v", err)
				}

				// Check the content
				if !bytes.Equal(recv, format[:i]) {
					t.Fatalf("server got: %v, want: %v", recv, format[:i])
				}

				// Check the client result
				if err := <-clientResult; err != nil {
					t.Fatalf("client result: %v", err)
				}
			})
		}
	}
}
