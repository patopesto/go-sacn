# sACN in go

A library for sACN (ANSI E1.31) in `Go`.

Fully supports and complies to the specification:

- All Packet types (Data, Sync and Discovery)
- Receiver with callbacks on sync packet reception
- Transmitter sending discovery packets


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

func dataPacketCallback(p packet.SACNPacket) {
    d, ok := p.(*packet.DataPacket)
    if ok == false {
        return
    }
    fmt.Printf("Received Data Packet for universe %d\n", d.Universe)
}
```

- Transmitter

TODO


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

## References

### Similar projects

- [sACN-Monitor](https://gitlab.com/patopest/sacn-monitor): An app to view data from all sACN universes, built using this library.
- Hundemeier's [go-sacn](https://github.com/Hundemeier/go-sacn): Only supports Data packets
- [go-artnet](https://github.com/jsimonetti/go-artnet)
- Open Lighting Architecure (OLA) [framework](https://github.com/OpenLightingProject/ola) (C implementations of control protocols).

### Documentation

- sACN (ANSI E1.31) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI_E1-31-2018.pdf).
- DMX (ANSI E1.11) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI-ESTA_E1-11_2008R2018.pdf)