package packet

import (
	"bytes"
	"reflect"
	"testing"
)

var sync_tests = []struct {
	name string
	p    SyncPacket
	b    [49]byte
	err  error
}{
	{
		name: "Sync packet", // Example from Appendix B.1 of spec document
		p: SyncPacket{
			RootLayer: RootLayer{
				PreambleSize:        0x0010,
				PostambleSize:       0x0000,
				ACNPacketIdentifier: packetIdentifierE117,
				RootLength:          0x7021,
				RootVector:          VECTOR_ROOT_E131_EXTENDED,
				CID:                 [16]byte{0xef, 0x07, 0xc8, 0xdd, 0x00, 0x64, 0x44, 0x01, 0xa3, 0xa2, 0x45, 0x9e, 0xf8, 0xe6, 0x14, 0x3e},
			},
			FrameLength: 0x700b,
			FrameVector: VECTOR_E131_EXTENDED_SYNCHRONIZATION,
			Sequence:    167,
			SyncAddress: 7962,
		},
		b: [49]byte{
			0x00, 0x10, 0x00, 0x00, 0x41, 0x53, 0x43, 0x2d, 0x45, 0x31, 0x2e, 0x31, 0x37, 0x00, 0x00, 0x00, 0x70, 0x21,
			0x00, 0x00, 0x00, 0x08, 0xef, 0x07, 0xc8, 0xdd, 0x00, 0x64, 0x44, 0x01, 0xa3, 0xa2, 0x45, 0x9e, 0xf8, 0xe6,
			0x14, 0x3e, 0x70, 0x0b, 0x00, 0x00, 0x00, 0x01, 0xa7, 0x1f, 0x1a, 0x00, 0x00,
		},
		err: nil,
	},
}

func TestSyncPacketUnmarshal(t *testing.T) {
	for _, tt := range sync_tests {
		var p SyncPacket
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

func TestSyncPacketMarshal(t *testing.T) {
	for _, tt := range sync_tests {
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
