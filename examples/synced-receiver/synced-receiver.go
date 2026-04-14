package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"gitlab.com/patopest/go-sacn"
	"gitlab.com/patopest/go-sacn/packet"
)

type syncGroup struct {
	universes map[uint16][]byte
	mu        sync.Mutex
}

var syncGroups = make(map[uint16]*syncGroup)
var mu sync.Mutex

func main() {
	fmt.Println("Hello from synced-receiver")

	itf, _ := net.InterfaceByName("en0")
	receiver, err := sacn.NewReceiver(itf)
	if err != nil {
		panic(err)
	}

	// Setup universes reception
	unis := []uint16{1, 2, 3}
	for _, u := range unis {
		receiver.JoinUniverse(u)
	}
	receiver.RegisterPacketCallback(packet.PacketTypeData, dataPacketCallback)
	receiver.RegisterPacketCallback(packet.PacketTypeSync, syncPacketCallback)
	receiver.RegisterTerminationCallback(universeTerminatedCallback)
	receiver.Start()

	fmt.Println("Receiver started. Waiting for data...")
	for {
		time.Sleep(1)
	}
}

func dataPacketCallback(p packet.SACNPacket, info sacn.PacketInfo) {
	d, ok := p.(*packet.DataPacket)
	if !ok {
		return
	}

	// Skipping universes which do not have sync enabled
	if d.SyncAddress == 0 {
		return
	}

	mu.Lock()
	sg, exists := syncGroups[d.SyncAddress]
	if !exists {
		sg = &syncGroup{universes: make(map[uint16][]byte)}
		syncGroups[d.SyncAddress] = sg
	}
	sg.mu.Lock()
	sg.universes[d.Universe] = d.GetData()
	sg.mu.Unlock()
	mu.Unlock()
}

func syncPacketCallback(p packet.SACNPacket, info sacn.PacketInfo) {
	s, ok := p.(*packet.SyncPacket)
	if !ok {
		return
	}

	mu.Lock()
	sg, exists := syncGroups[s.SyncAddress]
	mu.Unlock()

	if !exists {
		return
	}

	sg.mu.Lock()
	fmt.Printf("Received Sync %d:\n", s.SyncAddress)
	for uni, data := range sg.universes {
		fmt.Printf("  Universe %d: %v\n", uni, data[:min(10, len(data))])
	}
	sg.mu.Unlock()
}

func universeTerminatedCallback(universe uint16) {
	fmt.Printf("Universe %d terminated\n", universe)
	mu.Lock()
	for _, sg := range syncGroups {
		sg.mu.Lock()
		delete(sg.universes, universe)
		sg.mu.Unlock()
	}
	mu.Unlock()
}
