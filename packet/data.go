package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

// DataPacket is used to send a universe's DMX512-A data over the network. Most commonly used packet.
// It implements the [SACNPacket] interface.
type DataPacket struct {
	// inherits RootLayer
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

// Returns a new [DataPacket] with sensible defaults and empty DMX data.
// Use the Get and Set helpers to fill packet with data before sending it out.
func NewDataPacket() *DataPacket {
	return &DataPacket{ // Default packet with no data
		// Root Layer
		RootLayer: RootLayer{
			PreambleSize:        0x0010,
			PostambleSize:       0x0000,
			ACNPacketIdentifier: packetIdentifierE117,
			RootVector:          VECTOR_ROOT_E131_DATA,
			RootLength:          0x707D,
		},

		// Framing Layer
		FrameVector: VECTOR_E131_DATA_PACKET,
		FrameLength: 0x7057,
		Priority:    100,

		// Data Layer
		DMPVector:        VECTOR_DMP_SET_PROPERTY,
		DMPLength:        0x700A,
		Format:           0xA1,
		PropertyAddress:  0x0000,
		AddressIncrement: 0x0001,
		Length:           0x0000,
	}
}

// Returns the packet type
func (d *DataPacket) GetType() SACNPacketType {
	return PacketTypeData
}

// Returns the DMX512-A data of the packet (up to 512 bytes). Does not include the Start Code (byte 0 of a DMX packet).
func (d *DataPacket) GetData() []byte {
	return d.Data[1:]
}

// Set DMX512-A data. Does not include the Start Code (byte 0 of a DMX packet).
// Overwrites any existing data in the packet. Shall not be more than 512 bytes.
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
	length := 125 + data_length + 1 // +1 for zero start code

	d.Length = data_length + 1
	d.RootLength = 0x7000 | (length - 16)
	d.FrameLength = 0x7000 | (length - 38)
	d.DMPLength = 0x7000 | (length - 115)
}

// Returns the Start Code (byte 0 of a DMX packet). Value should be 0 for normal DMX512-A operations.
func (d *DataPacket) GetStartCode() uint8 {
	return d.Data[0]
}

// Sets the Start Code. (byte 0 of a DMX packet).
func (d *DataPacket) SetStartCode(code uint8) {
	d.Data[0] = code
}

// Returns the user-assigned source name defined in the packet as a string.
func (d *DataPacket) GetSourceName() string {
	name := string(d.SourceName[:])
	return strings.Trim(name, "\x00") // remove trailing zeros from array
}

// Sets the source name of the packet. Shall not be more than 64 characters.
func (d *DataPacket) SetSourceName(name string) error {
	if len(name) > 64 {
		return errors.New("Source name has to be < 64 bytes")
	}
	copy(d.SourceName[:], []byte(name))
	return nil
}

// Returns true if the Preview_Data (bit 7) is set in the Options of the packet. See Section 6.2.6 of ANSI E1.31—2018
func (d *DataPacket) IsPreviewData() bool {
	return cast.ToBool(d.Options >> 7)
}

// Sets the Preview_Data (bit 7) in the Options of the packet. See Section 6.2.6 of ANSI E1.31—2018
func (d *DataPacket) SetPreviewData(value bool) {
	d.Options |= cast.ToUint8(value) << 7
}

// Returns true if the Stream_Terminated (bit 6) is set in the Options of the packet. See Section 6.2.6 of ANSI E1.31—2018
func (d *DataPacket) IsStreamTerminated() bool {
	return cast.ToBool(d.Options >> 6)
}

// Sets the Stream_Terminated (bit 6) in the Options of the packet. See Section 6.2.6 of ANSI E1.31—2018
func (d *DataPacket) SetStreamTerminated(value bool) {
	d.Options |= cast.ToUint8(value) << 6
}

// Returns true if the Force_Synchronisation (bit 5) is set in the Options of the packet. See Section 6.2.6 of ANSI E1.31—2018
func (d *DataPacket) IsForceSynchronisation() bool {
	return cast.ToBool(d.Options >> 5)
}

// Sets the Force_Synchronisation (bit 5) in the Options of the packet. See Section 6.2.6 of ANSI E1.31—2018
func (d *DataPacket) SetForceSynchronisation(value bool) {
	d.Options |= cast.ToUint8(value) << 5
}

// Implements [encoding.BinaryUnmarshaler] for the [DataPacket].
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

// Implements [encoding.BinaryMarshaler] for the [DataPacket].
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
