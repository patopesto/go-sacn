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
		// Test direct unmarshaler
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

		// Test global unmarshaler
		d, err := Unmarshal(tt.b[:])
		if tt.err != err {
			t.Fatalf("unexpected error on \"%s\":\n- want: %v\n-  got: %v", tt.name, tt.err, err)
		}
		if d.GetType() != PacketTypeSync {
			t.Fatalf("unexpected packet type returned on \"%s\":\n- want: %v\n-  got: %v", tt.name, PacketTypeSync, d.GetType())
		}
	}
}

func TestSyncPacketMarshal(t *testing.T) {
	for _, tt := range sync_tests {
		// Test direct marshaler
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

		// Test global marshaler
		d, err := Marshal(&tt.p)
		if tt.err != err {
			t.Fatalf("unexpected error on \"%s\":\n- want: %v\n-  got: %v", tt.name, tt.err, err)
		}
		if !bytes.Equal(tt.b[:], d) {
			t.Fatalf("unexpected bytes on \"%s\":\n- want: [%#v] len:%d\n-  got: [%#v] len:%d", tt.name, tt.b, len(tt.b), d, len(d))
		}
	}
}

func TestNewSyncPacket(t *testing.T) {
	p := NewSyncPacket()

	// Verify default values
	if want, got := uint16(0x0010), p.PreambleSize; want != got {
		t.Fatalf("unexpected PreambleSize:\n- want: 0x%x\n-  got: 0x%x", want, got)
	}
	if want, got := uint16(0x0000), p.PostambleSize; want != got {
		t.Fatalf("unexpected PostambleSize:\n- want: 0x%x\n-  got: 0x%x", want, got)
	}
	if want, got := uint32(VECTOR_ROOT_E131_EXTENDED), p.RootVector; want != got {
		t.Fatalf("unexpected RootVector:\n- want: 0x%x\n-  got: 0x%x", want, got)
	}
	if want, got := uint32(VECTOR_E131_EXTENDED_SYNCHRONIZATION), p.FrameVector; want != got {
		t.Fatalf("unexpected FrameVector:\n- want: 0x%x\n-  got: 0x%x", want, got)
	}
	if want, got := uint16(0x7021), p.RootLength; want != got {
		t.Fatalf("unexpected RootLength:\n- want: 0x%x\n-  got: 0x%x", want, got)
	}
	if want, got := uint16(0x700B), p.FrameLength; want != got {
		t.Fatalf("unexpected FrameLength:\n- want: 0x%x\n-  got: 0x%x", want, got)
	}
	if want, got := uint16(0), p.SyncAddress; want != got {
		t.Fatalf("unexpected SyncAddress:\n- want: %d\n-  got: %d", want, got)
	}
	if want, got := PacketTypeSync, p.GetType(); want != got {
		t.Fatalf("unexpected packet type:\n- want: %d\n-  got: %d", want, got)
	}
}

func TestSyncPacketValidateErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*SyncPacket)
		wantErr string
	}{
		{
			name: "Invalid Root Vector",
			modify: func(p *SyncPacket) {
				p.RootVector = 0x9999
			},
			wantErr: "Invalid Root Vector",
		},
		{
			name: "Invalid Frame Vector",
			modify: func(p *SyncPacket) {
				p.RootVector = VECTOR_ROOT_E131_EXTENDED // reset to valid
				p.FrameVector = 0x9999
			},
			wantErr: "Invalid Frame Vector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewSyncPacket()
			tt.modify(p)

			err := p.validate()
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("unexpected error message:\n- want: %s\n-  got: %s", tt.wantErr, err.Error())
			}
		})
	}
}

func TestSyncPacketUnmarshalBinaryErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		wantErr string
	}{
		{
			name:    "Empty buffer",
			bytes:   []byte{},
			wantErr: "Root layer length incorrect",
		},
		{
			name: "Frame length mismatch",
			bytes: func() []byte {
				p := NewSyncPacket()
				b, _ := p.MarshalBinary()
				// Corrupt frame length (at bytes 38:40)
				b[38] = 0xFF
				b[39] = 0xFF
				return b
			}(),
			wantErr: "Incorrect packet size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p SyncPacket
			err := p.UnmarshalBinary(tt.bytes)
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}
