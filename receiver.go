package sacn

import (
	"fmt"
	"log"
	"net"
	"golang.org/x/net/ipv4"

	"github.com/libp2p/go-reuseport"
	"github.com/patopesto/go-sacn/packet"
)



type Receiver struct {
	// conn *net.UDPConn
	conn *ipv4.PacketConn
	itf  *net.Interface
	stop chan bool

}

const SACN_PORT = 5568


func NewReceiver(itf *net.Interface) *Receiver{
	r := &Receiver{}

	addr := fmt.Sprintf(":%d", SACN_PORT)
	listener, err := reuseport.ListenPacket("udp4", addr)
	if err != nil {
		log.Panic(err)
	}
	udpConn := listener.(*net.UDPConn)
	r.conn = ipv4.NewPacketConn(udpConn)
	r.itf = itf

	return r
}


func (r *Receiver) Start() {

	r.stop = make(chan bool)

	go r.recvLoop()
}

func(r *Receiver) JoinUniverse(universe uint16) {
	err := r.conn.JoinGroup(r.itf, universeToAddress(universe))
	if err != nil {
		panic(fmt.Sprintf("Could not join multicast group for universe %v: %v", universe, err))
	}
}

func (r *Receiver) LeaveUniverse(universe uint16) {
	err := r.conn.LeaveGroup(r.itf, universeToAddress(universe))
	if err != nil {
		panic(fmt.Sprintf("Could not leave multicast group for universe %v: %v", universe, err))
	}
}

func (r *Receiver) recvLoop() {

	defer r.conn.Close()

	for {
		select {
		case <- r.stop:
            return
        default:
			buf := make([]byte, 1024)
			n, _, source, err := r.conn.ReadFrom(buf)
			if err != nil {
				log.Panicln(err)
				continue
			}

			fmt.Printf("Received %d bytes from %s\n", n, source.String())
			var p packet.SACNPacket
			p, err = packet.Unmarshal(buf[:n])
			if err != nil {
				continue
			}

			r.handlePacket(p)
		}

	}
}

func (r *Receiver) handlePacket(p packet.SACNPacket) {
	fmt.Println("handle packet")
}


func universeToAddress(universe uint16) *net.UDPAddr {
	bytes := []byte{byte(universe >> 8), byte(universe & 0xFF)}
	ip := fmt.Sprintf("239.255.%v.%v", bytes[0], bytes[1])
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, SACN_PORT))
	return addr
}