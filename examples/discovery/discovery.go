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
	receiver, err := sacn.NewReceiver(itf)
	if err != nil {
		panic(err)
	}
	receiver.JoinUniverse(sacn.DISCOVERY_UNIVERSE)
	receiver.RegisterPacketCallback(packet.PacketTypeDiscovery, discoveryPacketCallback)
	receiver.Start()

	for {
		time.Sleep(1)
	}
}

func discoveryPacketCallback(p packet.SACNPacket, info sacn.PacketInfo) {
	d, ok := p.(*packet.DiscoveryPacket)
	if ok == false {
		return
	}

	fmt.Printf("Discovered universes from %s:\n", string(d.SourceName[:]))
	for i := 0; i < d.GetNumUniverses(); i++ {
		fmt.Printf("%d, ", d.Universes[i])
	}
}
