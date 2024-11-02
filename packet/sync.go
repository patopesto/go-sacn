package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// SyncPacket is used to synchronize multiple universes. See Section 11 of ANSI E1.31—2018.
// It implements the [SACNPacket] interface.
// The SyncAddress set in this packet shall be the same as the universe it is sent/received on.
type SyncPacket struct {
	// inherits RootLayer
	RootLayer

	// Framing layer
	FrameLength uint16
	FrameVector uint32
	Sequence    uint8
	SyncAddress uint16 // sync universe this packet is directed to.
	reserved    [2]byte
}

// Returns a new [SyncPacket] with sensible defaults and empty SyncAddress.
func NewSyncPacket() *SyncPacket {
	return &SyncPacket{
		// Root Layer
		RootLayer: RootLayer{
			PreambleSize:        0x0010,
			PostambleSize:       0x0000,
			ACNPacketIdentifier: packetIdentifierE117,
			RootVector:          VECTOR_ROOT_E131_EXTENDED,
			RootLength:          0x7021, // always fixed
		},

		// Framing Layer
		FrameVector: VECTOR_E131_EXTENDED_SYNCHRONIZATION,
		FrameLength: 0x700B, // always fixed
		SyncAddress: 0,
	}
}

// Returns the packet type
func (d *SyncPacket) GetType() SACNPacketType {
	return PacketTypeSync
}

// Implements [encoding.BinaryUnmarshaler] for the [SyncPacket].
func (d *SyncPacket) UnmarshalBinary(b []byte) error {
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
	d.Sequence = b[44]
	d.SyncAddress = binary.BigEndian.Uint16(b[45:47])

	return d.validate()
}

// Implements [encoding.BinaryMarshaler] for the [SyncPacket].
func (d *SyncPacket) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (d *SyncPacket) validate() error {
	// Root layer (specifics to SyncPacket)
	if d.RootVector != VECTOR_ROOT_E131_EXTENDED {
		return errors.New("Invalid Root Vector")
	}

	// Framing layer
	if d.FrameVector != VECTOR_E131_EXTENDED_SYNCHRONIZATION {
		return errors.New("Invalid Frame Vector")
	}

	return nil
}
