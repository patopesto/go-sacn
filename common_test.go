package sacn

import (
	// "fmt"
	"testing"
)

func TestuniverseToAddress(t *testing.T) {
	tests := []struct {
		universe uint16
		expected string
	}{
		{
			universe: 1,
			expected: "239.255.0.1",
		},
		{
			universe: 255,
			expected: "239.255.0.255",
		},
		{
			universe: 64214,
			expected: "239.255.250.214",
		},
	}

	for _, tt := range tests {
		addr := universeToAddress(tt.universe)

		if addr.Port != SACN_PORT {
			t.Fatalf("Wrong port %d != %d", addr.Port, SACN_PORT)
		}
		if !addr.IP.IsMulticast() {
			t.Fatalf("Addr is not multicast")
		}
		if addr.IP.String() != tt.expected {
			t.Fatalf("IP %v != %s", addr.IP.String(), tt.expected)
		}
	}
}

func TestSequence(t *testing.T) {
	tests := []struct {
		A        uint8
		B        uint8
		expected bool
	}{
		{
			A:        100,
			B:        110,
			expected: true,
		},
		{
			A:        255,
			B:        1,
			expected: true,
		},
		{
			A:        101,
			B:        100,
			expected: false,
		},
		{
			A:        100,
			B:        81,
			expected: false,
		},
		{
			A:        100,
			B:        80,
			expected: true,
		},
	}

	for _, tt := range tests {
		value := checkSequence(tt.A, tt.B)

		if value != tt.expected {
			t.Fatalf("Unexpected sequence check (A=%v, B=%v): %v != %v", tt.A, tt.B, value, tt.expected)
		}
	}
}
