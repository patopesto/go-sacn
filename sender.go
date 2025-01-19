package sacn

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"gitlab.com/patopest/go-sacn/packet"
)

// A sACN Sender. Use [NewSender] to create a receiver.
type Sender struct {
	conn *net.UDPConn

	universes map[uint16]*senderUniverse
	discovery *senderUniverse
	wg        sync.WaitGroup
	logger    *log.Logger

	// common options for packets
	cid        [16]byte
	sourceName string
	keepAlive  time.Duration
}

// Optional arguments for [NewSender] to be applied to all packets being sent by the sender.
// These can be overridden on a per packet basis if set in the [packet.SACNPacket] being sent.
type SenderOptions struct {
	CID        [16]byte    // the CID (Component Identifier): a RFC4122 compliant UUID.
	SourceName string      // A source name (must not be longer than 64 characters)
	Logger     *log.Logger // Optionally use an alternative logger instead of the default.
	// KeepAlive  time.Duration
}

// Stores all the information required per universe a sender is handling
type senderUniverse struct {
	number       uint16
	dataCh       chan packet.SACNPacket
	enabled      bool
	sequence     uint8
	multicast    bool
	destinations []net.UDPAddr
}

var universeNotFoundError = errors.New("Universe is not initialised, please use StartUniverse() first")

// NewSender creates a new [Sender]. Optionally pass a bind string of the host's ip address it should bind to (eg: "192.168.1.100").
// This is mandatory if multicast is being used on any universe.
func NewSender(address string, options *SenderOptions) (*Sender, error) {

	// Generate RFC 4122 compliant UUID. From ANSI E1.31-2019 Section 5.6
	cid, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	if options.CID[0] == 0 {
		bytes, _ := cid.MarshalBinary()
		copy(options.CID[:], bytes[:16])
	}
	if options.SourceName == "" {
		options.SourceName = "gitlab.com/patopest/go-sacn"
	}
	if len(options.SourceName) > 64 {
		return nil, errors.New("Source name is too long. Maximum is 64 bytes")
	}
	if options.Logger == nil {
		options.Logger = log.Default()
	}
	// if options.KeepAlive == 0 {
	// 	options.KeepAlive = 1 * time.Second
	// }

	server, err := net.ResolveUDPAddr("udp", address+":0")
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", server)
	if err != nil {
		return nil, err
	}

	s := &Sender{
		conn:       conn,
		universes:  make(map[uint16]*senderUniverse),
		cid:        options.CID,
		sourceName: options.SourceName,
		logger:     options.Logger,
		// keepAlive:  options.KeepAlive,
	}

	go s.sendDiscoveryLoop()

	return s, nil
}

// Stops the sender and all initialised universes
func (s *Sender) Close() {

	for _, uni := range s.universes {
		if uni.enabled {
			close(uni.dataCh)
		}
	}
	close(s.discovery.dataCh)
	s.wg.Wait()
	defer s.conn.Close()
}

// StartUniverse initialises a new universe to be sent by the sender.
// It returns a channel into which [packet.SACNPacket] can be written to for sending out on the network.
// Optionally you can use [Sender.Send] to also send packets for a universe.
func (s *Sender) StartUniverse(universe uint16) (chan<- packet.SACNPacket, error) {
	if s.IsEnabled(universe) == true {
		return nil, errors.New("Universe is already enabled")
	}
	if universe < 1 || universe >= 64000 { // From ANSI E1.31-2019 Section 6.2.7
		return nil, errors.New("Universe value is incorrect, should be between 1 and 63999")
	}

	ch := make(chan packet.SACNPacket, 3)
	uni := &senderUniverse{
		number:       universe,
		enabled:      true,
		dataCh:       ch,
		sequence:     0,
		multicast:    false,
		destinations: make([]net.UDPAddr, 0),
	}
	s.universes[universe] = uni

	go s.sendLoop(universe)

	return ch, nil
}

// StopUniverse stops sending packet for a universe.
// This closes the channel returned to by [Sender.StartUniverse].
// On closing, 3 [packet.DataPacket] will be sent out with the StreamTerminated bit set as specified in section 6.7.1 of ANSI E1.31—2018.
func (s *Sender) StopUniverse(universe uint16) error {

	uni, exists := s.universes[universe]
	if exists {
		close(uni.dataCh)
		return nil
	}
	return universeNotFoundError
}

// Send a packet on a universe.
// This is an alternative way to writing packets directly on the channel returned by [Sender.StartUniverse]
func (s *Sender) Send(universe uint16, p packet.SACNPacket) error {
	uni, exists := s.universes[universe]
	if exists {
		uni.dataCh <- p
		return nil
	}
	return universeNotFoundError
}

func (s *Sender) sendLoop(universe uint16) {

	uni := s.universes[universe]
	s.wg.Add(1)
	ch := uni.dataCh

	// Receive new packets to send out
	for p := range ch {
		uni.sequence += 1
		sequence := uni.sequence

		packetType := p.GetType()
		switch packetType {
		case packet.PacketTypeData:
			d, _ := p.(*packet.DataPacket)
			if d.CID[0] == 0 {
				d.CID = s.cid
			}
			d.Universe = universe
			d.Sequence = sequence // increment sequence number
			if d.GetSourceName() == "" {
				d.SetSourceName(s.sourceName)
			}
		case packet.PacketTypeSync:
			d, _ := p.(*packet.SyncPacket)
			if d.CID[0] == 0 {
				d.CID = s.cid
			}
			d.SyncAddress = universe
			d.Sequence = sequence
		case packet.PacketTypeDiscovery: // technically should never have this type of packet here
			d, _ := p.(*packet.DiscoveryPacket)
			if d.CID[0] == 0 {
				d.CID = s.cid
			}
			if d.GetSourceName() == "" {
				d.SetSourceName(s.sourceName)
			}
		default:
			continue
		}

		s.sendPacket(uni, p)
	}

	uni.enabled = false
	// Send packet with stream terminated bit 3 times
	p := packet.NewDataPacket()
	p.SetStreamTerminated(true)
	for i := 0; i < 3; i++ {
		s.sendPacket(uni, p)
	}

	// Destroy universe
	s.wg.Done()
	delete(s.universes, universe)
}

func (s *Sender) sendDiscoveryLoop() {

	s.discovery = &senderUniverse{
		number:    DISCOVERY_UNIVERSE,
		enabled:   true,
		multicast: true,
		dataCh:    make(chan packet.SACNPacket, 0), // still create a data channel to close on sender Close()
	}
	s.wg.Add(1)
	timer := time.NewTicker(UNIVERSE_DISCOVERY_INTERVAL * time.Second)
	defer timer.Stop()
	defer s.wg.Done()

	for {
		select {
		case <-s.discovery.dataCh: // channel was closed
			return
		case <-timer.C:
			num := len(s.universes)
			pages := num / 512
			universes := s.GetUniverses()
			for page := 0; page < pages+1; page += 1 {
				p := packet.NewDiscoveryPacket()
				p.Page = uint8(page)
				p.Last = uint8(pages)
				p.CID = s.cid
				p.SetSourceName(s.sourceName)

				start := page * 512
				end := (page + 1) * 512
				if end > len(universes) {
					end = len(universes)
				}
				p.SetUniverses(universes[start:end])

				s.sendPacket(s.discovery, p)
			}
		}
	}
}

func (s *Sender) sendPacket(universe *senderUniverse, p packet.SACNPacket) {

	bytes, err := p.MarshalBinary()
	if err != nil {
		s.logger.Println("Error", err)
		return
	}

	// send multicast if enabled
	if universe.multicast {
		_, err := s.conn.WriteToUDP(bytes, universeToAddress(universe.number))
		if err != nil {
			s.logger.Printf("Error sending multicast packet: %v\n", err)
		}
	}
	// send unicast
	for _, dest := range universe.destinations {
		_, err := s.conn.WriteToUDP(bytes, &dest)
		if err != nil {
			s.logger.Printf("Error sending unicast packet: %v\n", err)
		}
	}
}

// GetUniverses returns the list of all currently enabled universes for the sender.
func (s *Sender) GetUniverses() []uint16 {
	unis := make([]uint16, 0)
	for n, uni := range s.universes {
		if uni.enabled {
			unis = append(unis, n)
		}
	}
	return unis
}

// IsEnabled returns true if the universe is currently enabled.
func (s *Sender) IsEnabled(universe uint16) bool {
	uni, exists := s.universes[universe]
	if exists && uni.enabled {
		return true
	}
	return false
}

// IsMulticast returns wether or not multicast is turned on for the given universe.
func (s *Sender) IsMulticast(universe uint16) (bool, error) {
	uni, exists := s.universes[universe]
	if exists {
		return uni.multicast, nil
	}
	return false, universeNotFoundError
}

// SetMulticast is for setting whether or not a universe should be send out via multicast.
func (s *Sender) SetMulticast(universe uint16, multicast bool) error {
	uni, exists := s.universes[universe]
	if exists {
		uni.multicast = multicast
		return nil
	}
	return universeNotFoundError
}

// GetDestinations returns the list of unicast destinations the universe is configured to send it's packets to.
func (s *Sender) GetDestinations(universe uint16) ([]string, error) {
	dests := make([]string, 0)
	uni, exists := s.universes[universe]
	if exists {
		for _, dest := range uni.destinations {
			dests = append(dests, dest.IP.String())
		}
		return dests, nil
	}
	return nil, universeNotFoundError
}

// AddDestination adds a unicast destination that a universe should sent it's packets to.
// destination should be in the form of a string (eg: "192.168.1.100").
func (s *Sender) AddDestination(universe uint16, destination string) error {

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", destination, SACN_PORT))
	if err != nil {
		return err
	}

	uni, exists := s.universes[universe]
	if exists {
		uni.destinations = append(uni.destinations, *addr)
		return nil
	}
	return universeNotFoundError
}

// SetDestinations sets the list of unicast destinations that a univese should sent it's packets to.
// This overwrites the current list created by previous calls to this function or [Sender.AddDestination].
func (s *Sender) SetDestinations(universe uint16, destinations []string) error {

	dests := make([]net.UDPAddr, 0)
	for _, dest := range destinations {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dest, SACN_PORT))
		if err != nil {
			return err
		}
		dests = append(dests, *addr)
	}

	uni, exists := s.universes[universe]
	if exists {
		uni.destinations = dests
		return nil
	}
	return universeNotFoundError
}
