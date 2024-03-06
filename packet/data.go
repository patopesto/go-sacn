package packet

import (
	// "bytes"
	// "encoding"
	"encoding/binary"
	"errors"
)

// var _ SACNPacket = &DataPacket{}

type DataPacket struct {
	// Inherit RootLayer
	RootLayer

	// Framing layer
	FrameLength uint16
	FrameVector uint32
	SourceName  [64]byte
	Priority    uint8
	SyncAddress uint16
	Sequence    uint8
	Options     uint8
	Universe    uint16

	// DMP Layer
	DMPLength        uint16
	DMPVector        uint8
	Format           uint8
	PropertyAddress  uint16
	AddressIncrement uint16
	Length           uint16
	Data             [513]byte
}

func NewDataPacket() *DataPacket {
	// TODO: fill up default values
	return &DataPacket{}
}

func (d *DataPacket) GetType() SACNPacketType {
	return PacketTypeData
}

func (d *DataPacket) UnmarshalBinary(b []byte) error {
	// Root layer
	err := d.RootLayer.unmarshal(b)
	if err != nil {
		return err
	}

	// Framing layer
	d.FrameLength = binary.BigEndian.Uint16(b[38:40])
	if d.FrameLength & 0x0FFF > uint16(len(b)) {
		return errors.New("Incorrect packet size")
	}
	d.FrameVector = binary.BigEndian.Uint32(b[40:44])
	copy(d.SourceName[:], b[44:108])
	d.Priority = b[108]
	d.SyncAddress = binary.BigEndian.Uint16(b[109:111])
	d.Sequence = b[111]
	d.Options = b[112]
	d.Universe = binary.BigEndian.Uint16(b[113:115])

	// DMP Layer
	d.DMPLength = binary.BigEndian.Uint16(b[115:117])
	d.DMPVector = b[117]
	d.Format = b[118]
	d.PropertyAddress = binary.BigEndian.Uint16(b[119:121])
	d.AddressIncrement = binary.BigEndian.Uint16(b[121:123])
	d.Length = binary.BigEndian.Uint16(b[123:125])
	copy(d.Data[:], b[125:])

	return d.validate()
}

func (d *DataPacket) validate() error {
	// Root layer (specifics to DataPacket)
	if d.RootVector != VECTOR_ROOT_E131_DATA {
		return errors.New("Invalid Root Vector")
	}

	// Framing layer
	if d.FrameVector != VECTOR_E131_DATA_PACKET {
		return errors.New("Invalid Frame Vector")
	}

	// DMP layer
	if d.DMPVector != VECTOR_DMP_SET_PROPERTY {
		return errors.New("Invalid DMP Vector")
	}
	// Statics as defined in Section 7.
	if d.Format != 0xA1 || d.PropertyAddress != 0 || d.AddressIncrement != 1 {
		return errors.New("Invalid DMP Formats")
	}
	return nil
}
