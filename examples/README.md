# Examples

This directory contains example programs demonstrating how to use the go-sacn library.

## Overview

| Example                             | Description                                                         |
|-------------------------------------|---------------------------------------------------------------------|
| [sender](sender/)                   | Basic sACN sender that transmits DMX data on a single universe      |
| [receiver](receiver/)               | Basic sACN receiver that listens for DMX data on a single universe  |
| [synced-sender](synced-sender/)     | Sender that synchronizes multiple universes using sACN sync packets |
| [synced-receiver](synced-receiver/) | Receiver that handles synchronized universes                        |
| [discovery](discovery/)             | Receiver that listens for universe discovery announcements          |


## Running the Examples

### Network Interface Configuration

Most examples use a specific network interface (e.g., `"en0"`). You need to change this to match your system's network interface:

- **macOS**: Use `en0`, `en1`, etc. (run `ifconfig` to list interfaces)
- **Linux**: Use `eth0`, `wlan0`, etc. (run `ip link` or `ifconfig` to list interfaces)
- **Windows**: Use the interface name or index

Alternatively, you can pass `nil` to use the default interface:

```go
receiver, err := sacn.NewReceiver(nil)
```

### Basic

Sends DMX data on universe 1:

```bash
cd sender
go run sender.go
```

Receives DMX data on universe 1:

```bash
cd receiver
go run receiver.go
```

### Synced

Sends synchronized DMX data on multiple universes (1, 2, 3) with a sync address of 100:

```bash
cd synced-sender
go run synced-sender.go
```

Receives synchronized DMX data. The receiver listens on universes 1, 2, 3 and outputs them together when a sync packet is received on sync address 100:

```bash
cd synced-receiver
go run synced-receiver.go
```

### Discovery

Listens for universe discovery announcements to find which universes are being transmitted on the network:

```bash
cd discovery
go run discovery.go
```


## Common Patterns

### Creating a Sender

```go
sender, err := sacn.NewSender("192.168.1.100", &sacn.SenderOptions{
    SourceName: "My Application",
})
if err != nil {
    log.Fatal(err)
}
defer sender.Close()

ch, err := sender.StartUniverse(1)
if err != nil {
    log.Fatal(err)
}

// Send data
p := packet.NewDataPacket()
p.SetData([]byte{255, 128, 64})
ch <- p
```

### Creating a Receiver

```go
itf, _ := net.InterfaceByName("en0")
receiver, err := sacn.NewReceiver(itf)
if err != nil {
    log.Fatal(err)
}

receiver.JoinUniverse(1)
receiver.RegisterPacketCallback(packet.PacketTypeData, func(p packet.SACNPacket, info sacn.PacketInfo) {
    d := p.(*packet.DataPacket)
    fmt.Printf("Received data for universe %d\n", d.Universe)
})
receiver.Start()
```

### Adding Unicast Destinations

```go
sender.AddDestination(1, "192.168.1.200")
```

### Enabling Multicast

```go
sender.SetMulticast(1, true)
```


## Notes

- The sACN port is 5568 (defined by the ANSI E1.31 standard)
- Universe numbers range from 1 to 63999
- The discovery universe is 64214