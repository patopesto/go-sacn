package main

import (
	"fmt"
	"log"
	"time"

	"gitlab.com/patopest/go-sacn"
	"gitlab.com/patopest/go-sacn/packet"
)

const syncAddress = 100

func main() {
	fmt.Println("Hello from synced-sender")

	sender, err := sacn.NewSender("192.168.1.200", &sacn.SenderOptions{
		SourceName: "synced-sender",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sender.Close()

	unis := []uint16{1, 2, 3}
	dataChans := make(map[uint16]chan<- packet.SACNPacket)
	for _, uni := range unis {
		ch, err := sender.StartUniverse(uni)
		if err != nil {
			log.Fatal(err)
		}
		sender.SetMulticast(uni, true)
		dataChans[uni] = ch
	}

	syncCh, err := sender.StartUniverse(syncAddress)
	if err != nil {
		log.Fatal(err)
	}
	sender.SetMulticast(syncAddress, true)

	for i := 0; i < 10; i++ {
		log.Println("Sending packets")
		// Send data on the universes
		for _, uni := range unis {
			p := packet.NewDataPacket()
			p.SetData([]uint8{1, 2, 3, 4})
			p.SyncAddress = syncAddress

			dataChans[uni] <- p // send the packet
		}

		// Send the sync packet after having sent all the data
		syncP := packet.NewSyncPacket()
		syncP.SyncAddress = syncAddress
		syncCh <- syncP

		time.Sleep(1 * time.Second)
	}
}
