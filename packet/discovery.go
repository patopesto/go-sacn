package packet

import (
    // "bytes"
    // "encoding"
    "encoding/binary"
    "errors"
)

// var _ SACNPacket = &DataPacket{}

type DiscoveryPacket struct {
    // Inherit RootLayer
    RootLayer

    // Framing layer
    FrameLength uint16
    FrameVector uint32
    SourceName  [64]byte
    reserved    [4]byte

    // Universe Discovery Layer
    UDLLength        uint16
    UDLVector        uint32
    Page             uint8
    Last             uint8
    Universes        [1024]byte
}

func NewDiscoveryPacket() *DiscoveryPacket {
    // TODO: fill up default values
    return &DiscoveryPacket{}
}

func (d *DiscoveryPacket) UnmarshalBinary(b []byte) error {
    // Root layer
    err := d.RootLayer.unmarshal(b)
    if err != nil {
        return err
    }

    // Framing layer
    d.FrameLength = binary.BigEndian.Uint16(b[38:40])
    if d.FrameLength & 0x0FFF > uint16(len(b)) {
        return errors.New("Incorrect packet size")
    }
    d.FrameVector = binary.BigEndian.Uint32(b[40:44])
    copy(d.SourceName[:], b[44:108])

    // Universe Discovery Layer
    d.UDLLength = binary.BigEndian.Uint16(b[112:114]) 
    d.UDLVector = binary.BigEndian.Uint32(b[114:118])
    d.Page = b[118]
    d.Last = b[119]
    l := d.UDLLength & 0x0FFF - 6
    copy(d.Universes[:], b[120:120+l])

    return d.validate()
}

func (d *DiscoveryPacket) validate() error {
    // Root layer (specifics to DataPacket)
    if d.RootVector != VECTOR_ROOT_E131_EXTENDED {
        return errors.New("Invalid Root Vector")
    }

    // Framing layer
    if d.FrameVector != VECTOR_E131_EXTENDED_DISCOVERY {
        return errors.New("Invalid Frame Vector")
    }

    // Universe Discovery Layerr
    if d.UDLVector != VECTOR_UNIVERSE_DISCOVERY_UNIVERSE_LIST {
        return errors.New("Invalid Discovery Vector")
    }
    if d.Page > d.Last {
        return errors.New("Current page > Last page")
    }

    return nil
}
