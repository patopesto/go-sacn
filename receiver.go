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

// Reception mode of a packet
type PacketMode string

// Possible reception modes for a packet
const (
	PacketUnicast   PacketMode = "unicast"
	PacketMulticast PacketMode = "multicast"
	PacketBroadcast PacketMode = "broadcast"
)

// Struct of additional packet information when calling [PacketCallbackFunc] callbacks.
type PacketInfo struct {
	Source 		net.UDPAddr // The source address of the packet.
	Mode        PacketMode  // How the packet was received.
}

// PacketCallbackFunc is the function type to be used with [Receiver.RegisterPacketCallback].
// The arguments are the latest received [packet.SACNPacket] on any universe and a [PacketInfo] struct.
type PacketCallbackFunc func(p packet.SACNPacket, info PacketInfo)

// TerminationCallbackFunc is the function type to be used with [Receiver.RegisterTerminationCallback].
// The universe argument is the universe number which entered Network Data Loss conditions.
type TerminationCallbackFunc func(universe uint16)

// A sACN Receiver. Use [NewReceiver] to create a receiver.
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
}

// NewReceiver creates a new receiver bound to the provided interface
func NewReceiver(itf *net.Interface) (*Receiver, error) {
	r := &Receiver{}

	addr := fmt.Sprintf(":%d", SACN_PORT)
	listener, err := reuseport.ListenPacket("udp4", addr)
	if err != nil {
		return nil, err
	}
	udpConn := listener.(*net.UDPConn)
	r.conn = ipv4.NewPacketConn(udpConn)
    if err := r.conn.SetControlMessage(ipv4.FlagDst, true); err != nil { // Enable receiving of destination address info
        return nil, err
    }
	r.itf = itf

	r.lastPackets = make(map[uint16]networkPacket)
	r.streamTerminated = make(map[uint16]bool)
	r.packetCallbacks = make(map[packet.SACNPacketType]PacketCallbackFunc)

	return r, nil
}

// Starts the receiver
func (r *Receiver) Start() {

	r.stop = make(chan bool)

	go r.recvLoop()
}

// Stops the receiver
func (r *Receiver) Stop() {
	close(r.stop)
}

// JoinUniverse starts listening for packets sent on the provided universe.
// Universe number shall be in the range 1 to 63999.
// Joins the multicast group associated with the universe number.
func (r *Receiver) JoinUniverse(universe uint16) error {
	if universe == 0 || (universe > 64000 && universe != DISCOVERY_UNIVERSE) { // Section 9.1.1 of ANSI E1.31—2018
		return errors.New(fmt.Sprintf("Invalid universe number: %d\n", universe))
	}
	err := r.conn.JoinGroup(r.itf, universeToAddress(universe))
	if err != nil {
		return errors.New(fmt.Sprintf("Could not join multicast group for universe %v: %v", universe, err))
	}
	return nil
}

// Stops listening for packets sent on a universe.
// Leaves the multicast groups associated with the universe number.
func (r *Receiver) LeaveUniverse(universe uint16) error {
	err := r.conn.LeaveGroup(r.itf, universeToAddress(universe))
	if err != nil {
		return errors.New(fmt.Sprintf("Could not leave multicast group for universe %v: %v", universe, err))
	}
	return nil
}

// RegisterPacketCallback registers a callback of type PacketCallbackFunc.
// The callback will be triggered on reception of a new packet of type [packet.SACNPacketType] on any universe
func (r *Receiver) RegisterPacketCallback(packetType packet.SACNPacketType, callback PacketCallbackFunc) {
	r.packetCallbacks[packetType] = callback
}

// RegisterTerminationCallback registers a callback for when a universe enters Network Data Loss conditions as defined in section 6.7.1 of ANSI E1.31—2018.
//
// Network Data Loss conditions:
//   - Did not receive data for [NETWORK_DATA_LOSS_TIMEOUT].
//   - Data packet contained the StreamTerminated bit in the [packet.DataPacket] Options field.
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
			buf := make([]byte, 1144) // 1144 bytes is max packet size (full DiscoveryPacket)

			err := r.conn.SetDeadline(time.Now().Add(time.Millisecond * NETWORK_DATA_LOSS_TIMEOUT))
			if err != nil {
				log.Panic(fmt.Sprintf("Could not set deadline on socket: %v", err))
				return
			}

			n, cm, addr, _ := r.conn.ReadFrom(buf)
			if addr == nil { // timeout
				r.checkTimeouts()
				continue
			}

			// fmt.Printf("Received %d bytes from %s\n", n, source.String())
			var p packet.SACNPacket
			p, err = packet.Unmarshal(buf[:n])
			if err != nil {
				continue
			}

			var mode PacketMode
			if cm.Dst.Equal(net.IPv4bcast){ // Only handle local broadcast for now (ie: 255.255.255.255) not directed broadcast (ie: 192.168.1.255/24)
		        mode = "broadcast"
		    } else if cm.Dst.IsMulticast() {
		    	mode = "multicast"
		    } else {
		        mode = "unicast"
		    }

			info := PacketInfo{
				Source: *addr.(*net.UDPAddr),
				Mode: mode,
			}

			r.handlePacket(p, info)
		}

	}
}

func (r *Receiver) handlePacket(p packet.SACNPacket, info PacketInfo) {
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
		go callback(p, info)
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
