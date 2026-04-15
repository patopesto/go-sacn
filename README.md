# go-sacn

[![documentation](https://pkg.go.dev/badge/gitlab.com/patopest/go-sacn.svg)](https://pkg.go.dev/gitlab.com/patopest/go-sacn)
[![lib version](https://img.shields.io/gitlab/v/tag/patopest%2Fgo-sacn?label=latest)](https://gitlab.com/patopest/go-sacn/-/tags)
[![report](https://goreportcard.com/badge/gitlab.com/patopest/go-sacn)](https://goreportcard.com/report/gitlab.com/patopest/go-sacn)
![coverage](https://gitlab.com/patopest/go-sacn/badges/master/coverage.svg)
[![license](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

A pure Go library for sACN (ANSI E1.31), the standard for DMX512 lighting control data over IP networks.

## Overview

sACN (Streaming ACN) is a standard protocol defined by ANSI E1.31 for transporting DMX512 data over TCP/IP networks. It enables lighting control data to be sent over standard Ethernet infrastructure, allowing for large-scale lighting installations to be controlled remotely without dedicated DMX cables.

This library provides a complete implementation of the sACN specification in pure Go, including:

- **Full specification compliance** - All packet types (Data, Sync, Discovery) as defined in ANSI E1.31-2018
- **Sender** - Transmit DMX data on one or more universes with multicast and unicast support
- **Receiver** - Listen for sACN data with callback-based packet handling and stream termination detection
- **Universe discovery** - Automatically advertise and detect available sources and universes on the network

---

## Installation

```shell
go get gitlab.com/patopest/go-sacn
```

## Quick Start

### Creating a Receiver

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
    if !ok {
        return
    }
    data := d.GetData()
    fmt.Printf("Received Data Packet for universe %d from %s: %v ...\n", d.Universe, info.Source.IP.String(), data[:min(10, len(data))])
}
```

### Creating a Sender

```go
package main

import (
    "log"
    "time"

    "gitlab.com/patopest/go-sacn"
    "gitlab.com/patopest/go-sacn/packet"
)

func main() {
    opts := sacn.SenderOptions{
        SourceName: "go-sacn test source",
    }
    sender, err := sacn.NewSender("192.168.1.200", &opts) // change IP to interface you want to send from
    if err != nil {
        log.Fatal(err)
    }

    universe, err := sender.StartUniverse(123)
    if err != nil {
        log.Fatal(err)
    }
    sender.SetMulticast(123, true)

    p := packet.NewDataPacket()
    p.SetData([]uint8{1, 2, 3, 4})
    sender.Send(123, p)

    sender.StopUniverse(123)

    time.Sleep(1 * time.Second)
    sender.Close()
}
```

## Examples

See the [examples](./examples) directory for complete example programs:

| Example | Description |
|---------|-------------|
| [sender](./examples/sender/) | Basic sender transmitting DMX data on a single universe |
| [receiver](./examples/receiver/) | Basic receiver listening for DMX data on a single universe |
| [synced-sender](./examples/synced-sender/) | Sender synchronizing multiple universes using sync packets |
| [synced-receiver](./examples/synced-receiver/) | Receiver handling synchronized universes |
| [discovery](./examples/discovery/) | Receiver listening for universe discovery announcements |

Run an example with [go-task](https://github.com/go-task/task):

```shell
task run EXAMPLE=sender
```

---

## Key Concepts

### Universes

A universe is a collection of up to 512 DMX channels. In sACN, universes are identified by a 16-bit number (1-63999). Each universe has a multicast address in the 239.255.x.x range calculated from the universe number.

### Packet Types

- **Data Packets** - Contains the actual DMX512 lighting control data
- **Sync Packets** - Used to synchronize universes data processing or output on receivers.
- **Discovery Packets** - Used to announce available universes on the network

### Multicast vs Unicast

By default, data is sent using multicast to the appropriate multicast address for each universe. Alternatively, you can configure specific unicast destinations for point-to-point communication.

---

## API Reference

### Receiver

```go
// Create a new receiver bound to a network interface
receiver := sacn.NewReceiver(itf)

// Join a universe to start receiving data
receiver.JoinUniverse(1)

// Register callbacks for packet types
receiver.RegisterPacketCallback(packet.PacketTypeData, dataCallback)
receiver.RegisterPacketCallback(packet.PacketTypeSync, syncCallback)
receiver.RegisterPacketCallback(packet.PacketTypeDiscovery, discoveryCallback)

// Register callback for universe stream termination
receiver.RegisterTerminationCallback(terminationCallback)

// Start the receiver
receiver.Start()

// Stop the receiver
receiver.Stop()
```

### Sender

```go
// Create a sender bound to an IP address (required for multicast)
sender, err := sacn.NewSender("192.168.1.100", &sacn.SenderOptions{
    SourceName: "My Application",
})

// Start a universe and get a channel for sending packets
ch, err := sender.StartUniverse(1)

// Enable multicast for a universe
sender.SetMulticast(1, true)

// Add unicast destinations
sender.AddDestination(1, "192.168.1.200")

// Send packets via channel or direct send
ch <- dataPacket
sender.Send(1, dataPacket)

// Stop a universe (sends stream termination notification)
sender.StopUniverse(1)

// Clean up
sender.Close()
```

### Data Packets

```go
p := packet.NewDataPacket()
p.SetData([]uint8{1, 2, 3, 4})           // Set up to 512 bytes of DMX data
p.SetSourceName("My Controller")         // Optional: override default source name
p.SetPriority(100)                       // Optional: set priority (default 100)
p.SetPreviewData(true)                   // Optional: preview data flag
p.SetStreamTerminated(true)              // Optional: signal stream termination
```

## Development

### Running Tests

```shell
task tests

# With coverage report
task coverage
```

### Code Quality

```shell
task check
```

This formats, lints, and type-checks the code.


## Related Projects

- [sACN-Monitor](https://gitlab.com/patopest/sacn-monitor) - A desktop application for viewing sACN and ArtNet data from all universes, built using this library.
- Hundemeier's [go-sacn](https://github.com/Hundemeier/go-sacn): only supports Data packets
- [go-artnet](https://github.com/jsimonetti/go-artnet)
- Open Lighting Architecure (OLA) [framework](https://github.com/OpenLightingProject/ola) (C implementations of control protocols).

## References

- sACN (ANSI E1.31) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI_E1-31-2018.pdf)
- DMX512 (ANSI E1.11) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI-ESTA_E1-11_2008R2018.pdf)