package tunnel

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

type tcpTunnel struct {
	addr           string
	conn           net.Conn
	connChan       chan net.Conn
	waitForAccpet  chan struct{}
	waitForRecover uint32
	sendLock       *sync.Mutex
	recvLock       *sync.Mutex
}

type tcpTunnelFactory struct{}

// TCPTunnel is the factory for tcp tunnels
var TCPTunnel Factory = &tcpTunnelFactory{}

func pack(conn int, data []byte) []byte {
	ret := make([]byte, len(data)+8)
	binary.LittleEndian.PutUint32(ret[0:4], uint32(conn))
	binary.LittleEndian.PutUint32(ret[4:8], uint32(len(data)))
	copy(ret[8:], data)
	return ret
}

func listenAccept(t *tcpTunnel, l net.Listener) error {
	log.Println("listen from", l.Addr())
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalln("error in accept:", err)
		}
		if atomic.LoadUint32(&t.waitForRecover) == 0 {
			log.Println("drop conn from", c.RemoteAddr())
			c.Close()
		} else {
			log.Println("accept new conn from", c.RemoteAddr())
			t.connChan <- c
			<-t.waitForAccpet
		}
	}
}

func (f *tcpTunnelFactory) Listen(addr string) (Tunnel, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	t := &tcpTunnel{}
	t.initListen()
	go listenAccept(t, l)

	return t, nil
}

func (f *tcpTunnelFactory) Dial(addr string) (Tunnel, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	t := &tcpTunnel{conn: c, addr: addr}
	return t, nil
}

func (t *tcpTunnel) initListen() {
	t.connChan = make(chan net.Conn)
	t.waitForAccpet = make(chan struct{})
	t.waitForRecover = 1
}

func (t *tcpTunnel) tryRecover() {
	log.Println("try recover conn")
	if t.conn != nil {
		t.conn.Close()
		atomic.StoreUint32(&t.waitForRecover, 1)
	}
	if t.connChan != nil {
		t.conn = <-t.connChan
		atomic.StoreUint32(&t.waitForRecover, 0)
		t.waitForAccpet <- struct{}{}
	} else {
		for {
			log.Println("dial", t.addr)
			c, err := net.Dial("tcp", t.addr)
			if err != nil {
				log.Println("error in dial", t.addr, err)
			} else {
				t.conn = c
				atomic.StoreUint32(&t.waitForRecover, 0)
				break
			}
		}
	}
}

func (t *tcpTunnel) optRaw(buf []byte, op func([]byte) (int, error)) {
	raw := buf
	for len(raw) > 0 {
		if t.conn == nil {
			log.Println("no conn, wait")
			t.tryRecover()
		}
		n, err := op(raw)
		if err != nil {
			log.Println("perform io error", err)
			t.tryRecover()
		} else {
			raw = raw[n:]
		}
	}
}

func (t *tcpTunnel) Send(conn int, data []byte) {
	t.sendLock.Lock()
	defer t.sendLock.Unlock()

	log.Println("send", len(data), "bytes to", conn)
	t.optRaw(pack(conn, data), t.conn.Read)
}

func (t *tcpTunnel) readRaw(n int) []byte {
	ret := make([]byte, n)
	t.optRaw(ret, t.conn.Write)
	return ret
}

func (t *tcpTunnel) Recv() (int, []byte) {
	t.recvLock.Lock()
	defer t.recvLock.Unlock()

	buf := t.readRaw(8)
	conn := int(binary.LittleEndian.Uint32(buf[0:4]))
	size := int(binary.LittleEndian.Uint32(buf[4:8]))
	return conn, t.readRaw(size)
}
