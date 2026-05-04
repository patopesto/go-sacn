package sacn

import (
	"net"
	"sync"
	"testing"
	"time"

	"gitlab.com/patopest/go-sacn/packet"
)

// getTestInterface returns a multicast-capable IPv4 interface.
// It prefers non-loopback interfaces but falls back to loopback if none found.
func getTestInterface(t *testing.T) *net.Interface {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Failed to get network interfaces: %v", err)
	}

	// First, try to find a non-loopback multicast-capable interface with IPv4
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 &&
			iface.Flags&net.FlagMulticast != 0 &&
			iface.Flags&net.FlagLoopback == 0 {
			// Check if interface has an IPv4 address
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ipnet.IP.To4() != nil {
						return &iface
					}
				}
			}
		}
	}

	// Fall back to loopback
	names := []string{"lo0", "lo"}
	for _, name := range names {
		iface, err := net.InterfaceByName(name)
		if err == nil {
			return iface
		}
	}

	t.Skip("No suitable network interface found")
	return nil
}

func TestNewReceiver(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer func() {
		if receiver.stop != nil {
			receiver.Stop()
		} else {
			receiver.conn.Close()
		}
	}()

	if receiver.conn == nil {
		t.Errorf("Receiver conn should not be nil")
	}
	if receiver.itf != iface {
		t.Errorf("Receiver interface not set correctly")
	}
	if receiver.lastPackets == nil {
		t.Errorf("lastPackets map should be initialized")
	}
	if receiver.streamTerminated == nil {
		t.Errorf("streamTerminated map should be initialized")
	}
	if receiver.packetCallbacks == nil {
		t.Errorf("packetCallbacks map should be initialized")
	}
}

func TestReceiverStartStop(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}

	// Start receiver
	receiver.Start()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop receiver multiple times should not panic
	receiver.Stop()
	receiver.Stop()
	receiver.Stop()

	// Test passed if no panic occurred
}

func TestReceiverStop(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	// Stop receiver without starting it
	receiver.Stop()

	// Test passed if no panic occurred
}

func TestReceiverJoinLeaveUniverse(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer func() {
		if receiver.stop != nil {
			receiver.Stop()
		} else {
			receiver.conn.Close()
		}
	}()

	// Valid universe
	err = receiver.JoinUniverse(1)
	if err != nil {
		t.Fatalf("Failed to join universe 1: %v", err)
	}

	// Leave universe
	err = receiver.LeaveUniverse(1)
	if err != nil {
		t.Fatalf("Failed to leave universe 1: %v", err)
	}

	// Invalid universe (0)
	err = receiver.JoinUniverse(0)
	if err == nil {
		t.Errorf("Expected error for universe 0")
	}

	// Invalid universe (64001)
	err = receiver.JoinUniverse(64001)
	if err == nil {
		t.Errorf("Expected error for universe 64001")
	}

	// Discovery universe should be valid
	err = receiver.JoinUniverse(DISCOVERY_UNIVERSE)
	if err != nil {
		t.Errorf("Discovery universe should be valid: %v", err)
	}
	receiver.LeaveUniverse(DISCOVERY_UNIVERSE)
}

func TestReceiverPacketCallback(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	callbackCalled := false
	callback := func(p packet.SACNPacket, info PacketInfo) {
		callbackCalled = true
		wg.Done()
	}

	receiver.RegisterPacketCallback(packet.PacketTypeData, callback)

	if receiver.packetCallbacks[packet.PacketTypeData] == nil {
		t.Errorf("Callback should be registered")
	}

	// Trigger callback manually for testing
	receiver.packetCallbacks[packet.PacketTypeData](packet.NewDataPacket(), PacketInfo{})

	wg.Wait()
	if !callbackCalled {
		t.Errorf("Callback should have been called")
	}
}

func TestReceiverTerminationCallback(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	callbackCalled := false
	var receivedUniverse uint16
	callback := func(universe uint16) {
		callbackCalled = true
		receivedUniverse = universe
		wg.Done()
	}

	receiver.RegisterTerminationCallback(callback)

	if receiver.terminationCallback == nil {
		t.Errorf("Termination callback should be registered")
	}

	// Trigger termination callback manually
	receiver.terminateUniverse(42)

	wg.Wait()
	if !callbackCalled {
		t.Errorf("Termination callback should have been called")
	}
	if receivedUniverse != 42 {
		t.Errorf("Expected universe 42, got %d", receivedUniverse)
	}
}

func TestReceiverHandlePacket(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	dataReceived := false
	receiver.RegisterPacketCallback(packet.PacketTypeData, func(p packet.SACNPacket, info PacketInfo) {
		dataReceived = true
		wg.Done()
	})

	// Create a data packet
	p := packet.NewDataPacket()
	p.Universe = 1
	p.SetData([]byte{255, 255, 255})

	info := PacketInfo{
		Source: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5568},
		Mode:   PacketUnicast,
	}

	receiver.handlePacket(p, info)

	wg.Wait()
	if !dataReceived {
		t.Errorf("Data callback should have been called")
	}
}

func TestReceiverHandlePacketStreamTerminated(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	terminationCalled := false
	receiver.RegisterTerminationCallback(func(universe uint16) {
		terminationCalled = true
		wg.Done()
	})

	// Create a data packet with stream terminated
	p := packet.NewDataPacket()
	p.Universe = 1
	p.SetStreamTerminated(true)

	info := PacketInfo{
		Source: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5568},
		Mode:   PacketUnicast,
	}

	receiver.handlePacket(p, info)

	wg.Wait()
	if !terminationCalled {
		t.Errorf("Termination callback should have been called for stream terminated packet")
	}
}

func TestReceiverStoreLastPacket(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	p := packet.NewDataPacket()
	p.Universe = 1

	receiver.storeLastPacket(1, p)

	if _, ok := receiver.lastPackets[1]; !ok {
		t.Errorf("Packet should be stored for universe 1")
	}

	if receiver.streamTerminated[1] {
		t.Errorf("streamTerminated should be false after storing packet")
	}
}

func TestReceiverCheckTimeouts(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	terminationCalled := false
	receiver.RegisterTerminationCallback(func(universe uint16) {
		terminationCalled = true
		wg.Done()
	})

	// Store an old packet (manually set to be older than timeout)
	receiver.lastPackets[1] = networkPacket{
		ts:     time.Now().Add(-time.Millisecond * (NETWORK_DATA_LOSS_TIMEOUT + 100)),
		packet: packet.NewDataPacket(),
	}

	receiver.checkTimeouts()

	wg.Wait()
	if !terminationCalled {
		t.Errorf("Termination callback should have been called for timeout")
	}
}

func TestReceiverTerminateUniverse(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	terminationCalled := false
	receiver.RegisterTerminationCallback(func(universe uint16) {
		terminationCalled = true
		wg.Done()
	})

	receiver.terminateUniverse(1)

	wg.Wait()
	if !terminationCalled {
		t.Errorf("Termination callback should have been called")
	}

	if !receiver.streamTerminated[1] {
		t.Errorf("streamTerminated should be true after termination")
	}

	// Second termination should not call callback (already terminated)
	receiver.terminateUniverse(1)
}

func TestReceiverHandlePacketWithSyncAddress(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	// Create a data packet with sync address
	p := packet.NewDataPacket()
	p.Universe = 1
	p.SyncAddress = 100
	p.SetData([]byte{255, 255, 255})

	info := PacketInfo{
		Source: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5568},
		Mode:   PacketUnicast,
	}

	receiver.handlePacket(p, info)

	// Packet should be stored
	if _, ok := receiver.lastPackets[1]; !ok {
		t.Errorf("Packet should be stored for universe 1")
	}
}

func TestReceiverHandleSyncPacket(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	syncReceived := false
	receiver.RegisterPacketCallback(packet.PacketTypeSync, func(p packet.SACNPacket, info PacketInfo) {
		syncReceived = true
		wg.Done()
	})

	// Create a sync packet
	p := packet.NewSyncPacket()
	p.SyncAddress = 1

	info := PacketInfo{
		Source: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5568},
		Mode:   PacketUnicast,
	}

	receiver.handlePacket(p, info)

	wg.Wait()
	if !syncReceived {
		t.Errorf("Sync callback should have been called")
	}

	// Sync packet should be stored at sync address, not universe 0
	if _, ok := receiver.lastPackets[1]; !ok {
		t.Errorf("Sync packet should be stored at sync address")
	}
}

func TestReceiverLeaveUniverseError(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer func() {
		if receiver.stop != nil {
			receiver.Stop()
		} else {
			receiver.conn.Close()
		}
	}()

	// Try to leave a universe that was never joined
	// This may or may not return an error depending on the OS
	err = receiver.LeaveUniverse(9999)
	// We just verify it doesn't panic - the actual error behavior
	// depends on the underlying network implementation
	_ = err
}

func TestReceiverRegisterCallbackOverwrite(t *testing.T) {
	iface := getTestInterface(t)

	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	callback1Called := make(chan bool, 1)
	callback2Called := make(chan bool, 1)

	// Register first callback
	receiver.RegisterPacketCallback(packet.PacketTypeData, func(p packet.SACNPacket, info PacketInfo) {
		callback1Called <- true
	})

	// Verify first callback is registered
	if receiver.packetCallbacks[packet.PacketTypeData] == nil {
		t.Errorf("First callback should be registered")
	}

	// Register second callback (should overwrite first)
	receiver.RegisterPacketCallback(packet.PacketTypeData, func(p packet.SACNPacket, info PacketInfo) {
		callback2Called <- true
	})

	// Verify second callback is registered
	if receiver.packetCallbacks[packet.PacketTypeData] == nil {
		t.Errorf("Second callback should be registered")
	}

	// Create a data packet
	p := packet.NewDataPacket()
	p.Universe = 1
	p.SetData([]byte{255, 255, 255})

	info := PacketInfo{
		Source: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5568},
		Mode:   PacketUnicast,
	}

	receiver.handlePacket(p, info)

	// Verify second callback was called, not first
	select {
	case <-callback1Called:
		t.Errorf("First callback should NOT have been called (was overwritten)")
	case <-callback2Called:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Second callback should have been called")
	}

	// Verify first callback was not called
	select {
	case <-callback1Called:
		t.Errorf("First callback should NOT have been called (was overwritten)")
	default:
		// Expected - no message on channel
	}
}
