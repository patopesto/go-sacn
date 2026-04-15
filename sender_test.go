package sacn

import (
	"bytes"
	"log"
	"testing"
	"time"

	"gitlab.com/patopest/go-sacn/packet"
)

func TestNewSender(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
		Logger:     log.Default(),
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	if sender.cid != options.CID {
		t.Errorf("CID not set correctly")
	}
	if sender.sourceName != options.SourceName {
		t.Errorf("SourceName not set correctly")
	}
}

func TestNewSenderDefaultCID(t *testing.T) {
	options := &SenderOptions{
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	// CID should be auto-generated (not all zeros)
	if bytes.Equal(sender.cid[:], make([]byte, 16)) {
		t.Errorf("CID should be auto-generated")
	}
}

func TestNewSenderDefaultSourceName(t *testing.T) {
	options := &SenderOptions{
		CID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	if sender.sourceName != "gitlab.com/patopest/go-sacn" {
		t.Errorf("Default SourceName not set correctly, got: %s", sender.sourceName)
	}
}

func TestNewSenderSourceNameTooLong(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "This is a very long source name that exceeds the maximum allowed length of 64 characters for sACN packets",
	}

	_, err := NewSender("127.0.0.1", options)
	if err == nil {
		t.Fatalf("Expected error for source name too long, got nil")
	}
}

func TestSenderClose(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}

	// Close without starting any universe
	sender.Close()
}

func TestSenderStartUniverse(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	// Test valid universe
	ch, err := sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}
	if ch == nil {
		t.Fatalf("Channel is nil")
	}

	if !sender.IsEnabled(1) {
		t.Errorf("Universe 1 should be enabled")
	}

	// Test starting same universe again
	_, err = sender.StartUniverse(1)
	if err == nil {
		t.Errorf("Expected error when starting same universe twice")
	}

	// Test invalid universe (0)
	_, err = sender.StartUniverse(0)
	if err == nil {
		t.Errorf("Expected error for universe 0")
	}

	// Test invalid universe (64000)
	_, err = sender.StartUniverse(64000)
	if err == nil {
		t.Errorf("Expected error for universe 64000")
	}
}

func TestSenderStopUniverse(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	_, err = sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	err = sender.StopUniverse(1)
	if err != nil {
		t.Fatalf("Failed to stop universe: %v", err)
	}

	// Give some time for the goroutine to clean up
	time.Sleep(100 * time.Millisecond)

	if sender.IsEnabled(1) {
		t.Errorf("Universe 1 should not be enabled after StopUniverse")
	}

	// Test stopping non-existent universe
	err = sender.StopUniverse(999)
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}
}

func TestSenderSend(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	_, err = sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	p := packet.NewDataPacket()
	p.SetData([]byte{255, 255, 255})

	err = sender.Send(1, p)
	if err != nil {
		t.Fatalf("Failed to send packet: %v", err)
	}

	// Test sending to non-existent universe
	err = sender.Send(999, p)
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}
}

func TestSenderGetUniverses(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	// Initially no universes
	universes := sender.GetUniverses()
	if len(universes) != 0 {
		t.Errorf("Expected 0 universes, got %d", len(universes))
	}

	// Start some universes
	_, err = sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe 1: %v", err)
	}
	_, err = sender.StartUniverse(2)
	if err != nil {
		t.Fatalf("Failed to start universe 2: %v", err)
	}

	universes = sender.GetUniverses()
	if len(universes) != 2 {
		t.Errorf("Expected 2 universes, got %d", len(universes))
	}
}

func TestSenderMulticast(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	_, err = sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	// Test multicast (should be off by default)
	multicast, err := sender.IsMulticast(1)
	if err != nil {
		t.Fatalf("IsMulticast failed: %v", err)
	}
	if multicast {
		t.Errorf("Multicast should be off by default")
	}

	// Enable multicast
	err = sender.SetMulticast(1, true)
	if err != nil {
		t.Fatalf("SetMulticast failed: %v", err)
	}

	multicast, err = sender.IsMulticast(1)
	if err != nil {
		t.Fatalf("IsMulticast failed: %v", err)
	}
	if !multicast {
		t.Errorf("Multicast should be enabled")
	}

	// Test on non-existent universe
	_, err = sender.IsMulticast(999)
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}

	err = sender.SetMulticast(999, true)
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}
}

func TestSenderDestinations(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	_, err = sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	// Initially no destinations
	dests, err := sender.GetDestinations(1)
	if err != nil {
		t.Fatalf("GetDestinations failed: %v", err)
	}
	if len(dests) != 0 {
		t.Errorf("Expected 0 destinations, got %d", len(dests))
	}

	// Add a destination
	err = sender.AddDestination(1, "127.0.0.1")
	if err != nil {
		t.Fatalf("AddDestination failed: %v", err)
	}

	dests, err = sender.GetDestinations(1)
	if err != nil {
		t.Fatalf("GetDestinations failed: %v", err)
	}
	if len(dests) != 1 {
		t.Errorf("Expected 1 destination, got %d", len(dests))
	}

	// Add another destination
	err = sender.AddDestination(1, "192.168.1.1")
	if err != nil {
		t.Fatalf("AddDestination failed: %v", err)
	}

	dests, err = sender.GetDestinations(1)
	if err != nil {
		t.Fatalf("GetDestinations failed: %v", err)
	}
	if len(dests) != 2 {
		t.Errorf("Expected 2 destinations, got %d", len(dests))
	}

	// Invalid address
	err = sender.AddDestination(1, "not-a-valid-ip")
	if err == nil {
		t.Errorf("Expected error for invalid IP address")
	}

	// Test on non-existent universe
	_, err = sender.GetDestinations(999)
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}

	err = sender.AddDestination(999, "127.0.0.1")
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}
}

func TestSenderSetDestinations(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	_, err = sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	// Set multiple destinations at once
	err = sender.SetDestinations(1, []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"})
	if err != nil {
		t.Fatalf("SetDestinations failed: %v", err)
	}

	dests, err := sender.GetDestinations(1)
	if err != nil {
		t.Fatalf("GetDestinations failed: %v", err)
	}
	if len(dests) != 3 {
		t.Errorf("Expected 3 destinations, got %d", len(dests))
	}

	// Set new destinations (should replace old ones)
	err = sender.SetDestinations(1, []string{"172.16.0.1"})
	if err != nil {
		t.Fatalf("SetDestinations failed: %v", err)
	}

	dests, err = sender.GetDestinations(1)
	if err != nil {
		t.Fatalf("GetDestinations failed: %v", err)
	}
	if len(dests) != 1 {
		t.Errorf("Expected 1 destination after SetDestinations, got %d", len(dests))
	}

	// Invalid address in list
	err = sender.SetDestinations(1, []string{"127.0.0.1", "not-a-valid-ip"})
	if err == nil {
		t.Errorf("Expected error for invalid IP address in list")
	}

	// Test on non-existent universe
	err = sender.SetDestinations(999, []string{"127.0.0.1"})
	if err != universeNotFoundError {
		t.Errorf("Expected universeNotFoundError, got: %v", err)
	}
}

func TestSenderSendLoopDataPacket(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	ch, err := sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	p := packet.NewDataPacket()
	p.SetData([]byte{255, 255, 255})
	p.CID = [16]byte{} // Empty CID should be filled by sender

	ch <- p

	// Give some time for packet to be processed
	time.Sleep(50 * time.Millisecond)
}

func TestSenderSendLoopSyncPacket(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	ch, err := sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	p := packet.NewSyncPacket()
	p.CID = [16]byte{} // Empty CID should be filled by sender

	ch <- p

	// Give some time for packet to be processed
	time.Sleep(50 * time.Millisecond)
}

func TestSenderSendLoopDiscoveryPacket(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	ch, err := sender.StartUniverse(1)
	if err != nil {
		t.Fatalf("Failed to start universe: %v", err)
	}

	// Send a discovery packet (technically this shouldn't happen in normal operation)
	p := packet.NewDiscoveryPacket()
	p.CID = [16]byte{}          // Empty CID should be filled by sender
	p.SetSourceName("")         // Empty source name should be filled by sender
	p.SetUniverses([]uint16{1}) // Add a universe

	ch <- p

	// Give some time for packet to be processed
	time.Sleep(50 * time.Millisecond)
}

func TestSenderNewSenderInvalidAddress(t *testing.T) {
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Test Source",
	}

	// Try to create sender with invalid address
	_, err := NewSender("invalid-address", options)
	if err == nil {
		t.Fatalf("Expected error for invalid address")
	}
}
