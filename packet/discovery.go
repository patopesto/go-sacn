package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// DiscoveryPacket is used to discover universes sent by sources on the network.
// It implements the [SACNPacket] interface.
type DiscoveryPacket struct {
	// inherits RootLayer
	RootLayer

	// Framing Layer
	FrameLength uint16
	FrameVector uint32
	SourceName  [64]byte
	reserved    [4]byte

	// Universe Discovery Layer
	UDLLength uint16
	UDLVector uint32
	Page      uint8       // The current page number. Multiple pages will be sent if the source sends more than 512 universes.
	Last      uint8       // The number of pages the source sends.
	Universes [512]uint16 // The sorted list of universes currently sent by the source.
}

// Returns a new [DiscoveryPacket] with sensible defaults and empty Universes list.
func NewDiscoveryPacket() *DiscoveryPacket {
	return &DiscoveryPacket{ // Default packet with no data
		// Root Layer
		RootLayer: RootLayer{
			PreambleSize:        0x0010,
			PostambleSize:       0x0000,
			ACNPacketIdentifier: packetIdentifierE117,
			RootVector:          VECTOR_ROOT_E131_EXTENDED,
			RootLength:          0x7068,
		},

		// Framing Layer
		FrameVector: VECTOR_E131_EXTENDED_DISCOVERY,
		FrameLength: 0x7052,

		// Universe Discovery Layer
		UDLVector: VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST,
		UDLLength: 0x7008,
	}
}

// Returns the packet type
func (d *DiscoveryPacket) GetType() SACNPacketType {
	return PacketTypeDiscovery
}

// Returns the number of universes in the packet's Universes list.
func (d *DiscoveryPacket) GetNumUniverses() int {
	return int(d.UDLLength&0x0FFF-8) / 2
}

// Adds a universe at the end of the packet's Universes list,
func (d *DiscoveryPacket) AddUniverse(universe uint16) error {
	num := d.GetNumUniverses()
	if num >= 512 {
		return errors.New("Universe list is full, please create a new DiscoveryPacket with the next page")
	}
	d.Universes[num] = universe

	d.setNumUniverses(uint16(num + 1))
	return nil
}

// Sets the packet's Universes list. Overwrites the current list.
func (d *DiscoveryPacket) SetUniverses(universes []uint16) error {
	num := len(universes)
	if num > 512 {
		return errors.New("Universe list is too long, please create a new DiscoveryPacket with the next page universes[512:]")
	}
	copy(d.Universes[:], universes[:])

	d.setNumUniverses(uint16(num))
	return nil
}

func (d *DiscoveryPacket) setNumUniverses(num uint16) {
	d.UDLLength = 0x7000 | (num*2 + 8)
	d.FrameLength = d.UDLLength + 74
	d.RootLength = d.FrameLength + 38
}

// Returns the user-assigned source name defined in the packet as a string.
func (d *DiscoveryPacket) GetSourceName() string {
	name := string(d.SourceName[:])
	return strings.Trim(name, "\x00") // remove trailing zeros from array
}

// Sets the source name of the packet. Shall not be more than 64 characters.
func (d *DiscoveryPacket) SetSourceName(name string) error {
	if len(name) > 64 {
		return errors.New("Source name has to be < 64 bytes")
	}
	copy(d.SourceName[:], []byte(name))
	return nil
}

// Implements [encoding.BinaryUnmarshaler] for the [DiscoveryPacket].
func (d *DiscoveryPacket) UnmarshalBinary(b []byte) error {
	// Root layer
	err := d.RootLayer.unmarshal(b)
	if err != nil {
		return err
	}

	// Framing layer
	d.FrameLength = binary.BigEndian.Uint16(b[38:40])
	if d.FrameLength&0x0FFF > uint16(len(b)) {
		return errors.New(fmt.Sprintf("Incorrect packet size %d != %d", d.FrameLength&0x0FFF, len(b)))
	}
	d.FrameVector = binary.BigEndian.Uint32(b[40:44])
	copy(d.SourceName[:], b[44:108])

	// Universe Discovery Layer
	d.UDLLength = binary.BigEndian.Uint16(b[112:114])
	d.UDLVector = binary.BigEndian.Uint32(b[114:118])
	d.Page = b[118]
	d.Last = b[119]
	for i, j := 0, 120; j < len(b); i, j = i+1, j+2 {
		d.Universes[i] = binary.BigEndian.Uint16(b[j : j+2])
	}

	return d.validate()
}

// Implements [encoding.BinaryMarshaler] for the [DiscoveryPacket].
func (d *DiscoveryPacket) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, d); err != nil {
		return nil, err
	}
	buf.Truncate(int(120 + d.GetNumUniverses()*2)) // Truncate unused part of Universes array
	return buf.Bytes(), nil
}

func (d *DiscoveryPacket) validate() error {
	// Root layer (specifics to DataPacket)
	if d.RootVector != VECTOR_ROOT_E131_EXTENDED {
		return errors.New("Invalid Root Vector")
	}

	// Framing layer
	if d.FrameVector != VECTOR_E131_EXTENDED_DISCOVERY {
		return errors.New("Invalid Frame Vector")
	}

	// Universe Discovery Layerr
	if d.UDLVector != VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST {
		return errors.New("Invalid Discovery Vector")
	}
	if d.Page > d.Last {
		return errors.New("Current page > Last page")
	}

	return nil
}
