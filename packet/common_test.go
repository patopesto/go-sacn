package packet

import (
	"testing"
)

func TestRootLayerUnmarshalTooShort(t *testing.T) {
	r := RootLayer{}
	err := r.unmarshal([]byte{0x00, 0x10, 0x00, 0x00}) // Only 4 bytes
	if err == nil {
		t.Fatalf("expected error for too short buffer")
	}
	if err.Error() != "Root layer length incorrect" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestRootLayerValidatePreambleSize(t *testing.T) {
	r := RootLayer{}
	// Create a valid buffer with incorrect preamble size
	b := make([]byte, 38)
	b[0] = 0x00
	b[1] = 0x99 // Wrong preamble size (should be 0x0010)
	b[2] = 0x00
	b[3] = 0x00
	copy(b[4:16], packetIdentifierE117[:])

	err := r.unmarshal(b)
	if err == nil {
		t.Fatalf("expected error for incorrect preamble size")
	}
	if err.Error() != "Incorrect Preamble size in Root Layer" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestRootLayerValidatePostambleSize(t *testing.T) {
	r := RootLayer{}
	// Create a valid buffer with incorrect postamble size
	b := make([]byte, 38)
	b[0] = 0x00
	b[1] = 0x10 // Correct preamble size
	b[2] = 0x00
	b[3] = 0x99 // Wrong postamble size (should be 0x0000)
	copy(b[4:16], packetIdentifierE117[:])

	err := r.unmarshal(b)
	if err == nil {
		t.Fatalf("expected error for incorrect postamble size")
	}
	if err.Error() != "Incorrect Postamble size in Root Layer" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestRootLayerValidateACNPacketIdentifier(t *testing.T) {
	r := RootLayer{}
	// Create a valid buffer with incorrect ACN packet identifier
	b := make([]byte, 38)
	b[0] = 0x00
	b[1] = 0x10 // Correct preamble size
	b[2] = 0x00
	b[3] = 0x00 // Correct postamble size
	// Fill ACN packet identifier with wrong data
	for i := 4; i < 16; i++ {
		b[i] = 0xFF
	}

	err := r.unmarshal(b)
	if err == nil {
		t.Fatalf("expected error for incorrect ACN packet identifier")
	}
	if err.Error() != "Incorrect ACN Packet Identifier" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestUnmarshalUnhandledRootVector(t *testing.T) {
	// Create a data packet with invalid root vector
	p := NewDataPacket()
	b, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal packet: %v", err)
	}

	// Modify the root vector to an unhandled value
	// Root vector is at bytes 18-22
	b[18] = 0x00
	b[19] = 0x00
	b[20] = 0x99 // Invalid root vector
	b[21] = 0x99

	_, err = Unmarshal(b)
	if err == nil {
		t.Fatalf("expected error for unhandled root vector")
	}
	if err.Error() != "Unhandled packet type" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestUnmarshalUnhandledFrameVector(t *testing.T) {
	// Create a sync packet which uses extended root vector
	p := NewSyncPacket()
	b, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal packet: %v", err)
	}

	// Modify the frame vector to an unhandled value
	// Frame vector is at bytes 40-44
	b[40] = 0x00
	b[41] = 0x00
	b[42] = 0x99 // Invalid frame vector
	b[43] = 0x99

	_, err = Unmarshal(b)
	if err == nil {
		t.Fatalf("expected error for unhandled frame vector")
	}
	if err.Error() != "Unhandled packet type" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestUnmarshalRootLayerError(t *testing.T) {
	// Test with too short buffer
	_, err := Unmarshal([]byte{0x00, 0x10, 0x00, 0x00})
	if err == nil {
		t.Fatalf("expected error for too short buffer")
	}
	if err.Error() != "Root layer length incorrect" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestMarshal(t *testing.T) {
	p := NewDataPacket()
	p.SetData([]byte{0x01, 0x02, 0x03})

	b, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify it matches direct MarshalBinary
	direct, err := p.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	if len(b) != len(direct) {
		t.Fatalf("Marshal and MarshalBinary returned different lengths: %d vs %d", len(b), len(direct))
	}

	for i := range b {
		if b[i] != direct[i] {
			t.Fatalf("byte %d differs: Marshal=0x%x, MarshalBinary=0x%x", i, b[i], direct[i])
		}
	}
}
