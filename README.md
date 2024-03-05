# sACN in go

A library for sACN (ANSI E1.31) in `Go`.

Fully supports and complies to the specification:

- All Packet types (Data, Sync and Discovery)
- Receiver with callbacks on sync packet reception
- Transmitter sending discovery packets


## TODO

TODO

## References

### Similar projects

- Hundemeier's [go-sacn](https://github.com/Hundemeier/go-sacn): Only supports Data packets
- [go-artnet](https://github.com/jsimonetti/go-artnet)

### Documentation

- sACN (ANSI E1.31) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI_E1-31-2018.pdf).
- DMX (ANSI E1.11) [specification](https://tsp.esta.org/tsp/documents/docs/ANSI-ESTA_E1-11_2008R2018.pdf)
- Open Lighting Architecure (OLA) [framework](https://github.com/OpenLightingProject/ola) (low-level implementations of control protocols).