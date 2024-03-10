package main

import (
	"fmt"
	"net"
	"time"

	"gitlab.com/patopest/go-sacn"
	"gitlab.com/patopest/go-sacn/packet"
)

func main() {
	fmt.Println("hello")

	itf, _ := net.InterfaceByName("en0") // specific to your machine
	receiver := sacn.NewReceiver(itf)
	receiver.JoinUniverse(1)
	receiver.RegisterPacketCallback(packet.PacketTypeData, dataPacketCallback)
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

func universeTerminatedCallback(universe uint16) {
	fmt.Printf("Universe %d is terminated", universe)
}
