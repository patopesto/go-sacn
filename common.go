package sacn

import (
	"fmt"
	"net"
)

const (
	SACN_PORT                   = 5568
	DISCOVERY_UNIVERSE          = 64214
	UNIVERSE_DISCOVERY_INTERVAL = 10   // in seconds
	NETWORK_DATA_LOSS_TIMEOUT   = 2500 // in milliseconds
)

// Section 9.3 of spec
func universeToAddress(universe uint16) *net.UDPAddr {
	bytes := []byte{byte(universe >> 8), byte(universe & 0xFF)}
	ip := fmt.Sprintf("239.255.%v.%v", bytes[0], bytes[1])
	addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, SACN_PORT))
	return addr
}

// Section 6.7.2 of spec
func checkSequence(A uint8, B uint8) bool {
	var diff int8
	diff = int8(B) - int8(A)
	if diff <= 0 && diff > -20 { // Out-of-Order
		return false
	}
	return true
}
