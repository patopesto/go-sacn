package packet

import (
	"bytes"
	"reflect"
	"testing"
)

var discovery_tests = []struct {
	name string
	p    DiscoveryPacket
	b    []byte
	err  error
}{
	{
		name: "Discovery packet with 2 universes", // With 2 universes on page 3
		p: DiscoveryPacket{
			RootLayer: RootLayer{
				PreambleSize:        0x0010,
				PostambleSize:       0x0000,
				ACNPacketIdentifier: packetIdentifierE117,
				RootLength:          0x706c,
				RootVector:          VECTOR_ROOT_E131_EXTENDED,
				CID:                 [16]byte{0xef, 0x07, 0xc8, 0xdd, 0x00, 0x64, 0x44, 0x01, 0xa3, 0xa2, 0x45, 0x9e, 0xf8, 0xe6, 0x14, 0x3e},
			},
			FrameLength: 0x7056,
			FrameVector: VECTOR_E131_EXTENDED_DISCOVERY,
			SourceName:  [64]byte{},

			UDLLength: 0x700c,
			UDLVector: VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST,
			Page:      3,
			Last:      5,
			Universes: [512]uint16{0x01, 0x64},
		},
		b: []byte{
			0x00, 0x10, 0x00, 0x00, 0x41, 0x53, 0x43, 0x2d, 0x45, 0x31, 0x2e, 0x31, 0x37, 0x00, 0x00, 0x00, 0x70, 0x6c,
			0x00, 0x00, 0x00, 0x08, 0xef, 0x07, 0xc8, 0xdd, 0x00, 0x64, 0x44, 0x01, 0xa3, 0xa2, 0x45, 0x9e, 0xf8, 0xe6,
			0x14, 0x3e, 0x70, 0x56, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x70, 0x0c, 0x00, 0x00, 0x00, 0x01, 0x03, 0x05, 0x00, 0x01, 0x00, 0x64,
		},
		err: nil,
	},
	{
		name: "Discovery packet with empty list",
		p: DiscoveryPacket{
			RootLayer: RootLayer{
				PreambleSize:        0x0010,
				PostambleSize:       0x0000,
				ACNPacketIdentifier: packetIdentifierE117,
				RootLength:          0x7068,
				RootVector:          VECTOR_ROOT_E131_EXTENDED,
				CID:                 [16]byte{0xef, 0x07, 0xc8, 0xdd, 0x00, 0x64, 0x44, 0x01, 0xa3, 0xa2, 0x45, 0x9e, 0xf8, 0xe6, 0x14, 0x3e},
			},
			FrameLength: 0x7052,
			FrameVector: VECTOR_E131_EXTENDED_DISCOVERY,
			SourceName:  [64]byte{},

			UDLLength: 0x7008,
			UDLVector: VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST,
			Page:      0,
			Last:      0,
			Universes: [512]uint16{},
		},
		b: []byte{
			0x00, 0x10, 0x00, 0x00, 0x41, 0x53, 0x43, 0x2d, 0x45, 0x31, 0x2e, 0x31, 0x37, 0x00, 0x00, 0x00, 0x70, 0x68,
			0x00, 0x00, 0x00, 0x08, 0xef, 0x07, 0xc8, 0xdd, 0x00, 0x64, 0x44, 0x01, 0xa3, 0xa2, 0x45, 0x9e, 0xf8, 0xe6,
			0x14, 0x3e, 0x70, 0x52, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x70, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		},
		err: nil,
	},
}

func TestDiscoveryPacketUnmarshal(t *testing.T) {
	for _, tt := range discovery_tests {
		var p DiscoveryPacket
		err := p.UnmarshalBinary(tt.b[:])

		if tt.err != err {
			t.Fatalf("unexpected error on \"%s\":\n- want: %v\n-  got: %v", tt.name, tt.err, err)
		}
		if err != nil {
			return
		}

		if !reflect.DeepEqual(tt.p, p) {
			t.Fatalf("unexpected bytes on \"%s\":\n- want: [%#v]\n-  got: [%#v]", tt.name, tt.p, p)
		}
	}
}

func TestDiscoveryPacketMarshal(t *testing.T) {
	for _, tt := range discovery_tests {
		b, err := tt.p.MarshalBinary()

		if tt.err != err {
			t.Fatalf("unexpected error on \"%s\":\n- want: %v\n-  got: %v", tt.name, tt.err, err)
		}
		if err != nil {
			return
		}

		if !bytes.Equal(tt.b[:], b) {
			t.Fatalf("unexpected bytes on \"%s\":\n- want: [%#v] len:%d\n-  got: [%#v] len:%d", tt.name, tt.b, len(tt.b), b, len(b))
		}
	}
}
