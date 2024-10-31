package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/spf13/cast"
)

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
	return &DataPacket{
		// Root Layer
		RootLayer: RootLayer{
			PreambleSize:        0x0010,
			PostambleSize:       0x0000,
			ACNPacketIdentifier: packetIdentifierE117,
			RootVector:          VECTOR_E131_DATA_PACKET,
		},

		// Framing Layer
		FrameVector: VECTOR_E131_DATA_PACKET,
		Priority:    100,

		// Data Layer
		DMPVector:        VECTOR_DMP_SET_PROPERTY,
		Format:           0xA1,
		PropertyAddress:  0x0000,
		AddressIncrement: 0x0001,
	}
}

func (d *DataPacket) GetType() SACNPacketType {
	return PacketTypeData
}

// ---- Helpers for specific packet data ----
func (d *DataPacket) GetData() []byte {
	return d.Data[1:]
}

func (d *DataPacket) SetData(data []byte) {
	length := uint16(len(data))
	if length > 512 {
		data = data[:512]
		length = 512
	}

	copy(d.Data[1:], data[:])
	d.computeLength(length)
}

func (d *DataPacket) computeLength(data_length uint16) {
	length := 126 + data_length

	d.RootLength = 0x7000 | (length - 16)
	d.FrameLength = 0x7000 | (length - 38)
	d.DMPLength = 0x7000 | (length - 115)
	d.Length = data_length + 1
}

func (d *DataPacket) SetStartCode(code uint8) {
	d.Data[0] = code
}

func (d *DataPacket) IsPreviewData() bool {
	return cast.ToBool(d.Options >> 6)
}

func (d *DataPacket) SetPreviewData(value bool) {
	d.Options |= cast.ToUint8(value) << 6
}

func (d *DataPacket) IsStreamTerminated() bool {
	return cast.ToBool(d.Options >> 5)
}

func (d *DataPacket) SetStreamTerminated(value bool) {
	d.Options |= cast.ToUint8(value) << 5
}

func (d *DataPacket) IsForceSynchronisation() bool {
	return cast.ToBool(d.Options >> 4)
}

func (d *DataPacket) SetForceSynchronisation(value bool) {
	d.Options |= cast.ToUint8(value) << 4
}

func (d *DataPacket) UnmarshalBinary(b []byte) error {
	// Root layer
	err := d.RootLayer.unmarshal(b)
	if err != nil {
		return err
	}

	// Framing layer
	d.FrameLength = binary.BigEndian.Uint16(b[38:40])
	if d.FrameLength&0x0FFF > uint16(len(b)-38) {
		return errors.New(fmt.Sprintf("Incorrect packet size %d != %d", d.FrameLength&0x0FFF, len(b)-38))
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
	if d.DMPLength&0x0FFF > uint16(len(b)-115) {
		return errors.New(fmt.Sprintf("Incorrect packet size %d != %d", d.DMPLength&0x0FFF, len(b)-115))
	}
	d.DMPVector = b[117]
	d.Format = b[118]
	d.PropertyAddress = binary.BigEndian.Uint16(b[119:121])
	d.AddressIncrement = binary.BigEndian.Uint16(b[121:123])
	d.Length = binary.BigEndian.Uint16(b[123:125])
	if d.Length&0x0FFF > uint16(len(b)-125) {
		return errors.New(fmt.Sprintf("Incorrect packet size %d != %d", d.Length&0x0FFF, len(b)-126))
	}
	copy(d.Data[:], b[125:])

	return d.validate()
}

func (d *DataPacket) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, d); err != nil {
		return nil, err
	}
	buf.Truncate(int(125 + d.Length)) // Truncate unused part of Data array
	return buf.Bytes(), nil
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
