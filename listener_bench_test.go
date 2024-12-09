package dynamic_pp

import (
	"net"
	"testing"

	"github.com/pires/go-proxyproto"
)

func benchmarkProxyProtocolListener(buffSize int, b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	header := &proxyproto.Header{
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
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}

	ppln := &DynamicPPListener{Listener: ln}
	defer ppln.Close()

	for i := 0; i < b.N; i++ {
		func() {
			buff := make([]byte, buffSize)

			clientResult := make(chan error)
			go func() {
				defer close(clientResult)

				// Connect to the server
				clientConn, err := net.Dial("tcp", ppln.Addr().String())
				if err != nil {
					clientResult <- err
					return
				}
				defer clientConn.Close()

				// Send the proxy protocol header
				if _, err := header.WriteTo(clientConn); err != nil {
					clientResult <- err
					return
				}

				// Send some data
				if _, err := clientConn.Write(buff); err != nil {
					clientResult <- err
					return
				}
			}()

			conn, err := ppln.Accept()
			if err != nil {
				b.Fatal(err)
				return
			}
			defer conn.Close()

			if _, err := conn.Read(buff); err != nil {
				b.Fatal(err)
				return
			}

			_ = conn.RemoteAddr()
			if err = <-clientResult; err != nil {
				b.Fatal(err)
				return
			}
		}()
	}
}

func BenchmarkProxyProtocolListener_BuffSize256(b *testing.B) {
	benchmarkProxyProtocolListener(256, b)
}
func BenchmarkProxyProtocolListener_BuffSize512(b *testing.B) {
	benchmarkProxyProtocolListener(512, b)
}
func BenchmarkProxyProtocolListener_BuffSize1024(b *testing.B) {
	benchmarkProxyProtocolListener(1024, b)
}
func BenchmarkProxyProtocolListener_BuffSize4096(b *testing.B) {
	benchmarkProxyProtocolListener(4096, b)
}
func BenchmarkProxyProtocolListener_BuffSize8192(b *testing.B) {
	benchmarkProxyProtocolListener(8192, b)
}
func BenchmarkProxyProtocolListener_BuffSize16384(b *testing.B) {
	benchmarkProxyProtocolListener(16384, b)
}
