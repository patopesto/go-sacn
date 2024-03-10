package packet

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	// "fmt"
)

// From ANSI E1.31—2018 Appendix A.
const (
	VECTOR_ROOT_E131_DATA     = 0x0004
	VECTOR_ROOT_E131_EXTENDED = 0x0008

	VECTOR_E131_DATA_PACKET              = 0x0002
	VECTOR_E131_EXTENDED_SYNCHRONIZATION = 0x0001
	VECTOR_E131_EXTENDED_DISCOVERY       = 0x0002

	VECTOR_DMP_SET_PROPERTY                 = 0x02
	VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST = 0x0001
)

var packetIdentifierE117 = [12]byte{0x41, 0x53, 0x43, 0x2d, 0x45, 0x31, 0x2e, 0x31, 0x37, 0x00, 0x00, 0x00}

type SACNPacketType int

const (
	PacketTypeData SACNPacketType = iota
	PacketTypeSync
	PacketTypeDiscovery
)

type SACNPacket interface {
	// encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	validate() error
	GetType() SACNPacketType
}

type RootLayer struct {
	PreambleSize        uint16
	PostambleSize       uint16
	ACNPacketIdentifier [12]byte
	RootLength          uint16
	RootVector          uint32
	CID                 [16]byte
}

func (r *RootLayer) unmarshal(b []byte) error {
	if len(b) < 38 {
		return errors.New("Root layer length incorrect")
	}

	r.PreambleSize = binary.BigEndian.Uint16(b[0:2])
	r.PostambleSize = binary.BigEndian.Uint16(b[2:4])
	r.ACNPacketIdentifier = [12]byte{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15]}
	r.RootLength = binary.BigEndian.Uint16(b[16:18])
	r.RootVector = binary.BigEndian.Uint32(b[18:22])
	copy(r.CID[:16], b[22:38])

	return r.validate()
}

func (r *RootLayer) validate() error {
	if r.PreambleSize != 0x0010 {
		return errors.New("Incorrect Preamble size in Root Layer")
	}
	if r.PostambleSize != 0x0000 {
		return errors.New("Incorrect Postamble size in Root Layer")
	}
	if !bytes.Equal(r.ACNPacketIdentifier[:], packetIdentifierE117[:]) {
		return errors.New("Incorrect ACN Packet Identifier")
	}
	return nil
}

func Unmarshal(b []byte) (p SACNPacket, err error) {
	r := RootLayer{}
	err = r.unmarshal(b)
	if err != nil {
		return
	}

	errUnhandled := errors.New("Unhandled packet type")
	frameVector := binary.BigEndian.Uint32(b[40:44])

	switch r.RootVector {
	case VECTOR_ROOT_E131_DATA:
		// fmt.Println("Data packet");
		p = &DataPacket{}
	case VECTOR_ROOT_E131_EXTENDED:
		switch frameVector {
		case VECTOR_E131_EXTENDED_SYNCHRONIZATION:
			p = &SyncPacket{}
		case VECTOR_E131_EXTENDED_DISCOVERY:
			p = &DiscoveryPacket{}
		default:
			return nil, errUnhandled
		}
	default:
		return nil, errUnhandled
	}

	err = p.UnmarshalBinary(b)
	return
}
