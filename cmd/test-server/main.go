package main

import (
	"fmt"
	"net"
)

func main() {
	c, _ := net.ListenPacket("udp", ":12345")
	buf := make([]byte, 4096)
	n, addr, _ := c.ReadFrom(buf)
	fmt.Println(string(buf[:n]))
	c.WriteTo([]byte("pong"), addr)
}
