# sACN in go

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/patopest/go-sacn.svg)](https://pkg.go.dev/gitlab.com/patopest/go-sacn)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/patopest/go-sacn)](https://goreportcard.com/report/gitlab.com/patopest/go-sacn)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
![Go version](https://img.shields.io/gitlab/go-mod/go-version/patopest/go-sacn)
![GitLab Tag](https://img.shields.io/gitlab/v/tag/patopest%2Fgo-sacn)

A library for sACN (ANSI E1.31) in `Go`.

Fully supports and complies to the specification:

- All Packet types (Data, Sync and Discovery).
- Receiver with callbacks and stream termination detection.
- Transmitter sending discovery packets.


## Usage

```shell
go get gitlab.com/patopest/go-sacn
```

- Receiver

```go
package main

import (
    "fmt"
    "time"
    "net"

    "gitlab.com/patopest/go-sacn"
    "gitlab.com/patopest/go-sacn/packet"
)

func main() {
    fmt.Println("hello")

    itf, _ := net.InterfaceByName("en0") // change based on your machine
    receiver := sacn.NewReceiver(itf)
    receiver.JoinUniverse(1)
    receiver.RegisterPacketCallback(packet.PacketTypeData, dataPacketCallback)
    receiver.Start()

    for {
        time.Sleep(1)
    }
}

func dataPacketCallback(p packet.SACNPacket, info sacn.PacketInfo) {
    d, ok := p.(*packet.DataPacket)
    if ok == false {
        return
    }
    fmt.Printf("Received Data Packet for universe %d from %s\n", d.Universe, info.Source.IP.String())
}
```

- Transmitter

```go
package main

import (
    "log"
    "time"

    "gitlab.com/patopest/go-sacn"
    "gitlab.com/patopest/go-sacn/packet"
)

func main() {
    log.Println("Hello")

    opts := sacn.SenderOptions{ // Default for all packets sent by Sender if not provided in the packet itself.
        SourceName: "go-sacn test source"
    }
    sender, err := sacn.NewSender("192.168.1.200", &opts) // Create sender with binding to interface
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

    // Create new packet and fill it up with data
    p := packet.NewDataPacket()
    p.SetData([]uint8{1, 2, 3, 4})

    sender.Send(uni, p) // send the packet

    sender.StopUniverse(uni) // To stop the universe and advertise termination to receivers

    time.Sleep(1 * time.Second)

    sender.Close() // Close sender and all universes
}
```


See [examples](./examples) directory for more examples.


## Development

- Run an example to test your code

```shell
go run examples/receiver/receiver.go
```

- Tests

```shell
go test ./...
```

- Docs

```shell
go run golang.org/x/pkgsite/cmd/pkgsite@latest -open
```


## References

### Similar projects

- [sACN-Monitor](https://gitlab.com/patopest/sacn-monitor): An app to view data from all sACN universes, built using this library.
- Hundemeier's [go-sacn](https://github.com/Hundemeier/go-sacn): Only supports Data packets
- [go-artnet](https://github.com/jsimonetti/go-artnet)
- Open Lighting Architecure (OLA) [framework](https://github.com/OpenLightingProject/ola) (C implementations of control protocols).

### Documentation

- sACN (ANSI E1.31) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI_E1-31-2018.pdf).
- DMX (ANSI E1.11) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI-ESTA_E1-11_2008R2018.pdf)