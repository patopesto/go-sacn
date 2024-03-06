package packet

import (
    "encoding/binary"
    "errors"
)


type SyncPacket struct {
    // Inherit RootLayer
    RootLayer

    // Framing layer
    FrameLength uint16
    FrameVector uint32
    Sequence    uint8
    SyncAddress uint16
    reserved    [2]byte
}

func NewSyncPacket() *SyncPacket {
    // TODO: fill up default values
    return &SyncPacket{}
}

func (d *SyncPacket) GetType() SACNPacketType {
    return PacketTypeSync
}

func (d *SyncPacket) UnmarshalBinary(b []byte) error {
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
    d.Sequence = b[44]
    d.SyncAddress = binary.BigEndian.Uint16(b[45:47])

    return d.validate()
}

func (d *SyncPacket) validate() error {
    // Root layer (specifics to SyncPacket)
    if d.RootVector != VECTOR_ROOT_E131_EXTENDED {
        return errors.New("Invalid Root Vector")
    }

    // Framing layer
    if d.FrameVector != VECTOR_E131_EXTENDED_SYNCHRONIZATION {
        return errors.New("Invalid Frame Vector")
    }
    
    return nil
}
