package sacn

import (
	"errors"
	"fmt"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"time"

	"github.com/libp2p/go-reuseport"
	"gitlab.com/patopest/go-sacn/packet"
)

type PacketCallbackFunc func(p packet.SACNPacket, source string)
type TerminationCallbackFunc func(universe uint16)

type Receiver struct {
	conn *ipv4.PacketConn
	itf  *net.Interface
	stop chan bool

	lastPackets      map[uint16]networkPacket
	streamTerminated map[uint16]bool

	packetCallbacks     map[packet.SACNPacketType]PacketCallbackFunc
	terminationCallback TerminationCallbackFunc
}

type networkPacket struct {
	ts     time.Time
	packet packet.SACNPacket
	// source 	net.UDPAddr
}

func NewReceiver(itf *net.Interface) *Receiver {
	r := &Receiver{}

	addr := fmt.Sprintf(":%d", SACN_PORT)
	listener, err := reuseport.ListenPacket("udp4", addr)
	if err != nil {
		log.Panicln(err)
	}
	udpConn := listener.(*net.UDPConn)
	r.conn = ipv4.NewPacketConn(udpConn)
	r.itf = itf

	r.lastPackets = make(map[uint16]networkPacket)
	r.streamTerminated = make(map[uint16]bool)
	r.packetCallbacks = make(map[packet.SACNPacketType]PacketCallbackFunc)

	return r
}

func (r *Receiver) Start() {

	r.stop = make(chan bool)

	go r.recvLoop()
}

func (r *Receiver) Stop() {
	close(r.stop)
}

func (r *Receiver) JoinUniverse(universe uint16) error {
	if universe == 0 || (universe > 64000 && universe != DISCOVERY_UNIVERSE) { // Section 9.1.1 of spec document
		return errors.New(fmt.Sprintf("Invalid universe number: %d\n", universe))
	}
	err := r.conn.JoinGroup(r.itf, universeToAddress(universe))
	if err != nil {
		return errors.New(fmt.Sprintf("Could not join multicast group for universe %v: %v", universe, err))
	}
	return nil
}

func (r *Receiver) LeaveUniverse(universe uint16) error {
	err := r.conn.LeaveGroup(r.itf, universeToAddress(universe))
	if err != nil {
		return errors.New(fmt.Sprintf("Could not leave multicast group for universe %v: %v", universe, err))
	}
	return nil
}

func (r *Receiver) RegisterPacketCallback(packetType packet.SACNPacketType, callback PacketCallbackFunc) {
	r.packetCallbacks[packetType] = callback
}

func (r *Receiver) RegisterTerminationCallback(callback TerminationCallbackFunc) {
	r.terminationCallback = callback
}

func (r *Receiver) recvLoop() {

	defer r.conn.Close()

	for {
		select {
		case <-r.stop:
			return
		default:
			buf := make([]byte, 1144) // 1144 is max packet size (full DiscoveryPacket)

			err := r.conn.SetDeadline(time.Now().Add(time.Millisecond * NETWORK_DATA_LOSS_TIMEOUT))
			if err != nil {
				log.Panic(fmt.Sprintf("Could not set deadline on socket: %v", err))
			}

			n, _, addr, _ := r.conn.ReadFrom(buf)
			if addr == nil { // timeout
				r.checkTimeouts()
				continue
			}

			source := addr.(*net.UDPAddr)
			// fmt.Printf("Received %d bytes from %s\n", n, source.String())
			var p packet.SACNPacket
			p, err = packet.Unmarshal(buf[:n])
			if err != nil {
				continue
			}

			r.handlePacket(p, source.IP.String())
		}

	}
}

func (r *Receiver) handlePacket(p packet.SACNPacket, source string) {
	r.checkTimeouts()
	packetType := p.GetType()

	switch packetType {
	case packet.PacketTypeData:
		d, _ := p.(*packet.DataPacket)
		r.storeLastPacket(d.Universe, d)
		if d.IsStreamTerminated() { // Bit 6: Stream Terminated
			r.terminateUniverse(d.Universe)
			return
		}
		if d.SyncAddress > 0 {
			_, ok := r.streamTerminated[d.SyncAddress]
			if !ok { // only join sync universe if not already
				r.JoinUniverse(d.SyncAddress)
			}
		}
	case packet.PacketTypeSync:
		s, _ := p.(*packet.SyncPacket)
		r.storeLastPacket(s.SyncAddress, s)
	}

	callback := r.packetCallbacks[packetType]
	if callback != nil {
		go callback(p, source)
	}
}

func (r *Receiver) storeLastPacket(universe uint16, p packet.SACNPacket) {
	r.lastPackets[universe] = networkPacket{
		ts:     time.Now(),
		packet: p,
	}
	r.streamTerminated[universe] = false
}

func (r *Receiver) checkTimeouts() {
	for universe, last := range r.lastPackets {
		if time.Since(last.ts) > time.Millisecond*NETWORK_DATA_LOSS_TIMEOUT {
			r.terminateUniverse(universe)
		}
	}
}

func (r *Receiver) terminateUniverse(universe uint16) {
	if r.terminationCallback != nil && !r.streamTerminated[universe] {
		go r.terminationCallback(universe)
		r.streamTerminated[universe] = true
	}
}
