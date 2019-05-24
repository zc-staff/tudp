package tunnel

// Tunnel is a generic tunnel for udp
type Tunnel interface {
	Send(conn int, data []byte)
	Recv() (int, []byte)
}

// Factory is the factory for tunnel
type Factory interface {
	Listen(addr string) (Tunnel, error)
	Dial(addr string) (Tunnel, error)
}
