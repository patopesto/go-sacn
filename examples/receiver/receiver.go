package main

import (
    "fmt"
    "time"
    "net"

    "github.com/patopesto/go-sacn"
    "github.com/patopesto/go-sacn/packet"
)

func main() {
    fmt.Println("hello")

    itf, _ := net.InterfaceByName("en0")
    receiver := sacn.NewReceiver(itf)
    receiver.JoinUniverse(1)
    receiver.RegisterPacketCallback(packet.PacketTypeData, dataPacketCallback)
    receiver.RegisterPacketCallback(packet.PacketTypeSync, syncPacketCallback)
    receiver.RegisterPacketCallback(packet.PacketTypeDiscovery, discoveryPacketCallback)
    receiver.RegisterTerminationCallback(universeTerminatedCallback)
    receiver.Start()

    for {
    	time.Sleep(1)
    }
}

func dataPacketCallback(p packet.SACNPacket) {
    d, ok := p.(*packet.DataPacket)
    if ok == false {
        return
    }
    fmt.Printf("Received Data Packet for universe %d\n", d.Universe)
}

func syncPacketCallback(p packet.SACNPacket) {
    s, ok := p.(*packet.SyncPacket)
    if ok == false {
        return
    }
    fmt.Printf("Received Sync Packet with address %d\n", s.SyncAddress)
}

func discoveryPacketCallback(p packet.SACNPacket) {
    d, ok := p.(*packet.DiscoveryPacket)
    if ok == false {
        return
    }
    fmt.Printf("Received Discovery Packet with page %d\n", d.Page)
}

func universeTerminatedCallback(universe uint16) {
    fmt.Printf("Universe %d is terminated", universe);
}