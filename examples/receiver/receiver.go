package main

import (
	"fmt"
	"net"
	"time"

	"gitlab.com/patopest/go-sacn"
	"gitlab.com/patopest/go-sacn/packet"
)

func main() {
	fmt.Println("Hello from receiver")

	itf, _ := net.InterfaceByName("en0") // specific to your machine
	receiver, err := sacn.NewReceiver(itf)
	if err != nil {
		panic(err)
	}
	// Setup universe reception
	receiver.JoinUniverse(1)
	receiver.RegisterPacketCallback(packet.PacketTypeData, dataPacketCallback)
	receiver.RegisterTerminationCallback(universeTerminatedCallback)
	receiver.Start()

	fmt.Println("Receiver started. Waiting for data...")
	for {
		time.Sleep(1)
	}
}

func dataPacketCallback(p packet.SACNPacket, info sacn.PacketInfo) {
	d, ok := p.(*packet.DataPacket)
	if ok == false {
		return
	}

	data := d.GetData()
	fmt.Printf("Received Data Packet for universe %d from %s: %v ...\n", d.Universe, info.Source.IP.String(), data[:min(10, len(data))])
}

func universeTerminatedCallback(universe uint16) {
	fmt.Printf("Universe %d terminated\n", universe)
}
