package main

import (
	"log"
	"time"

	"gitlab.com/patopest/go-sacn"
	"gitlab.com/patopest/go-sacn/packet"
)

func main() {
	log.Println("Hello")

	sender, err := sacn.NewSender("192.168.1.200", &sacn.SenderOptions{}) // Create sender
	if err != nil {
		log.Fatal(err)
	}

	// Initialise universe
	var uni uint16 = 123
	universe, err := sender.StartUniverse(uni)
	if err != nil {
		log.Fatal(err)
	}
	sender.SetMulticast(uni, true)
	// sender.AddDestination(uni, "192.168.1.200")
	// sender.AddDestination(uni, "192.168.1.115")

	// Create new packet and fill it up with data
	p := packet.NewDataPacket()
	p.SetData([]uint8{1, 2, 3, 4})
	log.Println("Sending packet")

	for i := 0; i < 10; i++ {
		universe <- p // send the packet

		time.Sleep(1 * time.Second)
	}

	// To stop the universe and advertise termination to receivers
	close(universe)

	time.Sleep(1 * time.Second)

	// To close the sender altogether
	sender.Close()
}
