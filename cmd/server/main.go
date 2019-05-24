package main

import (
	"flag"
	"log"
	"net"
	"sync"

	"github.com/zc-staff/tudp/tunnel"
)

var (
	tcpAddr   = flag.String("tcpAddr", ":4344", "")
	udpAddr   = flag.String("udpAddr", ":12346", "")
	bufSize   = flag.Int("bufSize", 4096, "")
	conn2addr sync.Map
	addr2conn sync.Map
	conns     int
)

func listenTCP(t tunnel.Tunnel, udp net.PacketConn) {
	for {
		conn, data := t.Recv()
		log.Println("receive from tunnel conn", conn, len(data), "bytes")
		if v, ok := conn2addr.Load(conn); ok {
			addr := v.(net.Addr)
			n, err := udp.WriteTo(data, addr)
			if err != nil {
				log.Println("send to remote", addr, "via", udp.LocalAddr(), "error:", err)
			} else {
				log.Println("send to remote", addr, "via", udp.LocalAddr(), n, "bytes")
			}
		} else {
			log.Println("unknown conn, drop")
		}
	}
}

func listenUDP(t tunnel.Tunnel, udp net.PacketConn) {
	log.Println("listen udp", udp.LocalAddr())
	buf := make([]byte, *bufSize)
	for {
		n, addr, err := udp.ReadFrom(buf)
		if err != nil {
			log.Println("read from remote via", udp.LocalAddr(), "error:", err)
			continue
		}
		log.Println("read from remote", addr, "via", udp.LocalAddr(), n, "bytes")
		var conn int
		if v, ok := addr2conn.Load(addr); ok {
			conn = v.(int)
		} else {
			conns++
			conn = conns
			log.Println("new map", conn, "->", addr)
			conn2addr.Store(conn, addr)
			addr2conn.Store(addr, conn)
		}
		t.Send(conn, buf[:n])
		log.Println("send to tunnel conn", conn)
	}
}

func main() {
	flag.Parse()
	t, err := tunnel.TCPTunnel.Listen(*tcpAddr)
	if err != nil {
		log.Fatal("listen tunnel", *tcpAddr, "error:", err)
	}

	udp, err := net.ListenPacket("udp", *udpAddr)
	if err != nil {
		log.Fatal("listen packet conn", *udpAddr, "error:", err)
	}

	go listenTCP(t, udp)
	listenUDP(t, udp)
}
