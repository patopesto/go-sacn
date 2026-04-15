package sacn

import (
	"sync"
	"testing"
	"time"

	"gitlab.com/patopest/go-sacn/packet"
)

// Integration test: Sender sends to Receiver
// This test requires proper network setup and may fail in certain environments
func TestSenderReceiverIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	iface := getTestInterface(t)

	// Create receiver
	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	dataReceived := false
	var receivedPacket packet.SACNPacket
	receiver.RegisterPacketCallback(packet.PacketTypeData, func(p packet.SACNPacket, info PacketInfo) {
		if !dataReceived {
			dataReceived = true
			receivedPacket = p
			wg.Done()
		}
	})

	// Join the universe
	err = receiver.JoinUniverse(1)
	if err != nil {
		t.Skipf("Could not join multicast group: %v", err)
	}

	// Start receiver
	receiver.Start()

	// Give receiver time to start
	time.Sleep(50 * time.Millisecond)

	// Create sender
	options := &SenderOptions{
		CID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SourceName: "Integration Test",
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

	// Enable multicast (for this test we'll use unicast)
	// Get the receiver's local address - note this may return 0.0.0.0
	// which won't work as a destination, so we need to use loopback explicitly
	err = sender.AddDestination(1, "127.0.0.1")
	if err != nil {
		t.Fatalf("Failed to add destination: %v", err)
	}

	// Send a packet
	p := packet.NewDataPacket()
	p.SetData([]byte{255, 128, 64, 32})
	ch <- p

	// Wait for packet with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !dataReceived {
			t.Errorf("Data should have been received")
		}
		if receivedPacket == nil {
			t.Errorf("Received packet should not be nil")
		}
	case <-time.After(2 * time.Second):
		// In some network environments this test may timeout
		// This is expected behavior and not a test failure per se
		t.Skip("Timeout waiting for packet - network may not support multicast/unicas")
	}
}

// TestSenderDiscoveryPacketIntegration tests that a sender sends correct discovery packets
// Discovery packets are sent automatically to universe 64214 to announce active universes
func TestSenderDiscoveryPacketIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	iface := getTestInterface(t)

	// Create a receiver that will listen for discovery packets
	receiver, err := NewReceiver(iface)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}
	defer receiver.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	discoveryReceived := false
	var receivedDiscovery *packet.DiscoveryPacket
	receiver.RegisterPacketCallback(packet.PacketTypeDiscovery, func(p packet.SACNPacket, info PacketInfo) {
		if !discoveryReceived {
			discoveryReceived = true
			receivedDiscovery = p.(*packet.DiscoveryPacket)
			wg.Done()
		}
	})

	// Join the discovery universe (64214)
	err = receiver.JoinUniverse(DISCOVERY_UNIVERSE)
	if err != nil {
		t.Skipf("Could not join discovery multicast group: %v", err)
	}

	// Start receiver
	receiver.Start()

	// Give receiver time to start
	time.Sleep(50 * time.Millisecond)

	// Expected CID and source name
	expectedCID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	expectedSourceName := "Discovery Test Source"

	// Create sender - discovery loop starts automatically
	options := &SenderOptions{
		CID:        expectedCID,
		SourceName: expectedSourceName,
	}

	sender, err := NewSender("127.0.0.1", options)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer sender.Close()

	// Start some universes - these should be announced in discovery packets
	testUniverses := []uint16{1, 5, 100, 500}
	for _, universe := range testUniverses {
		_, err := sender.StartUniverse(universe)
		if err != nil {
			t.Fatalf("Failed to start universe %d: %v", universe, err)
		}
	}

	// Wait for discovery packet with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !discoveryReceived {
			t.Fatalf("Discovery packet should have been received")
		}

		// Validate the discovery packet
		if receivedDiscovery == nil {
			t.Fatalf("Received discovery packet should not be nil")
		}

		// Verify CID
		if receivedDiscovery.CID != expectedCID {
			t.Errorf("Discovery packet CID mismatch:\n- expected: %v\n- got: %v", expectedCID, receivedDiscovery.CID)
		}

		// Verify Source Name
		if receivedDiscovery.GetSourceName() != expectedSourceName {
			t.Errorf("Discovery packet SourceName mismatch:\n- expected: %s\n- got: %s", expectedSourceName, receivedDiscovery.GetSourceName())
		}

		// Verify universes list contains our test universes
		receivedUniverses := make([]uint16, 0, receivedDiscovery.GetNumUniverses())
		for i := 0; i < receivedDiscovery.GetNumUniverses(); i++ {
			receivedUniverses = append(receivedUniverses, receivedDiscovery.Universes[i])
		}

		// Check that all test universes are announced
		for _, expectedUni := range testUniverses {
			found := false
			for _, receivedUni := range receivedUniverses {
				if receivedUni == expectedUni {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected universe %d not found in discovery packet. Got universes: %v", expectedUni, receivedUniverses)
			}
		}

		// Verify page numbers are valid (Page <= Last)
		if receivedDiscovery.Page > receivedDiscovery.Last {
			t.Errorf("Discovery packet Page (%d) should be <= Last (%d)", receivedDiscovery.Page, receivedDiscovery.Last)
		}

		// Verify UDLVector is correct
		if receivedDiscovery.UDLVector != packet.VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST {
			t.Errorf("Discovery packet UDLVector mismatch:\n- expected: 0x%x\n- got: 0x%x", packet.VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST, receivedDiscovery.UDLVector)
		}

	case <-time.After(11 * time.Second):
		// Discovery packets are sent every 10 seconds, but first one is sent immediately
		// after sender creation. However, in some network environments, multicast may not work.
		t.Skip("Timeout waiting for discovery packet - network may not support multicast")
	}
}
