package main

import (
	"flag"
	"log"
	"net"
	"sync"

	"github.com/zc-staff/tudp/tunnel"
)

var (
	tcpAddr   = flag.String("tcpAddr", "127.0.0.1:4344", "")
	udpAddr   = flag.String("udpAddr", "127.0.0.1:12345", "")
	bufSize   = flag.Int("bufSize", 4096, "")
	conn2conn sync.Map
)

func listenTCP(t tunnel.Tunnel, remote *net.UDPAddr) {
	for {
		conn, data := t.Recv()
		log.Println("receive from conn", conn, len(data), "bytes")
		var c net.PacketConn
		if v, ok := conn2conn.Load(conn); ok {
			c = v.(net.PacketConn)
		} else {
			var err error
			c, err = net.DialUDP("udp", nil, remote)
			if err != nil {
				log.Println("create conn", conn, "error:", err)
				continue
			}
			log.Println("create conn", conn, "local", c.LocalAddr())
			conn2conn.Store(conn, c)
			go listenUDP(t, conn, c)
		}

		n, err := c.WriteTo(data, remote)
		if err != nil {
			log.Println("send to remote", remote, "via", c.LocalAddr(), "error:", err)
		} else {
			log.Println("send to remote", remote, "via", c.LocalAddr(), n, "bytes")
		}
	}
}

func listenUDP(t tunnel.Tunnel, conn int, c net.PacketConn) error {
	defer c.Close()
	buf := make([]byte, *bufSize)
	for {
		n, addr, err := c.ReadFrom(buf)
		if err != nil {
			log.Println("read from remote via", c.LocalAddr(), "error:", err)
			return err
		}
		log.Println("read from remote", addr, "via", c.LocalAddr(), n, "bytes")
		t.Send(conn, buf[:n])
		log.Println("send to tunnel conn", conn)
	}
}

func main() {
	flag.Parse()
	t, err := tunnel.TCPTunnel.Dial(*tcpAddr)
	if err != nil {
		log.Fatal("dial tcp tunnel", *tcpAddr, "error:", err)
	}

	addr, err := net.ResolveUDPAddr("udp", *udpAddr)
	if err != nil {
		log.Fatal("resolve udp addr", *udpAddr, "error:", err)
	}

	listenTCP(t, addr)
}
