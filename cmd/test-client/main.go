package main

import (
	"fmt"
	"net"
)

func main() {
	c, _ := net.ListenPacket("udp", ":0")
	a, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12346")
	c.WriteTo([]byte("ping"), a)
	buf := make([]byte, 4096)
	n, _, _ := c.ReadFrom(buf)
	fmt.Println(string(buf[:n]))
}
